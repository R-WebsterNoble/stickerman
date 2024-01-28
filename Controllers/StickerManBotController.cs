using System.Text.Json;
using System.Text.RegularExpressions;
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
        try
        {
            if (update.message != null)
                return Ok(await ProcessMessage(update.message));

            if (update.inline_query != null)
                return Ok(await ProcessInlineQuery(update.inline_query));

            if (update.callback_query != null)
                return Ok(await ProcessCallbackQuery(update.callback_query));

            //throw new NotImplementedException("No handler for this update");
            return Ok();
        }
        catch (Exception e)
        {
            _logger.LogError(e, "Error Processing Update {@Update}", update);

            return Ok(new BotResponse
            {
                chat_id = _config.GetValue<long>("UserIdToReportErrorsTo"),
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

        var userId = message.from!.id;
        if (sticker != null)
        {
            return await ProcessSticker(message, sticker, userId);
        }

        if(string.IsNullOrWhiteSpace(message.text))
            return DefaultResponse(message.chat.id);

        if (message.text.Length > 0 && message.text[0] == '/')
        {
            if (message.text == "/start")
            {
                if (await _stickerManDbService.IsUserAgeVerified(userId))
                    return DefaultResponse(message.chat.id);

                return new InlineKeyboardBotResponse
                {
                    chat_id = message.chat.id,
                    method = "sendMessage",
                    text = "Welcome! Before using this Bot for the first time, please verify your age:",
                    reply_markup = new ReplyMarkup
                    {
                        inline_keyboard = [
                        [new InlineKeyboard { text = "I am 18+", callback_data = "true" }, new InlineKeyboard { text = "Get me out of here", callback_data = "false" }]
                    ]
                    }
                };                
            }
            else if (message.text == "/start ImOver18")
            {
                if (!await _stickerManDbService.IsUserAgeVerified(userId))
                {
                    await _stickerManDbService.SetUserAgeVerified(userId);
                    return new BotResponse
                    {
                        chat_id = message.chat.id,
                        method = "sendMessage",
                        text = "Thank you, you have verified your age. You may now search for stickers in chats."
                    };
                }
            }
            else
                return DefaultResponse(message.chat.id);
        }

        var userPostId = await _stickerManDbService.GetUserPostFromSession(userId);
        if(userPostId == null)
            return DefaultResponse(message.chat.id);

        var userE621ApiKey = await GetUserE621ApiKey(userId, message.chat.username);
        var authenticationString = $"u{userId}:{userE621ApiKey}";
        var base64EncodedAuthenticationString = Convert.ToBase64String(System.Text.Encoding.UTF8.GetBytes(authenticationString));        
        await _e621Api.Update($"basic {base64EncodedAuthenticationString}", userPostId.Value, new UpdateRequest
        {
            post = new UpdateRequest.Post
            {
                tag_string_diff = message.text
            }
        });
        
        var tagsAddedMessage = message.text.Contains(' ') ? "That tag has been added to the sticker." : "Those tags have been added to the sticker.";

        return new BotResponse
        {
            chat_id = message.chat.id,
            method = "sendMessage",
            text = tagsAddedMessage
        };


    }

    BotResponse DefaultResponse(long chatid)
    {
        return new BotResponse
        {
            chat_id = chatid,
            method = "sendMessage",
            text = "Hi, I'm Sticker Manager Bot.\n" +
                   "I'll help you manage your stickers by letting you tag them so you can easily find them later.\n" +
                   "\n" +
                   "Usage:\n" +
                   "To add a sticker tag, first send me a sticker to this chat, then send the tags you'd like to add to the sticker.\n" +
                   "\n" +
                   "You can then easily search for tagged stickers in any chat. Just type: @StickerManBot followed by the tags of the stickers that you are looking for."
        };
    }

    private async Task<BotResponse> ProcessSticker(Message message, Sticker sticker, long userId)
    {
        var posts = await _e621Api.GetPost(sticker.file_unique_id);

        if (posts.posts.Length != 0)
        {
            var tags = posts.posts.First().tags;

            var allTags = tags.copyright.Select(t => $"[{t}](https://t.me/addstickers/{t})")
                .Concat(tags.general.Select(t => Regex.Replace(t, @"([_*\[\]\(\)~`>#\+\-\=|{}.!])", @"\$1")));

            return new MarkdownBotResponse()
            {
                chat_id = message.chat.id,
                method = "sendMessage",
                text = $"""
                Here are all the existing tags currently applied to that sticker:
                {string.Join('\n', allTags)}

                To add new tags please send them here\.
                You can add multiple tags with spaces inbeteen them\.
                """    
            };
        }

        var fileIdToGet = sticker.is_animated ? sticker.thumbnail!.file_id : sticker.file_id;
        var fileResponse = await _telegramApi.GetFile(new GetFileRequest { file_id = fileIdToGet });

        if (!fileResponse.ok)
            return new BotResponse
            {
                chat_id = _config.GetValue<long>("UserIdToReportErrorsTo"),
                method = "sendMessage",
                text = $"Something went wrong when getting details for a sticker from Telegram, Username {message.chat.username}"
            };

        var userE621ApiKey = await GetUserE621ApiKey(userId, message.chat.username);
        
        var authenticationString = $"u{userId}:{userE621ApiKey}";
        var base64EncodedAuthenticationString = Convert.ToBase64String(System.Text.Encoding.UTF8.GetBytes(authenticationString));
        var uploadResult = await _e621Api.Upload($"basic {base64EncodedAuthenticationString}", 
            new UploadRequest
            {
                upload = new UploadRequest.Upload
                {
                    direct_url =
                        $"https://api.telegram.org/file/bot{_config.GetValue<string>("TelegramApiToken")}/{fileResponse.result.file_path}",
                    tag_string = $"Copyright:{sticker.set_name}{(sticker.is_animated ? "animated" : "")}",
                    source = $"{sticker.file_unique_id}%0A{sticker.file_id}",
                    rating = "e"
                }
            });

        await _stickerManDbService.SetUserPost(userId, sticker.file_unique_id, uploadResult.post_id, userE621ApiKey);

        return new BotResponse
        {
            chat_id = message.chat.id,
            method = "sendMessage",
            text = "I've not seen that sticker before. To add new tags please send them here. You can add multiple tags with spaces inbeteen them."
        };
    }

    private async Task<string> GetUserE621ApiKey(long userId, string username)
    {
        var userE621ApiKey = await _stickerManDbService.GetUserE621ApiKey(userId);
        if (userE621ApiKey != null)
            return userE621ApiKey;

        var password = Guid.NewGuid().ToString();

        var e621User = await _e621Api.CreateUser(new CreateUserRequest
        {            
            user = new CreateUserRequest.User
            {
                name = $"u{userId}",
                password = password,
                password_confirmation = password,
                email = $"{username}@stickermanbot.com"
            }
        });

        string userApiKey = e621User.api_key.key;

        await _stickerManDbService.CreateUser(userId, userApiKey);

        return userApiKey;
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
                    text = "Click here to verify you are over the age of 18.",
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

    private async Task<BotResponse> ProcessCallbackQuery(CallbackQuery inlineQuery)
    {
        var chatId = inlineQuery.message.chat.id;

        if (bool.TryParse(inlineQuery.data, out var data) && data)
        {
            await _stickerManDbService.SetUserAgeVerified(chatId);
            return DefaultResponse(chatId);
        }
        else
            return new BotResponse
            {
                chat_id = chatId,
                method = "sendMessage",
                text = "Sorry, please come back when you are older."
            };
    }
}
