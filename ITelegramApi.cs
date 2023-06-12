using Refit;
using StickerManBot;
using StickerManBot.Types.e621;

namespace StickerManBot;

public interface ITelegramApi
{
    [Post("/getFile")]
    Task<GetFileResponse> GetFile(GetFileRequest getFileRequest);
}


#pragma warning disable CS8618 // Non-nullable field must contain a non-null value when exiting constructor. Consider declaring as nullable.

public class GetFileRequest
{
    public string file_id { get; set; }   
}

#pragma warning restore CS8618 // Non-nullable field must contain a non-null value when exiting constructor. Consider declaring as nullable.


