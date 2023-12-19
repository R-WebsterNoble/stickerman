using Dapper;
using Npgsql;
using StickerManBot.Types.Telegram;

namespace StickerManBot.services
{
    public class StickerManDbService : IDisposable
    {
        private readonly NpgsqlConnection _db;

        public StickerManDbService(IConfiguration configuration)
        {
            var connectionString = configuration.GetConnectionString("StickerManDSN");
            _db = new NpgsqlConnection(connectionString);
        }

        //create table sessions
        // (
        // user_id           bigint not null unique,
        // unique_file_id    text,
        // post_id           integer,
        // age_verified      boolean default false not null,
        // e621_user_api_key text
        // );

        public async Task<bool> IsUserAgeVerified(long userId)
        {
            return await _db.QuerySingleOrDefaultAsync<bool>("SELECT age_verified FROM public.sessions WHERE user_id = @user_id;", new {user_id = userId});
        }

        public async Task SetUserAgeVerified(long userId)
        {
            const string sql = "INSERT INTO public.sessions (user_id, age_verified)\n" +
                                                                            "VALUES ( @user_id, true)\n" +
                                                                            "on conflict (user_id) DO UPDATE SET age_verified = true;";
            await _db.ExecuteAsync(sql, new { user_id = userId });
        }

        public async Task<int?> GetUserPostFromSession(long userId)
        {
            const string sql = "SELECT post_id FROM public.sessions WHERE user_id = @user_id;";
            return await _db.QuerySingleOrDefaultAsync<int?>(sql, new { user_id = userId });
        }

        public async Task SetUserPost(long userId, string stickerFileUniqueId, int postId, string userE621ApiKey)
        {
            const string sql = """
                               UPDATE public.sessions SET
                                 unique_file_id = @unique_file_id,
                                 post_id = @post_id
                                 WHERE user_id = @user_id;
                               """;

            await _db.ExecuteAsync(sql, new { user_id = userId, unique_file_id = stickerFileUniqueId, post_id = postId });
        }

        public async Task CreateUser(long userId, string userE621ApiKey)
        {
            const string sql = """
                               INSERT INTO public.sessions (user_id, e621_user_api_key)
                                 VALUES (@user_id, @e621_user_api_key);
                               """;

            await _db.ExecuteAsync(sql, new { user_id = userId, e621_user_api_key = userE621ApiKey });
        }
        
        public async Task<string?> GetUserE621ApiKey(long userId)
        {
            const string sql = "SELECT e621_user_api_key FROM sessions WHERE user_id = @user_id;";

            return await _db.QuerySingleOrDefaultAsync<string?>(sql, new { user_id = userId });
        }

        public void Dispose()
        {
            _db.Dispose();
        }
    }
}
