using Refit;
using StickerManBot.Types.e621;

namespace StickerManBot;

public interface IE621Api
{
    [Get("/posts?tags=Source:{id}")]
    Task<Posts> GetPost(string id);

    [Get("/posts.json?limit=50&page={page}&tags={tags}")]
    Task<Posts> GetPosts(int page, string tags);

    [Post("/uploads.json")]
    Task<UploadResult> Upload([Header("Authorization")] string accessToken, UploadRequest uploadRequest);

    [Patch("/posts/{postID}.json")]
    Task Update(int postId, UpdateRequest updateRequest);

    [Post("/users.json")]
    Task<User> CreateUser(CreateUserRequest createUserRequest);

}

#pragma warning disable CS8618 // Non-nullable field must contain a non-null value when exiting constructor. Consider declaring as nullable.
#pragma warning disable IDE1006 // Naming Styles
// ReSharper disable UnusedMember.Global

public class UpdateRequest
{
    public Post post { get; set; }

    public class Post
    {
        public string tag_string_diff { get; set; }
    }
}



public class UploadRequest
{
    public Upload upload { get; set; }

    public class Upload
    {
        public string direct_url { get; set; }
        public string tag_string { get; set; }
        public string source { get; set; }
        public string rating { get; set; }
    }
}

public class CreateUserRequest
{
    public User user { get; set; }

    public class User
    {
        public string name { get; set; }
        public string password { get; set; }
        public string password_confirmation { get; set; }
    }
}


public class User
{
    public int id { get; set; }
    public DateTime created_at { get; set; }
    public string name { get; set; }
    public int level { get; set; }
    public int base_upload_limit { get; set; }
    public int post_upload_count { get; set; }
    public int post_update_count { get; set; }
    public int note_update_count { get; set; }
    public bool is_banned { get; set; }
    public bool can_approve_posts { get; set; }
    public bool can_upload_free { get; set; }
    public string level_string { get; set; }
    public object avatar_id { get; set; }
    public ApiKey api_key { get; set; }
}

public class ApiKey
{
    public int id { get; set; }
    public int user_id { get; set; }
    public string key { get; set; }
    public DateTime created_at { get; set; }
    public DateTime updated_at { get; set; }
}


#pragma warning restore CS8618 // Non-nullable field must contain a non-null value when exiting constructor. Consider declaring as nullable.
#pragma warning restore IDE1006 // Naming Styles