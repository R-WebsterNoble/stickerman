﻿using System.Text.Json;
using Microsoft.AspNetCore.Mvc;
using StickerManBot.services;
using StickerManBot.Types.Telegram;
using Result = StickerManBot.Types.Telegram.Result;

namespace StickerManBot.Controllers;

[ApiController]
[Route("~/")]
public class StickerManBotController : Controller
{
    private readonly ILogger<StickerManBotController> _logger;
    private readonly IConfiguration _config;
    private readonly IE621Api _e621Api;
    private readonly ITelegramApi _telegramApi;
    private readonly StickerManDbService _stickerManDbService;

    public StickerManBotController(ILogger<StickerManBotController> logger, IConfiguration config, IE621Api e621Api, ITelegramApi telegramApi, StickerManDbService stickerManDbService)
    {
        _logger = logger;
        _config = config;
        _e621Api = e621Api;
        _telegramApi = telegramApi;
        _stickerManDbService = stickerManDbService;
    }

    [HttpPost]
    public async Task<IActionResult> Post([FromBody] Update update)
    {
        if (update.message?.chat == null && update.inline_query == null)
            return Ok();

        try
        {
            if (update.message != null)
                return Ok(await ProcessMessage(update.message));

            if (update.inline_query != null)
                return Ok(await ProcessInlineQuery(update.inline_query));

            throw new NotImplementedException("No handler for this update");
        }
        catch (Exception e)
        {
            _logger.LogError(e, "Error Processing Update {@Update}", update);

            return Ok(new BotResponse
            {
                chat_id = 212760070,
                method = "sendMessage",
                text = "Error:\n" +
                       e.Message +
                       "\n\n" +
                       e.StackTrace +
                       "\n\n" +
                       JsonSerializer.Serialize(update)
            });
        }
    }

    private async Task<BotResponse> ProcessMessage(Message message)
    {
        if (message.reply_to_message != null)
        {
            message.reply_to_message.text = message.text;
            message = message.reply_to_message;
        }

        var sticker = message.sticker;

        if (sticker != null)
            {
        var posts = await _e621Api.GetPost(sticker.file_unique_id);

        if (posts.posts.Length == 0)
        {
                var fileIdToGet = sticker.is_animated ? sticker.thumbnail!.file_id : sticker.file_id;
                var fileResponse = await _telegramApi.GetFile(new GetFileRequest { file_id = fileIdToGet });

                if (!fileResponse.ok)
                    return new BotResponse
                    {
                        chat_id = message.chat.id,
                        method = "sendMessage",
                        text = "Something went wrong when getting details for this sticker from Telegram"
                    };


            await _e621Api.Upload(
                new UploadWrapper
                {
                    Upload = new Upload
                    {
                        direct_url =
                            $"https://api.telegram.org/file/bot{_config.GetValue<string>("TelegramApiToken")}/{fileResponse.result.file_path}",
                            tag_string = $"Copyright:{sticker.set_name}{(sticker.is_animated ? "Meta:animated" : "")}",
                            source = sticker.file_unique_id,
                        rating = "e"
                    }
                });

            return new BotResponse
            {
                chat_id = message.chat.id,
                method = "sendMessage",
                text = "Created new Sticker"
            };
        }

            var tags = posts.posts.First().tags;
            var allTags = tags.copyright.Concat(tags.general);
            return new BotResponse
                    {
                chat_id = message.chat.id,
                method = "sendMessage",
                text = "That's a nice sticker!\n" +
                       "\n" +
                       "Here are all the existing tag(s) currently applied to that sticker:\n" +
                       string.Join('\n', allTags)
            };
                    }
                });

        return new BotResponse
        {
            chat_id = message.chat.id,
            method = "sendMessage",
            text = "Updated existing Sticker"
        };
    }
    
    private async Task<AnswerInlineQuery> ProcessInlineQuery(InlineQuery inlineQuery)
    {
        if(!await _stickerManDbService.IsUserAgeVerified(inlineQuery.from.id))
            return new AnswerInlineQueryWithButton
            {
                method = "answerInlineQuery",
                inline_query_id = inlineQuery.id,
                results = ArraySegment<Result>.Empty,
                cache_time = 0,
                is_personal = true,
                next_offset = "",
                button = new InlineQueryResultsButton
                {
                    text = "Click here if you are over the age of 18.",
                    start_parameter = "ImOver18"
                }

            };

        _ = int.TryParse(inlineQuery.offset, out var page);

        if (page == 0)
            page = 1;

        var posts = await _e621Api.GetPosts(page, $"{inlineQuery.query}*");

        var results = posts.posts//.Where(p => p.sources.Length == 3 && p.sources[2].StartsWith("https://api.telegram.org")).
        .Select(p => new Result
        {
            type = "sticker",
            id = p.id.ToString(),
            sticker_file_id = p.sources[1]
        });

        var nextPage = posts.posts.Length < 50 ? "" : (page + 1).ToString();

        return new AnswerInlineQuery
        {
            method = "answerInlineQuery",
            inline_query_id = inlineQuery.id,
            results = results,
            cache_time = 0,
            is_personal = true,
            next_offset = nextPage
        };
    }
}
