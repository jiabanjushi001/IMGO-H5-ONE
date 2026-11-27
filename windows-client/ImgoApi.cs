using System.Globalization;
using System.IO;
using System.Net;
using System.Net.Http.Headers;
using System.Net.Http;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.RegularExpressions;

namespace Imgo.Desktop;

internal sealed class ApiException(string message, long code) : Exception(message)
{
    public long Code => code;
    public bool SessionExpired => code is -1 or 401;
}

internal static class ImgoApi
{
    private static readonly HttpClient Http = new(new HttpClientHandler
    {
        UseCookies = false,
        AllowAutoRedirect = false
    }) { Timeout = TimeSpan.FromSeconds(15) };

    public static Uri ServerUri(string value)
    {
        if (!Uri.TryCreate(value.Trim(), UriKind.Absolute, out var uri) ||
            (uri.Scheme != "https" && !(uri.Scheme == "http" && uri.IsLoopback)) ||
            uri.UserInfo.Length > 0 || uri.AbsolutePath != "/" || uri.Query.Length > 0 || uri.Fragment.Length > 0)
            throw new ArgumentException("请输入 HTTPS 服务地址。本机调试可用 http://127.0.0.1:8088。");
        return uri;
    }

    public static async Task<JsonElement> PostAsync(
        Uri server, string path, object payload, string? token, CancellationToken cancellationToken) =>
        (await PostPageAsync(server, path, payload, token, cancellationToken)).Data;

    public static async Task<ApiPage> PostPageAsync(
        Uri server, string path, object payload, string? token, CancellationToken cancellationToken)
    {
        using var request = new HttpRequestMessage(HttpMethod.Post, new Uri(server, path))
        {
            Content = JsonContent.Create(payload)
        };
        if (token is not null)
            request.Headers.TryAddWithoutValidation("Authorization", token);

        using var response = await Http.SendAsync(request, cancellationToken);
        return await ReadPageAsync(response, cancellationToken);
    }

    public static async Task UploadChatAttachmentAsync(Uri server, string token, string path,
        Conversation conversation, string id, CancellationToken cancellationToken, string? type = null)
    {
        var result = await UploadAsync(server, token, path, "uploadfile", cancellationToken, new
        {
            toContactId = conversation.Id,
            is_group = conversation.Group,
            id,
            type,
            fileName = Path.GetFileName(path)
        });
        if (Field(result, "id") != id || Number(result, "file_id") <= 0)
            throw new ApiException("服务器未确认附件消息，请检查会话后再试。", 0);
    }

