using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using StickerManBot.Types;
using System.Security.Cryptography;
using System.Text;

namespace StickerManBot.Controllers;

[ApiController]
[Route("~/")]
public class StickerManBotController : Controller
{
    [HttpPost]
    [Route("{ApiKey}")]
    public IActionResult Get(string apiKey, [FromBody] Update update)
    {
        // var apiKeyBytes = new ReadOnlySpan<byte>(Encoding.UTF8.GetBytes(apiKey));
        // if (!CryptographicOperations.FixedTimeEquals(new ReadOnlySpan<byte>(_apiKey), apiKeyBytes))
        //     return Unauthorized();

        return Ok(new 
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
    }

}
