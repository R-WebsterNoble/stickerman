global using StickerManBot.Types;
using StickerManBot;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddOptions<StickerManBotAuthenticationSchemeOptions>()
    .BindConfiguration("StickerManBotAuthenticationSchemeOptions")
    .ValidateDataAnnotations()
    .ValidateOnStart();

builder.Services.AddControllers();

builder.Services.AddAuthentication("StickerManBotAuthentication")
    .AddScheme<StickerManBotAuthenticationSchemeOptions, StickerManBotAuthenticationHandler>(
        "StickerManBotAuthentication", _ => { });

var app = builder.Build();

app.UseMiddleware<RequestLoggerMiddleware>();

app.UseRouting();

app.UseAuthentication();
app.UseAuthorization();

app.UseEndpoints(e => 
    e.MapControllers()
    .RequireAuthorization());

app.Run();