    public static async Task<JsonElement> UploadAsync(Uri server, string token, string path, string action,
        CancellationToken cancellationToken, object? message = null)
    {
        using var transfer = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        transfer.CancelAfter(TimeSpan.FromMinutes(5));
        await using var file = File.OpenRead(path);
        if (file.Length == 0) throw new IOException("无法上传空文件。");

        using var request = new HttpRequestMessage(HttpMethod.Post, new Uri(server, $"/common/upload/{action}"));
        request.Headers.TryAddWithoutValidation("Authorization", token);
        using var multipart = new MultipartFormDataContent();
        request.Content = multipart;
        var content = new StreamContent(file);
        content.Headers.ContentType = new MediaTypeHeaderValue("application/octet-stream");
        multipart.Add(content, "file", Path.GetFileName(path));
        if (message is not null) multipart.Add(new StringContent(JsonSerializer.Serialize(message)), "message");
        using var response = await TransferHttp.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, transfer.Token);
        return (await ReadPageAsync(response, transfer.Token)).Data;
    }

    public static async Task CopyAttachmentAsync(Uri server, string token, string location, Stream output,
        long? maxBytes, CancellationToken cancellationToken)
    {
        if (!Uri.TryCreate(server, location, out var target) || target.UserInfo.Length > 0 || !SameOrigin(server, target) ||
            !(target.AbsolutePath.StartsWith("/filedown/", StringComparison.Ordinal) ||
              target.AbsolutePath.StartsWith("/storage/", StringComparison.Ordinal) ||
              target.AbsolutePath.StartsWith("/avatar/", StringComparison.Ordinal)))
            throw new IOException("附件地址无效。");

        using var transfer = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        transfer.CancelAfter(TimeSpan.FromMinutes(5));
        for (var attempt = 0; attempt < 3; attempt++)
        {
            using var request = new HttpRequestMessage(HttpMethod.Get, target);
            if (SameOrigin(server, target)) request.Headers.TryAddWithoutValidation("Authorization", token);
            using var response = await TransferHttp.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, transfer.Token);
            if ((int)response.StatusCode is 301 or 302 or 303 or 307 or 308)
            {
                if (response.Headers.Location is not { } redirect || !Uri.TryCreate(target, redirect, out var next) ||
                    next.UserInfo.Length > 0 || (next.Scheme != "https" && !SameOrigin(server, next)))
                    throw new IOException("附件下载地址无效。");
                target = next;
                continue;
            }

            if (response.StatusCode == HttpStatusCode.Unauthorized && SameOrigin(server, target))
                throw new ApiException("登录已失效，请重新登录。", 401);
            if (!response.IsSuccessStatusCode)
                throw new IOException(response.StatusCode == HttpStatusCode.Forbidden ? "没有权限下载附件。" :
                    $"下载附件失败（HTTP {(int)response.StatusCode}）。");
            await using var input = await response.Content.ReadAsStreamAsync(transfer.Token);
            if (maxBytes is not { } limit)
            {
                await input.CopyToAsync(output, transfer.Token);
                return;
            }
            if (response.Content.Headers.ContentLength > limit)
                throw new IOException("附件超过预览大小限制，请先保存到本地查看。");
            var buffer = new byte[81920];
            long written = 0;
            while (true)
            {
                var count = await input.ReadAsync(buffer, transfer.Token);
                if (count == 0) return;
                written += count;
                if (written > limit) throw new IOException("附件超过预览大小限制，请先保存到本地查看。");
                await output.WriteAsync(buffer.AsMemory(0, count), transfer.Token);
            }
        }
        throw new IOException("附件下载跳转次数过多。");
    }

    private static bool SameOrigin(Uri server, Uri target) =>
        Uri.Compare(server, target, UriComponents.SchemeAndServer, UriFormat.SafeUnescaped, StringComparison.OrdinalIgnoreCase) == 0;

    private static readonly HttpClient TransferHttp = new(new HttpClientHandler
    {
        UseCookies = false,
        AllowAutoRedirect = false
    }) { Timeout = Timeout.InfiniteTimeSpan };

    private static async Task<ApiPage> ReadPageAsync(HttpResponseMessage response, CancellationToken cancellationToken)
    {
        if (!response.IsSuccessStatusCode)
            throw new ApiException($"服务返回 HTTP {(int)response.StatusCode}，请确认 API 地址。", (int)response.StatusCode);

        try
        {
            using var document = await JsonDocument.ParseAsync(
                await response.Content.ReadAsStreamAsync(cancellationToken), cancellationToken: cancellationToken);
            var root = document.RootElement;
            var code = Number(root, "code", int.MinValue);
            if (code != 0)
            {
                var message = Field(root, "msg");
                throw new ApiException(message.Length == 0 ? "服务返回了异常结果。" : message, code);
            }
            return new ApiPage(root.GetProperty("data").Clone(), Number(root, "count"));
        }
        catch (Exception exception) when (exception is JsonException or KeyNotFoundException or InvalidOperationException)
        {
            throw new ApiException("服务返回格式异常，请确认填写的是 IMGO API 地址。", 0);
        }
    }

    public static Uri? AvatarUri(Uri server, string value)
    {
        if (value.Length == 0 || !Uri.TryCreate(server, value, out var target) || !SameOrigin(server, target) ||
            target.UserInfo.Length > 0) return null;
        return target.AbsolutePath.StartsWith("/avatar/", StringComparison.Ordinal) ||
            target.AbsolutePath.StartsWith("/storage/avatar/", StringComparison.Ordinal) ||
            target.AbsolutePath.StartsWith("/storage/image/", StringComparison.Ordinal) ? target : null;
    }

    public static JsonElement Property(JsonElement value, string key) =>
        value.ValueKind == JsonValueKind.Object && value.TryGetProperty(key, out var property) ? property : default;

    public static string Field(JsonElement value, string key)
    {
        var property = Property(value, key);
        return property.ValueKind switch
        {
            JsonValueKind.String => property.GetString() ?? "",
            JsonValueKind.Number => property.GetRawText(),
            _ => ""
        };
    }

    public static long Number(JsonElement value, string key, long fallback = 0) =>
        long.TryParse(Field(value, key), NumberStyles.Integer, CultureInfo.InvariantCulture, out var number)
            ? number : fallback;

    private static readonly Regex ChatMarkup = new(
        """<(?<close>/)?(?<tag>p|div|br|span|b|strong|i|em|u|s|strike|del|ins|a|code|pre|blockquote|ul|ol|li|h[1-6]|font|small|sub|sup|hr)(?:\s+(?:[^<>"']|"[^"]*"|'[^']*')*)?/?>""",
        RegexOptions.IgnoreCase | RegexOptions.CultureInvariant | RegexOptions.NonBacktracking);

    public static string MessageText(JsonElement data, string field) => Field(data, "type") switch
    {
        "image" => "[图片]",
        "voice" => "[语音]",
        "video" => "[视频]",
        "file" => "[文件]",
        "webrtc" => Field(data, field) is { Length: > 0 } result ? result : "[通话]",
        "text" or "" => PlainChatText(Field(data, field)),
        _ => Field(data, field)
    };

    private static string PlainChatText(string value)
    {
        // Decode after removing markup so escaped tags typed by the sender stay literal.
        var text = ChatMarkup.Replace(value, match =>
        {
            var tag = match.Groups["tag"].Value.ToLowerInvariant();
            return tag is "br" or "hr" ||
                (match.Groups["close"].Success && tag is "p" or "div" or "li" or "pre" or "blockquote"
                    or "h1" or "h2" or "h3" or "h4" or "h5" or "h6")
                ? "\n" : "";
        });
        return WebUtility.HtmlDecode(text == value ? value : text.TrimEnd('\n'));
    }
}

