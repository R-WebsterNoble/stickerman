using Microsoft.AspNetCore.Mvc;
using StickerManBot.Types.Telegram;

namespace StickerManBot.Controllers;

[ApiController]
[Route("~/")]
public class StickerManBotController : Controller
{
    private readonly IConfiguration _config;
    private readonly IE621Api _e621Api;
    private readonly ITelegramApi _telegramApi;

    public StickerManBotController(IConfiguration config, IE621Api e621Api, ITelegramApi telegramApi)
    {
        _config = config;
        _e621Api = e621Api;
        _telegramApi = telegramApi;
    }

    [HttpPost]
    public async Task<IActionResult> Post([FromBody] Update update)
    {
        if (update.message == null || update.message.chat == null)
            return Ok();

        var message = update.message.reply_to_message ?? update.message;

        var stickerFileId = message.sticker?.file_id;
        var stickerFileUniqueId = message.sticker?.file_unique_id;

        if (stickerFileId == null || stickerFileUniqueId == null)
            return Ok(new BotResponse
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
            });

        var posts = await _e621Api.GetPosts(stickerFileUniqueId);

        if (posts.posts.Length == 0)
        {
            var fileResponse = await _telegramApi.GetFile(new GetFileRequest { file_id = stickerFileId });

            await _e621Api.Upload(
                new blah
                {
                    Upload = new Upload
                    {
                        direct_url = $"https://api.telegram.org/file/bot{_config.GetValue<string>("TelegramApiToken")}/{fileResponse.result.file_path}",
                        tag_string = update.message.text?? "",
                        source = stickerFileUniqueId,
                        rating = "e"
                    }
                });

            return Ok(new BotResponse
            {
                chat_id = update.message.chat.id,
                method = "sendMessage",
                text = "Created new Sticker"
            });
        }

        await _e621Api.Update(posts.posts.First().id,
            new blah2
            {
                Post = new Post
                {
                    tag_string_diff = update.message.text ?? ""
                }
            });

        return Ok(new BotResponse
        {
            chat_id = update.message.chat.id,
            method = "sendMessage",
            text = "Updated existing Sticker"
        });
    }

}
