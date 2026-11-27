using System.ComponentModel;
using System.IO;
using System.Net.WebSockets;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace Imgo.Desktop;

internal sealed class AccountSession(Uri server, string token, long userId, string account, string name, JsonElement profile = default) : INotifyPropertyChanged
{
    private readonly CancellationTokenSource lifetime = new();
    private long revision = 1;
    private long historyRevision;
    public long HistoryRevision => Interlocked.Read(ref historyRevision);
    public void InvalidateHistory()
    {
        Interlocked.Increment(ref historyRevision);
        Invalidate();
    }
    public Uri Server { get; } = server;
    public long UserId { get; } = userId;
    public string Account { get; } = account;
    public string Email { get; set; } = ImgoApi.Field(profile, "email");
    public bool IsSuperAdmin => UserId == 1;
    public bool HasBackendRole => IsSuperAdmin || ImgoApi.Number(profile, "admin_role_id") > 0;
    public bool CanAddGroupFriends { get; private set; } = ImgoApi.Number(profile, "group_friend_add") == 1;
    public bool IsAgent { get; } = ImgoApi.Number(profile, "agent_mode") == 1;
    private readonly HashSet<string> permissions = ReadPermissions(profile);
    public bool HasPermission(string permission) => IsSuperAdmin || permissions.Contains(permission);

    private static HashSet<string> ReadPermissions(JsonElement profile) =>
        ImgoApi.Property(profile, "menu_permissions") is { ValueKind: JsonValueKind.Array } items
            ? items.EnumerateArray().Where(item => item.ValueKind == JsonValueKind.String)
                .Select(item => item.GetString()!).ToHashSet(StringComparer.Ordinal)
            : [];