internal sealed record Conversation(string Id, string Name, long Group, long Unread, string Preview,
    string Account = "", long Timestamp = 0, bool Mentioned = false, bool Pinned = false, bool Muted = false, long Role = 0, bool ManagerCanInvite = false, Uri? Avatar = null, bool Online = false)
{
    public string Kind => Group switch { 1 => "群聊", 2 => "系统公告", 3 => "文件助手", _ => "私聊" };
    public string Badge => Unread > 99 ? "99+" : Unread > 0 ? Unread.ToString() : "";
    public bool HasUnread => Unread > 0;
    public string Initial => Name.Length > 0 ? StringInfo.GetNextTextElement(Name) : "?";
    public string LastTime
    {
        get
        {
            if (Timestamp is <= 0 or > 253402300799999) return "";
            var date = DateTimeOffset.FromUnixTimeMilliseconds(Timestamp).ToLocalTime();
            return date.ToString(date.Date == DateTime.Today ? "HH:mm" : "MM-dd");
        }
    }
}

internal sealed record ChatMessage(long Id, string Sender, string Time, string Body, bool IsMine, string Receipt = "", string Type = "text", string FileName = "", string Download = "", long ParentId = 0, string Quote = "", long FileId = 0, string Poster = "",
    long SenderId = 0, long Timestamp = 0, Uri? Avatar = null, string AvatarInitial = "",
    string TimeDivider = "", bool ShowAvatar = false, bool ShowSender = false)
{
    public bool HasAttachment => Download.Length > 0;
    public bool IsImage => Type == "image";
    public string Thumbnail => IsImage ? Download : Type == "video" && Poster.StartsWith("/storage/", StringComparison.Ordinal) ? Poster : "";
    public bool HasThumbnail => Thumbnail.Length > 0;
    public bool IsPlayable => Type is "voice" or "video";
    public bool HasQuote => Quote.Length > 0;
    public string AttachmentLabel => FileName.Length > 0 ? FileName : "附件";
}

internal sealed record ApiPage(JsonElement Data, long Count);
