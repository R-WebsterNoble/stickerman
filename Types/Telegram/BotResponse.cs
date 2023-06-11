namespace StickerManBot.Types.Telegram;

public class BotResponse
{
    public string method { get; set; }
    public long chat_id { get; set; }
    public string text { get; set; }
}
