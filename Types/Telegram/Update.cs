// ReSharper disable UnusedMember.Global
#pragma warning disable IDE1006
#pragma warning disable CS8618 // Non-nullable field must contain a non-null value when exiting constructor. Consider declaring as nullable.

namespace StickerManBot.Types.Telegram;

public class BotResponse
{
    public string method { get; set; }
    public long chat_id { get; set; }
    public string text { get; set; }
}

public class InlineKeyboardBotResponse : BotResponse
{
    public ReplyMarkup reply_markup { get; set; }
}

public class ReplyMarkup
{
    public InlineKeyboard[][] inline_keyboard { get; set; }
}

public class InlineKeyboard
{
    public string text { get; set; }
    public string callback_data { get; set; }
}

public class MarkdownBotResponse : BotResponse
{
    public string parse_mode => "MarkdownV2";
}

public class Update
{
    public int update_id { get; set; }
    public Message? message { get; set; }
    public InlineQuery? inline_query { get; set; }
    public CallbackQuery? callback_query { get; set; }
}

public class CallbackQuery
{
    public long id { get; set; }
    public User from { get; set; }
    public Message message { get; set; }
    public string chat_instance { get; set; }
    public string data { get; set; }
}

public class Message
{
    public int message_id { get; set; }
    public User? from { get; set; }
    public Chat chat { get; set; }
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
    public string? language_code { get; set; }
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
    public PhotoSize? thumbnail { get; set; }
}

public class InlineQuery
{
    public string id { get; set; }
    public User from { get; set; }
    public string chat_type { get; set; }
    public string query { get; set; }
    public string offset { get; set; }
}


public class AnswerInlineQuery
{
    public string method { get; set; }
    public string inline_query_id { get; set; }
    public IEnumerable<Result> results { get; set; }
    public int cache_time { get; set; }
    public bool is_personal { get; set; }
    public string next_offset { get; set; }
}

public class AnswerInlineQueryWithButton : AnswerInlineQuery
{
    public InlineQueryResultsButton button { get; set; }
}

public class Result
{
    public string type { get; set; }
    public string id { get; set; }
    public string sticker_file_id { get; set; }
}

public class InlineQueryResultsButton
{
    public string text { get; set; }
    public string start_parameter { get; set; }
}

public class PhotoSize
{
    public string file_id { get; set; }
    public string file_unique_id { get; set; }
    public int file_size { get; set; }
    public int width { get; set; }
    public int height { get; set; }
}