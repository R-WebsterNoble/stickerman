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

        public async Task SetUserPost(long userId, string stickerFileUniqueId, int postId)
        {
            const string sql = "INSERT INTO public.sessions (user_id, unique_file_id, post_id)\n" +
                               "VALUES (@user_id, @unique_file_id, @post_id)\n" +
                               "on conflict (user_id) DO UPDATE SET unique_file_id = @unique_file_id,\n" +
                               "                                    post_id        = @post_id;";

            await _db.ExecuteAsync(sql, new { user_id = userId, unique_file_id = stickerFileUniqueId, post_id = postId });
        }

        public void Dispose()
        {
            _db.Dispose();
        }
    }
}
