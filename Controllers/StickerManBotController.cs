using System.Text.Json;
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

        if (sticker == null)
            if (message.text == "/start ImOver18")
            {
                await _stickerManDbService.SetUserAgeVerified(message.from!.id);
                return new BotResponse
                {
                    chat_id = message.chat.id,
                    method = "sendMessage",
                    text = "You have verified your age"
                };
            }
            else return new BotResponse
            {
                chat_id = message.chat.id,
                method = "sendMessage",
                text = "Hi, I'm Sticker Manager Bot.\n" +
                       "I'll help you manage your stickers by letting you tag them so you can easily find them later.\n" +
                       "\n" +
                       "Usage:\n" +
                       "To add a sticker tag, first send me a sticker to this chat, then send the tags you'd like to add to the sticker.\n" +
                       "\n" +
                       "You can then easily search for tagged stickers in any chat. Just type: @StickerManBot followed by the tags of the stickers that you are looking for.\n" +
                       "For information on how to share stickers with a friend type \"/helpGroups\""
            };

        var posts = await _e621Api.GetPost(sticker.file_unique_id);

        if (posts.posts.Length == 0)
        {
            var fileResponse = await _telegramApi.GetFile(new GetFileRequest { file_id = sticker.file_id });

            await _e621Api.Upload(
                new UploadWrapper
                {
                    Upload = new Upload
                    {
                        direct_url =
                            $"https://api.telegram.org/file/bot{_config.GetValue<string>("TelegramApiToken")}/{fileResponse.result.file_path}",
                        tag_string = $"Copyright:{sticker.set_name}",
                        source = $"{sticker.file_unique_id}%0A{sticker.file_id}%0A{message.chat.id}%0A{message.chat.username}",
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

        if(!string.IsNullOrWhiteSpace(message.text))
            await _e621Api.Update(posts.posts.First().id,
                new UpdateWrapper
                {
                    Post = new Post
                    {
                        tag_string_diff = message.text
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
