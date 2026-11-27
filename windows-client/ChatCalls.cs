using System.IO;
using System.Runtime.InteropServices;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;
using Microsoft.Web.WebView2.Core;
using Microsoft.Web.WebView2.Wpf;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private Call? activeCall;
    private const string CallPage = "https://imgo-call.invalid/index.html";
    private const string CallRoute = "/enterprise/im/sendToMsg";

    private sealed class Call(AccountSession session, long target, string name, bool video, bool incoming, string id, long messageId, bool receiveOnlyVideo)
    {
        public AccountSession Session { get; } = session;
        public long Target { get; } = target;
        public string Name { get; } = name;
        public bool Video { get; } = video;
        public bool Incoming { get; } = incoming;
        public PreparedVideo? Prepared { get; init; }
        public bool Recorded => receiveOnlyVideo || Prepared is not null;
        public bool Camera => Video && !Recorded;
        public CancellationTokenSource MediaCancellation { get; } = new();
        public string? StagedMedia { get; set; }
        public string Id { get; } = id;
        public long MessageId { get; set; } = messageId;
        public DateTimeOffset Deadline { get; set; } = DateTimeOffset.UtcNow.AddSeconds(40);
        public DateTimeOffset? ConnectedAt { get; set; }
        public bool Starting { get; set; }
        public bool Ready { get; set; }
        public bool Accepted { get; set; }
        public bool Ending { get; set; }
        public bool CloseAppAfterEnd { get; set; }
        public bool Mute { get; set; }
        public bool CameraOff { get; set; }
        public bool UpdatingCameraMenu { get; set; }
        public string CameraId { get; set; } = "";
        public Task<JsonElement>? Invitation { get; set; }
        public Window Window { get; set; } = null!;
        public TextBlock Status { get; set; } = null!;
        public Button Accept { get; set; } = null!;
        public Button End { get; set; } = null!;
        public Button Mic { get; set; } = null!;
        public Button VideoToggle { get; set; } = null!;
        public ComboBox Cameras { get; set; } = null!;
        public TextBlock CameraStatus { get; set; } = null!;
        public Button Playback { get; set; } = null!;
        public WebView2 View { get; } = new();
        public List<object> Pending { get; } = [];
        public SemaphoreSlim SignalGate { get; } = new(1);
        public int EndCode => ConnectedAt.HasValue ? 906 : Incoming && !Accepted ? 903 : Accepted ? 905 : 902;
    }

    private sealed record CallCamera(string Id, string Label);

    private async Task RefreshCallAvailability(AccountSession session)
    {
        try
        {
            var info = await session.PostAsync("/common/pub/getSystemInfo", new { });
            session.CallsEnabled = ImgoApi.Number(ImgoApi.Property(info, "chatInfo"), "webrtc") == 1;
            if (visibleAccount == session) UpdateButtons();
        }
        catch (Exception error) when (IsRequestFailure(error)) { }
    }

    private long PrivateCallTarget() =>
        visibleConversation is { Group: 0 } conversation && long.TryParse(conversation.Id, out var id) &&
        id > 0 && id != visibleAccount?.UserId ? id : 0;

    private void StartVoiceCall(object sender, RoutedEventArgs e) => StartOutgoingCall(false);

    private void StartOutgoingCall(bool video, PreparedVideo? prepared = null)
    {
        var target = PrivateCallTarget();
        if (activeCall is not null || ActiveChatAccount is not { CallsEnabled: true } session || target == 0) return;
        var call = new Call(session, target, visibleConversation!.Name, video, false, Guid.NewGuid().ToString("N"), 0, false) { Prepared = prepared };
        ShowCall(call);
        _ = StartCallMedia(call);
    }

    private void ReceiveCallSignal(AccountSession session, JsonElement packet)
    {
        // Keep the socket reader independent of call setup and HTTP signaling.
        _ = Dispatcher.InvokeAsync(() => { _ = HandleCallSignal(session, packet); });
    }

    private async Task HandleCallSignal(AccountSession session, JsonElement packet)
    {
        if (session.Expired || session.SigningOut || !accounts.Contains(session)) return;
        var ext = ImgoApi.Property(packet, "extends");
        var action = ImgoApi.Field(ext, "event");
        var sender = ImgoApi.Property(packet, "fromUser");
        var fromId = ImgoApi.Number(sender, "user_id");
        var messageId = ImgoApi.Number(packet, "msg_id");
        var id = ImgoApi.Field(packet, "id");
        if (fromId <= 0 || id.Length == 0) return;

        var call = activeCall;
        if (action == "calling" && fromId != session.UserId && messageId > 0)
        {
            if (call is not null)
            {
                if (call.Session == session && call.Target == fromId && call.MessageId == messageId) return;
                try
                {
                    await session.PostAsync(CallRoute, new
                    {
                        id, msg_id = messageId, toContactId = fromId,
                        type = ImgoApi.Number(ext, "type"), @event = "hangup", code = 907
                    });
                }
                catch (Exception error) when (IsRequestFailure(error)) { }
                return;
            }
            var name = ImgoApi.Field(sender, "realname");
            if (name.Length == 0) name = ImgoApi.Field(sender, "account");
            if (name.Length == 0) name = fromId.ToString();
            ShowCall(new Call(session, fromId, name, ImgoApi.Number(ext, "type") == 1, true, id, messageId,
                ImgoApi.Field(ext, "video_mode") == "recorded"));
            System.Media.SystemSounds.Asterisk.Play();
            return;
        }

        if (call is null || call.Session != session || call.Id != id || call.Ending ||
            (call.MessageId > 0 && call.MessageId != messageId)) return;
        if (fromId == session.UserId)
        {
            if (action == "otherOpt" && call.Incoming && (!call.Accepted || ImgoApi.Number(ext, "code") != 904))
                CloseCall(call, "已在其他端处理");
            else if (action == "busy" && !call.Incoming)
                CloseCall(call, "对方不在线");
            return;
        }
        if (call.Target != fromId || messageId <= 0) return;
        call.MessageId = messageId;

        switch (action)
        {
            case "acceptRtc" when !call.Incoming && !call.Accepted:
                call.Accepted = true;
                SetConnecting(call);
                SendToMedia(call, new { kind = "makeOffer" });
                break;
            case "hangup":
                CloseCall(call, ImgoApi.Field(packet, "content"));
                break;
            case "offer" when call.Incoming && call.Accepted:
            case "answer" when !call.Incoming && call.Accepted:
                var sdp = ImgoApi.Field(ext, "sdp");
                if (sdp.Length is > 0 and <= 262144) SendToMedia(call, new { kind = action, sdp });
                break;
            case "candidate":
            case "iceCandidate":
                var candidate = ImgoApi.Property(ext, "iceCandidate");
                if (candidate.ValueKind is JsonValueKind.String or JsonValueKind.Object)
                    SendToMedia(call, new { kind = "iceCandidate", candidate });
                break;
        }
    }

    private void ShowCall(Call call)
    {
        activeCall = call;
        UpdateButtons();
        var label = call.Recorded ? "预录视频通话" : call.Video ? "视频通话" : "语音通话";
        var heading = new TextBlock
        {
            Text = call.Name, FontSize = 22, FontWeight = FontWeights.SemiBold, TextTrimming = TextTrimming.CharacterEllipsis
        };
        call.Status = new TextBlock
        {
            Text = call.Incoming ? $"来电 · {label} · {call.Session.Account}" : "正在准备麦克风和摄像头…",
            Foreground = (Brush)FindResource("ChatMuted"), TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 6, 0, 0)
        };
        if (!call.Incoming && !call.Video) call.Status.Text = "正在准备麦克风…";
        var panel = new Grid { Margin = new Thickness(22) };
        panel.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        panel.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
        panel.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        var details = new StackPanel();
        details.Children.Add(heading);
        details.Children.Add(call.Status);
        call.CameraStatus = new TextBlock
        {
            Foreground = (Brush)FindResource("ChatMuted"), TextWrapping = TextWrapping.Wrap,
            Visibility = Visibility.Collapsed, Margin = new Thickness(0, 5, 0, 0)
        };
        details.Children.Add(call.CameraStatus);
        panel.Children.Add(details);
        if (call.Video) call.View.Margin = new Thickness(0, 16, 0, 0);
        else
        {
            call.View.Width = call.View.Height = 1;
            call.View.IsHitTestVisible = false;
        }
        call.View.Visibility = Visibility.Hidden;
        Grid.SetRow(call.View, 1);
        panel.Children.Add(call.View);

        var buttons = new WrapPanel { HorizontalAlignment = HorizontalAlignment.Right, Margin = new Thickness(0, 12, 0, 0) };
        call.Mic = AddCallButton(buttons, "静音", () =>
        {
            call.Mute = !call.Mute;
            call.Mic.Content = call.Mute ? "取消静音" : call.Mic.Tag;
            SendToMedia(call, new { kind = "toggleTrack", track = "audio", enabled = !call.Mute });
        }, false);
        call.Cameras = new ComboBox
        {
            DisplayMemberPath = nameof(CallCamera.Label), Width = 146, Margin = new Thickness(6, 4, 0, 0),
            Visibility = Visibility.Collapsed, ToolTip = "选择通话摄像头"
        };
        System.Windows.Automation.AutomationProperties.SetName(call.Cameras, "选择通话摄像头");
        call.Cameras.SelectionChanged += (_, _) =>
        {
            if (activeCall == call && !call.UpdatingCameraMenu && call.Cameras.SelectedItem is CallCamera camera &&
                (camera.Id != call.CameraId || call.CameraOff))
                UpdateCallCamera(call, true, camera.Id);
        };
        buttons.Children.Add(call.Cameras);
        call.VideoToggle = AddCallButton(buttons, "关闭摄像头", () => UpdateCallCamera(call, call.CameraOff, call.CameraId), false);
        call.Playback = AddCallButton(buttons, "启用声音", () => SendToMedia(call, new { kind = "play" }), false);
        call.Accept = AddCallButton(buttons, "接听", () => _ = StartCallMedia(call), call.Incoming, primary: true);
        call.End = AddCallButton(buttons, call.Incoming ? "拒绝" : "取消", () => _ = EndCall(call, call.EndCode), true);
        Grid.SetRow(buttons, 2);
        panel.Children.Add(buttons);

        call.Window = new Window
        {
            Owner = this, Title = $"IMGO · {label}", Content = panel, ShowInTaskbar = false,
            Width = call.Video ? 620 : 400, Height = call.Video ? 490 : 195, MinWidth = 390, MinHeight = 185,
            WindowStartupLocation = WindowStartupLocation.CenterOwner
        };
        call.Window.Closing += (_, args) =>
        {
            if (activeCall != call) return;
            args.Cancel = true;
            _ = Dispatcher.InvokeAsync(() => { _ = EndCall(call, call.EndCode); });
        };
        if (call.Incoming && WindowState == WindowState.Minimized) WindowState = WindowState.Normal;
        call.Window.Show();
    }

    private Button AddCallButton(Panel panel, string label, Action perform, bool show, bool primary = false)
    {
        var button = new Button
        {
            Content = label, Margin = new Thickness(6, 4, 0, 0),
            Visibility = show ? Visibility.Visible : Visibility.Collapsed
        };
        if (primary) button.Style = (Style)FindResource("ChatPrimaryAction");
        button.Click += (_, _) => perform();
        panel.Children.Add(button);
        return button;
    }

    private async Task StartCallMedia(Call call)
    {
        if (activeCall != call || call.Starting || call.Ending) return;
        call.Starting = true;
        call.Accept.IsEnabled = false;
        call.Status.Text = call.Camera ? "正在打开麦克风和摄像头…" : "正在打开麦克风…";
        try
        {
            var info = await call.Session.PostAsync("/common/pub/getSystemInfo", new { });
            if (activeCall != call || call.Ending) return;
            var settings = ImgoApi.Property(info, "chatInfo");
            if (!call.Incoming && ImgoApi.Number(settings, "webrtc") != 1)
            {
                call.Session.CallsEnabled = false;
                CloseCall(call, "服务器未开启音视频通话");
                return;
            }

            var iceServers = new List<object> { new { urls = "stun:stun.l.google.com:19302" } };
            var relay = ImgoApi.Field(settings, "stun").Trim();
            if (relay.StartsWith("stun:", StringComparison.OrdinalIgnoreCase) || relay.StartsWith("stuns:", StringComparison.OrdinalIgnoreCase))
                iceServers.Add(new { urls = relay });
            else if (relay.StartsWith("turn:", StringComparison.OrdinalIgnoreCase) || relay.StartsWith("turns:", StringComparison.OrdinalIgnoreCase))
                iceServers.Add(new { urls = relay, username = ImgoApi.Field(settings, "stunUser"), credential = ImgoApi.Field(settings, "stunPass") });

            if (call.Prepared is not null)
            {
                call.Status.Text = "正在读取预录视频…";
                call.Deadline = DateTimeOffset.UtcNow.AddMinutes(5);
                var folder = await StagePreparedVideo(call, ImgoApi.Number(ImgoApi.Property(info, "fileUpload"), "videoSize", 200));
                if (activeCall != call || call.Ending)
                {
                    await RemovePreparedVideo(folder);
                    return;
                }
                call.StagedMedia = folder;
                call.Deadline = DateTimeOffset.UtcNow.AddSeconds(40);
            }

            var data = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "Imgo", "CallMedia");
            var environment = await CoreWebView2Environment.CreateAsync(userDataFolder: data,
                options: new CoreWebView2EnvironmentOptions("--autoplay-policy=no-user-gesture-required"));
            if (activeCall != call || call.Ending) return;
            call.View.Visibility = Visibility.Visible;
            await call.View.EnsureCoreWebView2Async(environment);
            if (activeCall != call || call.Ending) return;
            var view = call.View.CoreWebView2;
            view.Settings.AreDevToolsEnabled = false;
            view.Settings.AreDefaultContextMenusEnabled = false;
            view.Settings.AreHostObjectsAllowed = false;
            view.NavigationStarting += (_, args) => args.Cancel = args.Uri != CallPage;
            view.NewWindowRequested += (_, args) => args.Handled = true;
            view.PermissionRequested += (_, args) =>
            {
                args.SavesInProfile = false;
                args.State = activeCall == call && !call.Ending && Uri.TryCreate(args.Uri, UriKind.Absolute, out var origin) &&
                    origin.Scheme == "https" && origin.Host == "imgo-call.invalid" && origin.IsDefaultPort &&
                    (args.PermissionKind == CoreWebView2PermissionKind.Microphone ||
                     args.PermissionKind == CoreWebView2PermissionKind.Camera && call.Camera)
                    ? CoreWebView2PermissionState.Allow : CoreWebView2PermissionState.Deny;
            };
            view.ProcessFailed += (_, _) => { _ = EndCall(call, call.EndCode, "通话组件已退出"); };
            view.SetVirtualHostNameToFolderMapping("imgo-call.invalid", call.StagedMedia ?? Path.Combine(AppContext.BaseDirectory, "CallMedia"),
                CoreWebView2HostResourceAccessKind.Deny);
            view.WebMessageReceived += async (_, args) =>
            {
                if (args.Source != CallPage || activeCall != call || call.Ending) return;
                try
                {
                    using var message = JsonDocument.Parse(args.WebMessageAsJson);
                    await HandleMediaEvent(call, message.RootElement, iceServers);
                }
                catch (Exception error) when (IsRequestFailure(error) || error is InvalidOperationException or COMException)
                {
                    await EndCall(call, call.EndCode, "通话信令失败 · " + ErrorText(error));
                }
            };
            view.Navigate(CallPage);
        }
        catch (WebView2RuntimeNotFoundException)
        {
            await EndCall(call, call.EndCode, "系统缺少 Microsoft Edge WebView2 Runtime，无法启动通话");
        }
        catch (Exception error) when (IsRequestFailure(error) || error is InvalidOperationException or COMException or IOException or UnauthorizedAccessException)
        {
            await EndCall(call, call.EndCode, "无法启动通话 · " + ErrorText(error));
        }
    }

    private async Task HandleMediaEvent(Call call, JsonElement message, List<object> iceServers)
    {
        switch (ImgoApi.Field(message, "kind"))
        {
            case "ready":
                SendToMedia(call, new
                {
                    kind = "start", camera = call.Camera, iceServers,
                    prepared = call.Prepared is { } source ? "prepared." + source.Extension : null
                }, force: true);
                break;
            case "mediaReady" when !call.Ready:
                call.Ready = true;
                call.Mic.Content = ImgoApi.Field(message, "audioSource") == "recorded" ? "静音视频" : "静音";
                call.Mic.Tag = call.Mic.Content;
                call.Mic.Visibility = Visibility.Visible;
                call.VideoToggle.Visibility = call.Camera ? Visibility.Visible : Visibility.Collapsed;
                if (call.Incoming)
                {
                    call.Accept.Visibility = Visibility.Collapsed;
                    call.Accepted = true;
                    SetConnecting(call);
                    await SendCallSignal(call, "acceptRtc", 904);
                }
                else
                {
                    call.Status.Text = "正在呼叫，等待对方接听…";
                    call.Deadline = DateTimeOffset.UtcNow.AddSeconds(40);
                    call.Invitation = SendCallSignal(call, "calling", 901);
                    var invitation = await call.Invitation;
                    if (activeCall != call || call.Ending) return;
                    var msgId = ImgoApi.Number(invitation, "msg_id");
                    if (msgId == 0)
                    {
                        CloseCall(call, "对方不在线");
                        return;
                    }
                    call.MessageId = msgId;
                }
                foreach (var pending in call.Pending) SendToMedia(call, pending);
                call.Pending.Clear();
                break;
            case "offer":
            case "answer":
                await SendCallSignal(call, ImgoApi.Field(message, "kind"), sdp: ImgoApi.Field(message, "sdp"));
                break;
            case "iceCandidate":
                await SendCallSignal(call, "iceCandidate", candidate: ImgoApi.Field(message, "candidate"));
                break;
            case "state":
                switch (ImgoApi.Field(message, "state"))
                {
                    case "connected":
                        call.ConnectedAt ??= DateTimeOffset.UtcNow;
                        call.Deadline = DateTimeOffset.MaxValue;
                        call.End.Content = "挂断";
                        break;
                    case "disconnected":
                        call.Deadline = DateTimeOffset.UtcNow.AddSeconds(15);
                        call.Status.Text = "连接中断，正在恢复…";
                        break;
                    case "failed":
                        await EndCall(call, call.EndCode, "音视频连接失败，请重试");
                        break;
                }
                break;
            case "cameras" when call.Camera:
                var devices = ImgoApi.Property(message, "devices");
                if (devices.ValueKind != JsonValueKind.Array) break;
                call.CameraId = ImgoApi.Field(message, "deviceId");
                call.UpdatingCameraMenu = true;
                try
                {
                    call.Cameras.ItemsSource = devices.EnumerateArray()
                        .Select(item => new CallCamera(ImgoApi.Field(item, "id"), ImgoApi.Field(item, "label")))
                        .Where(item => item.Id.Length > 0 && item.Label.Length > 0).ToArray();
                    call.Cameras.SelectedValuePath = nameof(CallCamera.Id);
                    call.Cameras.SelectedValue = call.CameraId;
                    call.Cameras.Visibility = call.Cameras.Items.Count > 1 ? Visibility.Visible : Visibility.Collapsed;
                }
                finally { call.UpdatingCameraMenu = false; }
                break;
            case "cameraChanged" when call.Camera:
                call.CameraOff = ImgoApi.Number(message, "enabled") == 0;
                call.CameraId = ImgoApi.Field(message, "deviceId");
                call.VideoToggle.Content = call.CameraOff ? "打开摄像头" : "关闭摄像头";
                SelectCallCamera(call, call.CameraId);
                call.Cameras.IsEnabled = call.VideoToggle.IsEnabled = true;
                break;
            case "cameraError" when call.Camera:
                SelectCallCamera(call, call.CameraId);
                call.Cameras.IsEnabled = call.VideoToggle.IsEnabled = true;
                call.CameraStatus.Text = "无法打开所选摄像头。请检查设备连接、权限或占用情况。";
                call.CameraStatus.Visibility = Visibility.Visible;
                break;
            case "playing":
                call.Playback.Visibility = Visibility.Collapsed;
                break;
            case "playbackBlocked":
                call.Playback.Visibility = Visibility.Visible;
                break;
            case "error":
                var reason = ImgoApi.Field(message, "name");
                await EndCall(call, call.EndCode, reason switch
                {
                    "NotAllowedError" => "无法访问麦克风或摄像头，请检查 Windows 隐私设置",
                    "NotFoundError" => "找不到麦克风或摄像头",
                    "PreparedVideoError" => "无法播放预录视频，请检查文件格式",
                    "NotReadableError" => "麦克风或摄像头不可用，可能被其他程序占用",
                    _ => "音视频连接失败，请重试"
                });
                break;
        }
    }

    private static void SendToMedia(Call call, object command, bool force = false)
    {
        if (call.Ending) return;
        if (force || call.Ready)
            call.View.CoreWebView2?.PostWebMessageAsJson(JsonSerializer.Serialize(command));
        else if (call.Pending.Count < 256)
            call.Pending.Add(command);
    }

    private void UpdateCallCamera(Call call, bool enabled, string deviceId)
    {
        if (activeCall != call || !call.Ready || call.Ending || !call.Camera) return;
        call.Cameras.IsEnabled = call.VideoToggle.IsEnabled = false;
        call.CameraStatus.Visibility = Visibility.Collapsed;
        SendToMedia(call, new { kind = "selectCamera", enabled, deviceId });
    }

    private static void SelectCallCamera(Call call, string id)
    {
        call.UpdatingCameraMenu = true;
        try { call.Cameras.SelectedValue = id; }
        finally { call.UpdatingCameraMenu = false; }
    }

    private static void SetConnecting(Call call)
    {
        call.Deadline = DateTimeOffset.UtcNow.AddSeconds(35);
        call.Status.Text = "已接听，正在建立连接…";
        call.End.Content = "挂断";
    }

    private static async Task<JsonElement> SendCallSignal(Call call, string action, int code = 0, string? sdp = null, string? candidate = null)
    {
        await call.SignalGate.WaitAsync(call.Session.Cancellation);
        try
        {
            if ((action != "hangup" && call.Ending) || (call.MessageId == 0 && action != "calling")) return default;
            return await call.Session.PostAsync(CallRoute, new
            {
                id = call.Id, msg_id = call.MessageId, toContactId = call.Target, type = call.Video ? 1 : 0,
                @event = action, code, sdp, iceCandidate = candidate,
                video_mode = call.Recorded ? "recorded" : "",
                video_file_id = action == "calling" ? call.Prepared?.Id ?? 0 : 0,
                callTime = (int)(DateTimeOffset.UtcNow - (call.ConnectedAt ?? DateTimeOffset.UtcNow)).TotalSeconds
            });
        }
        finally { call.SignalGate.Release(); }
    }

    private void CheckCall()
    {
        if (activeCall is not { Ending: false } call) return;
        if (call.Session.Expired)
        {
            CloseCall(call, "登录已失效，通话已结束");
            return;
        }
        if (DateTimeOffset.UtcNow >= call.Deadline)
            _ = EndCall(call, call.ConnectedAt.HasValue ? 906 : 905, "通话超时");
        else if (call.ConnectedAt is { } since && call.Deadline == DateTimeOffset.MaxValue)
            call.Status.Text = $"正在通话 · {DateTimeOffset.UtcNow - since:hh\\:mm\\:ss}";
    }

    private async Task EndCall(Call call, int code, string? message = null)
    {
        if (activeCall != call) return;
        if (call.Ending) return;
        call.Ending = true;
        call.MediaCancellation.Cancel();
        call.Window.IsEnabled = false;
        call.Status.Text = "正在结束通话…";
        await System.Windows.Threading.Dispatcher.Yield();
        call.View.Dispose();
        try
        {
            if (call.MessageId == 0 && call.Invitation is not null)
                call.MessageId = ImgoApi.Number(await call.Invitation, "msg_id");
            if (call.MessageId > 0)
                await SendCallSignal(call, "hangup", code);
        }
        catch (Exception error) when (IsRequestFailure(error)) { message ??= "通话已在本机结束，服务器未确认"; }
        CloseCall(call, message ?? "通话已结束");
        if (call.CloseAppAfterEnd) Close();
    }

    private void CloseCall(Call call, string message)
    {
        if (activeCall != call) return;
        activeCall = null;
        call.MediaCancellation.Cancel();
        if (!call.Ending) call.View.Dispose();
        if (call.StagedMedia is { } folder) _ = RemovePreparedVideo(folder);
        call.Window.Close();
        call.Session.InvalidateHistory();
        Status.Text = message.Length > 0 ? message : "通话已结束";
        UpdateButtons();
        ScheduleRefreshes();
    }
}
