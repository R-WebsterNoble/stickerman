using System.Text;
using JetBrains.Annotations;
using Microsoft.Extensions.Logging;

namespace StickerManBot;

public class RequestLoggerMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<RequestLoggerMiddleware> _logger;

    public RequestLoggerMiddleware(RequestDelegate next, ILogger<RequestLoggerMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    [UsedImplicitly]
    public async Task InvokeAsync(HttpContext context)
    {
        context.Request.EnableBuffering();

        var buffer = new byte[Convert.ToInt32(context.Request.ContentLength)];
        var readBytes = await context.Request.Body.ReadAsync(buffer, context.RequestAborted);

        if (readBytes != buffer.Length)
            _logger.LogInformation("unable to read full Request buffer Length, {bufferLength}, {readBytes}", buffer.Length, readBytes);

        var requestBody = Encoding.UTF8.GetString(buffer);
        context.Request.Body.Seek(0, SeekOrigin.Begin);

        _logger.LogInformation("Request Body:{requestBody}", requestBody);

        var originalBodyStream = context.Response.Body;

        using var responseBody = new MemoryStream();
        context.Response.Body = responseBody;

        await _next(context);

        context.Response.Body.Seek(0, SeekOrigin.Begin);
        buffer = new byte[Convert.ToInt32(context.Response.Body.Length)];
        readBytes = await context.Response.Body.ReadAsync(buffer, context.RequestAborted);

        if (readBytes != buffer.Length)
            _logger.LogInformation("unable to read full Response buffer Length, {bufferLength}, {readBytes}", buffer.Length, readBytes);

        context.Response.Body.Seek(0, SeekOrigin.Begin);

        _logger.LogInformation("Response Body:{response}", Encoding.UTF8.GetString(buffer));
        await responseBody.CopyToAsync(originalBodyStream, context.RequestAborted);
    }
}