    public Task<ApiPage> PostPageAsync(string path, object payload) =>
        ImgoApi.PostPageAsync(Server, path, payload, token, Cancellation);
    private string nameLabel = FormatName(name, account);
    private static string FormatName(string name, string account) => name.Length == 0 || name == account ? account : $"{name}（{account}）";
    public void UpdateName(string name)
    {
        nameLabel = FormatName(name, Account);
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(nameof(Label)));
    }
    public event PropertyChangedEventHandler? PropertyChanged;
    public string Label => Unread > 0 ? $"{nameLabel} · {Unread} 未读" : nameLabel;
    public long Unread { get; private set; }

    public void UpdateUnread()
    {
        var unread = Conversations.Sum(conversation => conversation.Unread);
        if (unread == Unread) return;
        Unread = unread;
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(nameof(Label)));
    }
    public CancellationToken Cancellation => lifetime.Token;
    public string SelectedConversation { get; set; } = "";
    public long FriendRequests { get; set; }
    public bool CallsEnabled { get; set; }
    public bool EnterToSend { get; set; }
    public Dictionary<string, Dictionary<string, string>> Mentions { get; } = new(StringComparer.Ordinal);
    public Dictionary<string, string> Drafts { get; } = new(StringComparer.Ordinal);
    public void Invalidate() => Interlocked.Increment(ref revision);
    public long Revision => Interlocked.Read(ref revision);
    public long LastContactRevision { get; set; }
    public DateTimeOffset RefreshAfter { get; set; }
    public DateTimeOffset LastContacts { get; set; }
    public List<Conversation> Conversations { get; set; } = [];
    public string ConnectionStatus { get; private set; } = "连接中";
    public bool Expired { get; private set; }
    public bool SigningOut { get; set; }

    public Task<ApiPage> GetAdminPageAsync(string path, object payload) => PostPageAsync(path, payload);

    public Task<JsonElement> PostAsync(string path, object payload) =>
        ImgoApi.PostAsync(Server, path, payload, token, Cancellation);

    public Task<JsonElement> GetMessagesAsync(Conversation conversation, int limit, long beforeId) =>
        PostAsync("/enterprise/im/getmessagelist", new
        {
            toContactId = conversation.Id,
            is_group = conversation.Group,
            page = 1,
            limit,
            last_id = beforeId
        });

    public Task UploadAttachmentAsync(string path, Conversation conversation, string id, string? type = null) =>
        ImgoApi.UploadChatAttachmentAsync(Server, token, path, conversation, id, Cancellation, type);

    public Task<JsonElement> UploadAsync(string path, string action) =>
        ImgoApi.UploadAsync(Server, token, path, action, Cancellation);

    public async Task CopyAttachmentAsync(string location, Stream output, long? maxBytes = null,
        CancellationToken cancellation = default)
    {
        using var linked = CancellationTokenSource.CreateLinkedTokenSource(Cancellation, cancellation);
        await ImgoApi.CopyAttachmentAsync(Server, token, location, output, maxBytes, linked.Token);
    }

    public void Stop() => lifetime.Cancel();

    public void Expire()
    {
        Expired = true;
        ConnectionStatus = "登录已失效，请退出后重新登录";
        Stop();
    }

    public event Action<AccountSession, JsonElement>? CallSignal;

    public async Task ListenAsync()
    {
        var address = new UriBuilder(Server) { Scheme = Server.Scheme == "https" ? "wss" : "ws", Path = "/wss" };
        var delay = TimeSpan.FromSeconds(2);
        while (!Cancellation.IsCancellationRequested)
        {
            using var socket = new ClientWebSocket();
            using var connection = CancellationTokenSource.CreateLinkedTokenSource(Cancellation);
            try
            {
                using var connect = CancellationTokenSource.CreateLinkedTokenSource(connection.Token);
                connect.CancelAfter(TimeSpan.FromSeconds(12));
                await socket.ConnectAsync(address.Uri, connect.Token);
                await SendAsync(socket, new SocketCommand("bindUid", token), connect.Token);
                await SendAsync(socket, new SocketCommand("ping"), connect.Token);
                var heartbeat = HeartbeatAsync(socket, connection);
                try
                {
                    while (!connection.IsCancellationRequested)
                    {
                        using var message = await ReceiveAsync(socket, connection.Token);
                        var root = message.RootElement;
                        var type = ImgoApi.Field(root, "type");
                        if (type == "error" && ImgoApi.Number(root, "code") == 401)
                        {
                            Expire();
                            return;
                        }
                        if (type == "pong" && ConnectionStatus != "实时在线")
                        {
                            ConnectionStatus = "实时在线";
                            Interlocked.Increment(ref historyRevision);
                            delay = TimeSpan.FromSeconds(2);
                            Interlocked.Increment(ref revision);
                        }
                        else if (type == "webrtc")
                        {
                            var signal = ImgoApi.Property(root, "data");
                            var callEvent = ImgoApi.Field(ImgoApi.Property(signal, "extends"), "event");
                            if (callEvent is "calling" or "hangup" or "otherOpt")
                                InvalidateHistory();
                            if (signal.ValueKind == JsonValueKind.Object)
                                CallSignal?.Invoke(this, signal.Clone());
                        }
                        else if (type == "groupFriendAddPermission")
                            CanAddGroupFriends = ImgoApi.Number(ImgoApi.Property(root, "data"), "group_friend_add") == 1;
                        else if (type is "simple" or "group" or "addGroup" or "removeGroup" or "removeUser" or "appendContact" or "undoMessage" or "delMessage" or "clearMessage" or "adminNotice" or "setChatTop" or "setIsNotice" or "friendApply" or "readAll" or "groupSetting" or "setManager" or "editGroupName" or "editGroupAvatar" or "setNotice" or "setNoSpeak" or "changeOwner" or "userIPChanged")
                        {
                            if (type is "removeGroup" or "removeUser" or "undoMessage" or "delMessage" or "clearMessage")
                                Interlocked.Increment(ref historyRevision);
                            Interlocked.Increment(ref revision);
                        }
                    }
                }
                finally
                {
                    connection.Cancel();
                    await heartbeat;
                }
            }
            catch (Exception exception) when (exception is OperationCanceledException or WebSocketException or IOException or JsonException)
            {
                if (Cancellation.IsCancellationRequested)
                    return;
            }

            ConnectionStatus = "正在重连";
            try { await Task.Delay(delay, Cancellation); }
            catch (OperationCanceledException) { return; }
            delay = TimeSpan.FromSeconds(Math.Min(delay.TotalSeconds * 2, 30));
        }
    }

    private sealed record SocketCommand(
        [property: JsonPropertyName("type")] string Type,
        [property: JsonPropertyName("token"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
        string? Token = null);

    private static Task SendAsync(ClientWebSocket socket, SocketCommand command, CancellationToken cancellation) =>
        socket.SendAsync(new ArraySegment<byte>(JsonSerializer.SerializeToUtf8Bytes(command)),
            WebSocketMessageType.Text, true, cancellation);

    private static async Task HeartbeatAsync(ClientWebSocket socket, CancellationTokenSource connection)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(25));
        try
        {
            while (await timer.WaitForNextTickAsync(connection.Token))
                await SendAsync(socket, new SocketCommand("ping"), connection.Token);
        }
        catch (Exception exception) when (exception is OperationCanceledException or WebSocketException or IOException)
        {
            connection.Cancel();
        }
    }

    private static async Task<JsonDocument> ReceiveAsync(ClientWebSocket socket, CancellationToken cancellation)
    {
        using var timeout = CancellationTokenSource.CreateLinkedTokenSource(cancellation);
        timeout.CancelAfter(TimeSpan.FromSeconds(65));
        using var stream = new MemoryStream();
        var buffer = new byte[8192];
        WebSocketReceiveResult result;
        do
        {
            result = await socket.ReceiveAsync(new ArraySegment<byte>(buffer), timeout.Token);
            if (result.MessageType != WebSocketMessageType.Text || stream.Length + result.Count > 1024 * 1024)
                throw new IOException("WebSocket 已关闭或收到无效数据。");
            stream.Write(buffer, 0, result.Count);
        }
        while (!result.EndOfMessage);

        stream.Position = 0;
        return await JsonDocument.ParseAsync(stream, cancellationToken: timeout.Token);
    }
}
