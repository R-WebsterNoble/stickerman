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
            return await _db.QuerySingleOrDefaultAsync<bool>("SELECT true FROM public.user_age_verification WHERE user_id = @user_id", new {user_id = userId});
        }

        public async Task SetUserAgeVerified(long userId)
        {
            await _db.ExecuteAsync("INSERT INTO public.user_age_verification (user_id) VALUES (@user_id) ON CONFLICT (user_id) DO NOTHING", new { user_id = userId });
        }

        public void Dispose()
        {
            _db.Dispose();
        }
    }
}
