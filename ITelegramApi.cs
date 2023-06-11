using Refit;
using StickerManBot;
using StickerManBot.Types.e621;

namespace StickerManBot;

public interface ITelegramApi
{
    [Post("/getFile")]
    Task<GetFileResponse> GetFile(GetFileRequest getFileRequest);
}


public class GetFileRequest
{
    public string file_id { get; set; }   
}



