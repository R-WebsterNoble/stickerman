using System.Net.Http.Headers;
using Refit;
using StickerManBot;
using StickerManBot.services;
using static StickerManBot.RequestLoggerMiddleware;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddLogging();
builder.Services.AddHttpLogging(_ => { });

// builder.Services.AddTransient<HttpLoggingHandler>();

builder.Services.AddOptions<StickerManBotAuthenticationSchemeOptions>()
    .BindConfiguration("StickerManBotAuthenticationSchemeOptions")
    .ValidateDataAnnotations()
    .ValidateOnStart();

builder.Services.AddAuthentication("StickerManBotAuthentication")
    .AddScheme<StickerManBotAuthenticationSchemeOptions, StickerManBotAuthenticationHandler>(
        "StickerManBotAuthentication", null);


builder.Services
    .AddRefitClient<IE621Api>()
    .ConfigureHttpClient((provider, client) =>
    {
        var configuration = provider.GetRequiredService<IConfiguration>();
        client.BaseAddress = new Uri(configuration.GetValue<string>("e621Api:Url"));
        var authenticationString = $"{configuration.GetValue<string>("e621Api:Username")}:{configuration.GetValue<string>("e621Api:Key")}";
        var base64EncodedAuthenticationString = Convert.ToBase64String(System.Text.Encoding.UTF8.GetBytes(authenticationString));
        client.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Basic", base64EncodedAuthenticationString);
        client.DefaultRequestHeaders.Accept.Clear();
        client.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
    });//.AddHttpMessageHandler<HttpLoggingHandler>();

builder.Services
    .AddRefitClient<ITelegramApi>()
    .ConfigureHttpClient((provider, client) =>
    {
        var configuration = provider.GetRequiredService<IConfiguration>();
        var token = configuration.GetValue<string>("TelegramApiToken");
        client.BaseAddress = new Uri($"https://api.telegram.org/bot{token}");
        client.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
    });//.AddHttpMessageHandler<HttpLoggingHandler>();

builder.Services.AddScoped<StickerManDbService>();

builder.Services.AddControllers();

var app = builder.Build();


app.UseMiddleware<RequestLoggerMiddleware>();

app.UseRouting();

app.UseAuthentication();
app.UseAuthorization();

app.MapControllers().RequireAuthorization();

app.Run();
