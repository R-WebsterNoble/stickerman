using System.Text.Json;
using Microsoft.AspNetCore.Mvc;
using StickerManBot.Types.e621;
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

    public StickerManBotController(ILogger<StickerManBotController> logger, IConfiguration config, IE621Api e621Api, ITelegramApi telegramApi)
    {
        _logger = logger;
        _config = config;
        _e621Api = e621Api;
        _telegramApi = telegramApi;
    }

    [HttpPost]
    public async Task<IActionResult> Post([FromBody] Update update)
    {
        if (update.message?.chat == null && update.inline_query == null)
            return Ok();

        try
        {
            if (update.inline_query != null)
                return Ok(await ProcessInlineQuery(update));

            if (update.message != null)
                return Ok(await ProcessMessage(update));

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

    private async Task<BotResponse> ProcessMessage(Update update)
    {
        var message = update.message.reply_to_message ?? update.message;

        var stickerFileId = message.sticker?.file_id;
        var stickerFileUniqueId = message.sticker?.file_unique_id;

        if (stickerFileId == null || stickerFileUniqueId == null)
            return new BotResponse
            {
                chat_id = update.message.chat.id,
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

        var posts = await _e621Api.GetPost(stickerFileUniqueId);

        if (posts.posts.Length == 0)
        {
            var fileResponse = await _telegramApi.GetFile(new GetFileRequest { file_id = stickerFileId });

            await _e621Api.Upload(
                new UploadWrapper
                {
                    Upload = new Upload
                    {
                        direct_url =
                            $"https://api.telegram.org/file/bot{_config.GetValue<string>("TelegramApiToken")}/{fileResponse.result.file_path}",
                        tag_string = update.message.text ?? "",
                        source = $"{stickerFileUniqueId}%0A{stickerFileId}",
                        rating = "e"
                    }
                });

            return new BotResponse
            {
                chat_id = update.message.chat.id,
                method = "sendMessage",
                text = "Created new Sticker"
            };
        }

        await _e621Api.Update(posts.posts.First().id,
            new UpdateWrapper
            {
                Post = new Post
                {
                    tag_string_diff = update.message.text ?? ""
                }
            });

        return new BotResponse
        {
            chat_id = update.message.chat.id,
            method = "sendMessage",
            text = "Updated existing Sticker"
        };
    }
    
    private async Task<AnswerInlineQuery> ProcessInlineQuery(Update update)
    {
        var posts = await _e621Api.GetPosts($"{update.inline_query.query}*");

        var enumerable = posts.posts.Where(p => p.sources.Length == 3 && p.sources[2].StartsWith("https://api.telegram.org")).ToArray();
        var results = enumerable.Select(p => new Result
        {
            type = "sticker",
            id = p.id.ToString(),
            sticker_file_id = p.sources[1]
        }).ToArray();

        return new AnswerInlineQuery
        {
            method = "answerInlineQuery",
            inline_query_id = update.inline_query.id,
            results = results,
            cache_time = 0,
            is_personal = true,
            next_offset = ""
        };
    }
}
