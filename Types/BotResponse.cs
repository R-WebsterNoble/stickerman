namespace StickerManBot.Types;

public class BotResponse
{
    public string method { get; set; }
    public long chat_id { get; set; }
    public string text { get; set; }
}
