using Refit;
using StickerManBot.Types.e621;

namespace StickerManBot;

public interface IE621Api
{
    [Get("/posts?tags=Source:{id}")]
    Task<Posts> GetPosts(string id);

    [Post("/uploads.json")]
    Task Upload(blah upload);

    [Patch("/posts/{Post_ID}.json")]
    Task Update(int Post_ID, blah2 blah);
}

public class blah2
{
    public Post Post { get; set; }
}

public class Post
{
    public string tag_string_diff { get; set; }
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