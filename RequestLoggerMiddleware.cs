using System.Diagnostics;
using System.Net.Http.Headers;
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
            _logger.LogInformation("unable to read full Request buffer Length, {bufferLength}, {readBytes}",
                buffer.Length, readBytes);

        var requestBody = Encoding.UTF8.GetString(buffer);
        context.Request.Body.Seek(0, SeekOrigin.Begin);

        _logger.LogInformation("Request Body:{requestBody}", requestBody.Replace('\n',' '));

        var originalBodyStream = context.Response.Body;

        using var responseBody = new MemoryStream();
        context.Response.Body = responseBody;

        await _next(context);

        context.Response.Body.Seek(0, SeekOrigin.Begin);
        buffer = new byte[Convert.ToInt32(context.Response.Body.Length)];
        readBytes = await context.Response.Body.ReadAsync(buffer, context.RequestAborted);

        if (readBytes != buffer.Length)
            _logger.LogInformation("unable to read full Response buffer Length, {bufferLength}, {readBytes}",
                buffer.Length, readBytes);

        context.Response.Body.Seek(0, SeekOrigin.Begin);

        _logger.LogInformation("Response Body:{response}", Encoding.UTF8.GetString(buffer));
        await responseBody.CopyToAsync(originalBodyStream, context.RequestAborted);
    }

    public class HttpLoggingHandler : DelegatingHandler
    {
        private readonly ILogger<HttpLoggingHandler> _logger;

        public HttpLoggingHandler(ILogger<HttpLoggingHandler> logger)
        {
            _logger = logger;
        }

        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request,
            CancellationToken cancellationToken)
        {
            var req = request;
            var id = Guid.NewGuid().ToString();
            var msg = $"[{id} -   Request]";

            Debug.WriteLine($"{msg}========Start==========");
            Debug.WriteLine($"{msg} {req.Method} {req.RequestUri.PathAndQuery} {req.RequestUri.Scheme}/{req.Version}");
            Debug.WriteLine($"{msg} Host: {req.RequestUri.Scheme}://{req.RequestUri.Host}");

            foreach (var header in req.Headers)
                Debug.WriteLine($"{msg} {header.Key}: {string.Join(", ", header.Value)}");

            if (req.Content != null)
            {
                foreach (var header in req.Content.Headers)
                    Debug.WriteLine($"{msg} {header.Key}: {string.Join(", ", header.Value)}");

                if (req.Content is StringContent || IsTextBasedContentType(req.Headers) ||
                    this.IsTextBasedContentType(req.Content.Headers))
                {
                    var result = await req.Content.ReadAsStringAsync(cancellationToken).ConfigureAwait(false);

                    Debug.WriteLine($"{msg} Content:");
                    Debug.WriteLine($"{msg} {result}");
                }
            }

            var start = DateTime.Now;

            var response = await base.SendAsync(request, cancellationToken).ConfigureAwait(false);

            var end = DateTime.Now;

            Debug.WriteLine($"{msg} Duration: {end - start}");
            Debug.WriteLine($"{msg}==========End==========");

            msg = $"[{id} - Response]";
            Debug.WriteLine($"{msg}=========Start=========");

            var resp = response;

            Debug.WriteLine(
                $"{msg} {req.RequestUri.Scheme.ToUpper()}/{resp.Version} {(int)resp.StatusCode} {resp.ReasonPhrase}");

            foreach (var header in resp.Headers)
                Debug.WriteLine($"{msg} {header.Key}: {string.Join(", ", header.Value)}");

            if (resp.Content != null)
            {
                foreach (var header in resp.Content.Headers)
                    Debug.WriteLine($"{msg} {header.Key}: {string.Join(", ", header.Value)}");

                if (resp.Content is StringContent || this.IsTextBasedContentType(resp.Headers) ||
                    this.IsTextBasedContentType(resp.Content.Headers))
                {
                    start = DateTime.Now;
                    var result = await resp.Content.ReadAsStringAsync(cancellationToken).ConfigureAwait(false);
                    end = DateTime.Now;

                    Debug.WriteLine($"{msg} Content:");
                    Debug.WriteLine($"{msg} {result}");
                    Debug.WriteLine($"{msg} Duration: {end - start}");
                }
            }

            Debug.WriteLine($"{msg}==========End==========");
            return response;
        }

    //     protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request,
    //          CancellationToken cancellationToken)
    //     {
    //         var req = request;
    //         var id = Guid.NewGuid().ToString();
    //         var msg = $"[{id} -   Request]";
    //
    //         _logger.LogDebug("{msg}========Start==========", msg);
    //         _logger.LogDebug("{msg} {reqMethod} {reqPathAndQuery} {reqScheme}/{reqVersion}", msg, req.Method, req.RequestUri.PathAndQuery, req.RequestUri.Scheme, req.Version);
    //         _logger.LogDebug("{msg} Host: {reqSchemeHost}", msg, $"{req.RequestUri.Scheme}://{req.RequestUri.Host}");
    //
    //         foreach (var header in req.Headers)
    //             _logger.LogDebug("{msg} {headerKey}: {headerValues}", msg, header.Key, string.Join(", ", header.Value));
    //
    //         if (req.Content != null)
    //         {
    //             foreach (var header in req.Content.Headers)
    //                 _logger.LogDebug("{msg} {headerKey}: {headerValues}", msg, header.Key, string.Join(", ", header.Value));
    //
    //             if (req.Content is StringContent || IsTextBasedContentType(req.Headers) ||
    //                 this.IsTextBasedContentType(req.Content.Headers))
    //             {
    //                 var result = await req.Content.ReadAsStringAsync();
    //
    //                 _logger.LogDebug("{msg} Content:", msg);
    //                 _logger.LogDebug("{msg} {content}", msg, result);
    //             }
    //         }
    //
    //         var start = DateTime.Now;
    //
    //         var response = await base.SendAsync(request, cancellationToken).ConfigureAwait(false);
    //
    //         var end = DateTime.Now;
    //
    //         _logger.LogDebug("{msg} Duration: {duration}", msg, end - start);
    //         _logger.LogDebug("{msg}==========End==========", msg);
    //
    //         msg = $"[{id} - Response]";
    //         _logger.LogDebug("{msg}=========Start=========", msg);
    //
    //         var resp = response;
    //
    //         _logger.LogDebug("{msg} {reqScheme}/{respVersion} {statusCode} {reasonPhrase}", msg, req.RequestUri.Scheme.ToUpper(), resp.Version, (int)resp.StatusCode, resp.ReasonPhrase);
    //
    //         foreach (var header in resp.Headers)
    //             _logger.LogDebug("{msg} {headerKey}: {headerValues}", msg, header.Key, string.Join(", ", header.Value));
    //
    //         if (resp.Content != null)
    //         {
    //             foreach (var header in resp.Content.Headers)
    //                 _logger.LogDebug("{msg} {headerKey}: {headerValues}", msg, header.Key, string.Join(", ", header.Value));
    //
    //             if (resp.Content is StringContent || this.IsTextBasedContentType(resp.Headers) ||
    //                 this.IsTextBasedContentType(resp.Content.Headers))
    //             {
    //                 start = DateTime.Now;
    //                 var result = await resp.Content.ReadAsStringAsync();
    //                 end = DateTime.Now;
    //
    //                 _logger.LogDebug("{msg} Content:", msg);
    //                 _logger.LogDebug("{msg} {content}", msg, result);
    //                 _logger.LogDebug("{msg} Duration: {duration}", msg, end - start);
    //             }
    //         }
    //
    //         _logger.LogDebug("{msg}==========End==========", msg);
    //         return response;
    //     }

        readonly string[] types = new[] { "html", "text", "xml", "json", "txt", "x-www-form-urlencoded" };

        bool IsTextBasedContentType(HttpHeaders headers)
        {
            IEnumerable<string> values;
            if (!headers.TryGetValues("Content-Type", out values))
                return false;
            var header = string.Join(" ", values).ToLowerInvariant();

            return types.Any(t => header.Contains(t));
        }
    }


}
