// ReSharper disable UnusedMember.Global
#pragma warning disable IDE1006
#pragma warning disable CS8618 // Non-nullable field must contain a non-null value when exiting constructor. Consider declaring as nullable.
namespace StickerManBot.Types.e621;

public class Posts
{
    public Post[] posts { get; set; }
}

public class Post
{
    public int id { get; set; }
    public object created_at { get; set; }
    public object updated_at { get; set; }
    public File file { get; set; }
    public Preview preview { get; set; }
    public Sample sample { get; set; }
    public Score score { get; set; }
    public Tags tags { get; set; }
    public object[] locked_tags { get; set; }
    public int change_seq { get; set; }
    public Flags flags { get; set; }
    public string rating { get; set; }
    public int fav_count { get; set; }
    public string[] sources { get; set; }
    public object[] pools { get; set; }
    public Relationships relationships { get; set; }
    public object approver_id { get; set; }
    public int uploader_id { get; set; }
    public string description { get; set; }
    public int comment_count { get; set; }
    public bool is_favorited { get; set; }
    public bool has_notes { get; set; }
    public object duration { get; set; }
}

public class File
{
    public int width { get; set; }
    public int height { get; set; }
    public string ext { get; set; }
    public int size { get; set; }
    public string md5 { get; set; }
    public string url { get; set; }
}

public class Preview
{
    public int width { get; set; }
    public int height { get; set; }
    public string url { get; set; }
}

public class Sample
{
    public bool has { get; set; }
    public int height { get; set; }
    public int width { get; set; }
    public string url { get; set; }
    public Alternates alternates { get; set; }
}

public class Alternates
{
}

public class Score
{
    public int up { get; set; }
    public int down { get; set; }
    public int total { get; set; }
}

public class Tags
{
    public string[] general { get; set; }
    public string[] species { get; set; }
    public string[] character { get; set; }
    public string[] copyright { get; set; }
    public string[] artist { get; set; }
    public object[] invalid { get; set; }
    public string[] lore { get; set; }
    public string[] meta { get; set; }
}

public class Flags
{
    public bool pending { get; set; }
    public bool flagged { get; set; }
    public bool note_locked { get; set; }
    public bool status_locked { get; set; }
    public bool rating_locked { get; set; }
    public bool deleted { get; set; }
}

public class Relationships
{
    public object parent_id { get; set; }
    public bool has_children { get; set; }
    public bool has_active_children { get; set; }
    public object[] children { get; set; }
}


public class GetFileResponse
{
    public bool ok { get; set; }
    public Result result { get; set; }
}

public class Result
{
    public string file_id { get; set; }
    public string file_unique_id { get; set; }
    public int file_size { get; set; }
    public string file_path { get; set; }
}