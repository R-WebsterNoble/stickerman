using Refit;
using StickerManBot.Types.e621;

namespace StickerManBot;

public interface IE621Api
{
    [Get("/posts?tags=Source:{id}")]
    Task<Posts> GetPost(string id);

    [Get("/posts.json?limit=50&tags={tags}")]
    Task<Posts> GetPosts(string tags);

    [Post("/uploads.json")]
    Task Upload(UploadWrapper upload);

    [Patch("/posts/{Post_ID}.json")]
    Task Update(int Post_ID, UpdateWrapper blah);
}

public class UpdateWrapper
{
    public Post Post { get; set; }
}

public class Post
{
    public string tag_string_diff { get; set; }
}

public class UploadWrapper
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