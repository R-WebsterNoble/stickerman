using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using System.Security.Cryptography;
using System.Text;
using StickerManBot.Types.Telegram;

namespace StickerManBot.Controllers;

[ApiController]
[Route("~/")]
public class StickerManBotController : Controller
{
    private readonly IE621Api _e621Api;

    public StickerManBotController(IE621Api e621Api)
    {
        _e621Api = e621Api;
    }

    [HttpPost]
    public async Task<IActionResult> Post([FromBody] Update update)
    {
        if (update.message == null)
            return Ok();

        if (update.message.sticker == null)
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
        ;
        await _e621Api.Upload(
            new blah
            {
                Upload = new Upload
                {
                    direct_url = "https://static1.e926.net/data/c9/e8/c9e85c6ecc8f80af55914d8e24689a85.png",
                    tag_string = "green_body paws tail fangs",
                    source = "",
                    rating = "e"
                }
            });

        return Ok();
    }

}
