using System.ComponentModel.DataAnnotations;
using System.Security.Claims;
using System.Security.Cryptography;
using System.Text;
using System.Text.Encodings.Web;
using JetBrains.Annotations;
using Microsoft.AspNetCore.Authentication;
using Microsoft.Extensions.Options;

namespace StickerManBot;

public class StickerManBotAuthenticationSchemeOptions : AuthenticationSchemeOptions
{
    private string _token = null!;

    [StringLength(44, MinimumLength = 44)]
    [UsedImplicitly]
    public string Token
    {
        get => _token;
        set
        {
            TokenBytes = Encoding.UTF8.GetBytes(value);
            _token = value;
        }
    }

    public byte[] TokenBytes { get; private set; } = null!;
}

public class StickerManBotAuthenticationHandler : AuthenticationHandler<StickerManBotAuthenticationSchemeOptions>
{
    private readonly IOptionsMonitor<StickerManBotAuthenticationSchemeOptions> _options;

    private static readonly AuthenticationTicket SuccessAuthenticationTicket = new AuthenticationTicket(new ClaimsPrincipal(new ClaimsIdentity(new List<Claim>(), "auth")), "StickerManBotAuthentication");

    public StickerManBotAuthenticationHandler(IOptionsMonitor<StickerManBotAuthenticationSchemeOptions> options, ILoggerFactory logger, UrlEncoder encoder)
        : base(options, logger, encoder)
    {
        _options = options;
    }

    protected override Task<AuthenticateResult> HandleAuthenticateAsync()
    {
        if (!Request.Headers.TryGetValue("X-Telegram-Bot-Api-Secret-Token", out var token))
            return Task.FromResult(AuthenticateResult.Fail("X-Telegram-Bot-Api-Secret-Token header absent"));

        if (string.IsNullOrEmpty(token))
            return Task.FromResult(AuthenticateResult.Fail("X-Telegram-Bot-Api-Secret-Token header present but null"));

        if (!CryptographicOperations.FixedTimeEquals(new ReadOnlySpan<byte>(_options.CurrentValue.TokenBytes), new ReadOnlySpan<byte>(Encoding.UTF8.GetBytes(token!))))
            return Task.FromResult(AuthenticateResult.Fail("X-Telegram-Bot-Api-Secret-Token header present but mismatch"));

        return Task.FromResult(AuthenticateResult.Success(SuccessAuthenticationTicket));
    }
}
