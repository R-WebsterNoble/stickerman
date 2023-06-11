namespace StickerManBot.Types.Telegram;

public class Update
{
    public int update_id { get; set; }
    public Message? message { get; set; }
}

public class Message
{
    public int message_id { get; set; }
    public User? from { get; set; }
    public Chat? chat { get; set; }
    public int date { get; set; }
    public Sticker? sticker { get; set; }
    public Message? reply_to_message { get; set; }
    public string? text { get; set; }
}

public class User
{
    public long id { get; set; }
    public bool is_bot { get; set; }
    public string first_name { get; set; }
    public string username { get; set; }
    public string language_code { get; set; }
    public bool is_premium { get; set; }
}

public class Chat
{
    public long id { get; set; }
    public string first_name { get; set; }
    public string username { get; set; }
    public string type { get; set; }
}

public class Sticker
{
    public int width { get; set; }
    public int height { get; set; }
    public string emoji { get; set; }
    public string set_name { get; set; }
    public bool is_animated { get; set; }
    public bool is_video { get; set; }
    public string type { get; set; }
    public string file_id { get; set; }
    public string file_unique_id { get; set; }
    public int file_size { get; set; }
}
