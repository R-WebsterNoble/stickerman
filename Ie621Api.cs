using Refit;
using StickerManBot.Types.e621;

namespace StickerManBot;

public interface IE621Api
{
    [Get("/posts/")]
    Task<Posts> GetPosts();

    [Post("/uploads.json")]
    Task Upload(blah upload);
}

public class blah
{
    public Upload Upload { get; set; }
}

public class Upload
{
    public string direct_url { get; set; }
    public string tag_string { get; set; }
    public string source { get; set; }
    public string rating { get; set; }
}