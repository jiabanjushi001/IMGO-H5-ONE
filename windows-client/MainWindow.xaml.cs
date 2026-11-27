using System.Collections.ObjectModel;
using System.Net.Http;
using System.IO;
using Microsoft.Win32;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using System.Windows.Threading;

namespace Imgo.Desktop;

public partial class MainWindow : Window
{
    private readonly ObservableCollection<AccountSession> accounts = [];
    private readonly HashSet<AccountSession> refreshing = [];
    private readonly CancellationTokenSource lifetime = new();
    private readonly DispatcherTimer timer = new() { Interval = TimeSpan.FromSeconds(1) };
    private bool changingContactList;
    private AccountSession? visibleAccount;
    private Conversation? visibleConversation;
    private List<ChatMessage> visibleMessages = [];
    private long messageRevision = -1;
    private DateTimeOffset lastMessages;
    private const int HistoryPageSize = 60;
    private long historyRevision;
    private int conversationGeneration;
    private bool hasOlderMessages;
    private bool loadingOlder;
    private bool sending;
    private bool loadingQuickReplies;
    private bool transferringAttachment;

    public MainWindow()
    {
        InitializeComponent();
        SizeChanged += (_, _) => AdjustChatWidth();
        Accounts.ItemsSource = accounts;
        SignOutButton.IsEnabled = false;
        timer.Tick += (_, _) =>
        {
            ScheduleRefreshes();
            CheckCall();
        };
        timer.Start();
        Loaded += (_, _) => AddAccount(this, new RoutedEventArgs());
    }

    private void AddAccount(object sender, RoutedEventArgs e)
    {
        LoginError.Text = "";
        LoginPanel.Visibility = Visibility.Visible;
        if (ServerAddress.Text.Length == 0) ServerAddress.Focus();
        else LoginAccount.Focus();
    }

    private void CancelLogin(object sender, RoutedEventArgs e) => CloseLogin();

    private void CloseLogin()
    {
        LoginPassword.Clear();
        GoogleCode.Clear();
        LoginPanel.Visibility = Visibility.Collapsed;
    }

    private async void Login(object sender, RoutedEventArgs e)
    {
        LoginError.Text = "";
        LoginButton.IsEnabled = CancelLoginButton.IsEnabled = false;
        try
        {
            var server = ImgoApi.ServerUri(ServerAddress.Text);
            var account = LoginAccount.Text.Trim();
            if (account.Length == 0 || LoginPassword.Password.Length == 0)
                throw new ArgumentException("请输入账号和密码。");

            var data = await ImgoApi.PostAsync(server, "/common/pub/login", new
            {
                account,
                password = LoginPassword.Password,
                admin_login = 1,
                google_code = GoogleCode.Text.Trim()
            }, null, lifetime.Token);
            var info = ImgoApi.Property(data, "userInfo");
            var userId = ImgoApi.Number(info, "user_id");
            var token = ImgoApi.Field(data, "authToken");
            if (userId <= 0 || token.Length == 0)
                throw new InvalidOperationException("登录响应缺少会话信息。");

            var session = new AccountSession(server, token, userId, account, ImgoApi.Field(info, "displayName"), info);
            session.EnterToSend = ReadSendShortcut(session, info);
            var previous = accounts.FirstOrDefault(item => item.Server == server && item.UserId == userId);
            if (previous is not null)
            {
                try { await previous.PostAsync("/common/pub/logout", new { }); }
                catch (Exception exception) when (IsRequestFailure(exception)) { }
                previous.Stop();
                accounts.Remove(previous);
            }
            accounts.Add(session);
            Accounts.SelectedItem = session;
            LoginAccount.Clear();
            CloseLogin();
            Status.Text = "已登录，正在读取会话…";
            session.CallSignal += ReceiveCallSignal;
            _ = RefreshCallAvailability(session);
            _ = session.ListenAsync();
            ScheduleRefreshes();
        }
        catch (Exception exception) when (exception is ArgumentException or InvalidOperationException || IsRequestFailure(exception))
        {
            LoginError.Text = ErrorText(exception);
        }
        finally
        {
            LoginButton.IsEnabled = CancelLoginButton.IsEnabled = true;
        }
    }

    private async void SignOut(object sender, RoutedEventArgs e)
    {
        if (visibleAccount is not { } session || session.SigningOut) return;
        session.SigningOut = true;
        if (activeCall is { } call && call.Session == session) await EndCall(call, call.EndCode);
        UpdateButtons();
        try
        {
            if (!session.Expired)
                await session.PostAsync("/common/pub/logout", new { });
        }
        catch (ApiException exception) when (exception.SessionExpired) { }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (visibleAccount == session) Status.Text = ErrorText(exception);
            return;
        }
        finally
        {
            session.SigningOut = false;
            UpdateButtons();
        }

        session.Stop();
        var wasVisible = visibleAccount == session;
        accounts.Remove(session);
        if (wasVisible && accounts.Count > 0) Accounts.SelectedIndex = 0;
    }

    private void AccountChanged(object sender, SelectionChangedEventArgs e)
    {
        SaveDraft();
        CloseConversationPane();
        if (DirectoryEntries is not null) DirectoryEntries.SelectedItem = null;
        visibleAccount = Accounts.SelectedItem as AccountSession;
        visibleConversation = null;
        SetQuote(null);
        NetworkProfileButton.Visibility = Visibility.Collapsed;
        var mayManage = visibleAccount is { } account && AdminWorkspace.CanOpen(account);
        AdminButton.Visibility = mayManage ? Visibility.Visible : Visibility.Collapsed;
        if (!mayManage) SetAdminVisible(false);
        if (Admin.Visibility == Visibility.Visible) Admin.SetAccount(visibleAccount);
        Compose.Clear();
        ClearMessages();
        ApplyContacts(visibleAccount);
        Status.Text = visibleAccount is null ? "添加账号后开始聊天" : "";
        UpdateButtons();
        ScheduleRefreshes();
    }

    private void ToggleAdmin(object sender, RoutedEventArgs e) => SetAdminVisible(Admin.Visibility != Visibility.Visible);

    private void SetAdminVisible(bool isVisible)
    {
        if (isVisible) CloseConversationPane();
        Admin.Visibility = isVisible ? Visibility.Visible : Visibility.Collapsed;
        ChatWorkspace.Visibility = isVisible ? Visibility.Collapsed : Visibility.Visible;
        AdminButton.Content = isVisible ? "进入聊天" : "管理中心";
        Admin.SetAccount(isVisible ? visibleAccount : null);
    }

    private void ConversationFilterChanged(object sender, RoutedEventArgs e)
    {
        if (Contacts is not null && ConversationSearch is not null)
            ApplyContacts(visibleAccount);
    }

    private void ApplyContacts(AccountSession? session)
    {
        List<Conversation> all = session?.Conversations ?? [];
        var key = ConversationSearch.Text.Trim();
        var items = all.Where(item =>
            (OnlyUnread?.IsChecked != true || item.HasUnread) &&
            (OnlyMentions?.IsChecked != true || item.Mentioned) &&
            (key.Length == 0 || item.Name.Contains(key, StringComparison.OrdinalIgnoreCase) ||
                item.Account.Contains(key, StringComparison.OrdinalIgnoreCase) ||
                item.Id.StartsWith("group-", StringComparison.Ordinal) && item.Id.AsSpan(6).SequenceEqual(key.AsSpan()))).ToList();
        var selected = all.Find(item => item.Id == session?.SelectedConversation);
        changingContactList = true;
        try
        {
            Contacts.ItemsSource = items;
            Contacts.SelectedValue = selected?.Id;
        }
        finally { changingContactList = false; }
        ConversationEmpty.Visibility = items.Count == 0 ? Visibility.Visible : Visibility.Collapsed;
        ConversationEmpty.Text = session is null ? "添加账号后查看会话" :
            key.Length > 0 ? "没有匹配的会话" : OnlyUnread.IsChecked == true ? "没有未读会话" :
            OnlyMentions.IsChecked == true ? "没有提及你的会话" : "暂无会话";
        ShowConversation(selected);
        if (IsDirectoryVisible) RefreshDirectory();
    }

    private void ContactChanged(object sender, SelectionChangedEventArgs e)
    {
        if (changingContactList) return;
        ShowConversation(Contacts.SelectedItem as Conversation);
        ScheduleRefreshes();
    }

    private void ShowConversation(Conversation? conversation)
    {
        if (visibleConversation?.Id != conversation?.Id)
        {
            SaveDraft();
            CloseConversationPane();
            visibleConversation = conversation;
            SetQuote(null);
            NetworkProfileButton.Visibility = Visibility.Collapsed;
            ClearMessages();
            Compose.Text = visibleAccount is { } account && conversation is not null
                ? account.Drafts.GetValueOrDefault(conversation.Id, "") : "";
        }
        visibleConversation = conversation;
        if (visibleAccount is { } session) session.SelectedConversation = conversation?.Id ?? "";
        ChatTitle.Text = conversation is { Group: 0, Account.Length: > 0 } person && person.Account != person.Name
            ? $"{person.Account}（{person.Name}）" : conversation?.Name ?? "聊天";
        ChatInitial.Text = conversation?.Initial ?? "";
        if (!Equals(ChatAvatarImage.DataContext, conversation) || conversation is null) ChatAvatarImage.DataContext = conversation;
        ChatKind.ToolTip = null;
        ChatKind.Text = conversation is { Group: 0 } contact ? (contact.Online ? "在线" : "离线") : conversation?.Kind ?? "";
        if (conversation?.Group == 1) ChatKind.Text = $"群聊 · 群号 {conversation.Id.Replace("group-", "", StringComparison.Ordinal)}";
        if (NetworkProfileButton.Visibility == Visibility.Visible) ChatKind.Text = networkChatSummary;
        ChatAvatar.Visibility = ChatKind.Visibility = conversation is null ? Visibility.Collapsed : Visibility.Visible;
        UpdateChatEmpty();
        UpdateButtons();
    }

    private void SaveDraft()
    {
        if (visibleAccount is { } session && visibleConversation is { } conversation)
            session.Drafts[conversation.Id] = Compose.Text;
    }

    private void ClearMessages()
    {
        conversationGeneration++;
        messageRevision = -1;
        historyRevision = visibleAccount?.HistoryRevision ?? 0;
        hasOlderMessages = loadingOlder = false;
        visibleMessages = [];
        Messages.ItemsSource = null;
        UpdateChatEmpty();
    }

    private void UpdateButtons()
    {
        var canCompose = visibleAccount is { Expired: false, SigningOut: false } && visibleConversation is { Group: not 2 };
        Compose.IsEnabled = canCompose && !sending;
        SendButton.IsEnabled = Compose.IsEnabled && !string.IsNullOrWhiteSpace(Compose.Text);
        QuickReplyButton.IsEnabled = Compose.IsEnabled && !loadingQuickReplies;
        EditorTools.IsEnabled = ActiveChatAccount is not null && visibleConversation is not null;
        foreach (var tool in EditorTools.Children.OfType<WrapPanel>())
            tool.IsEnabled = canCompose && !transferringAttachment;
        ContactsMenuButton.IsEnabled = ActiveChatAccount is not null;
        DirectoryAddButton.IsEnabled = ActiveChatAccount is not null;
        DirectoryRailButton.IsEnabled = FilesRailButton.IsEnabled = ProfileRailButton.IsEnabled = ActiveChatAccount is not null;
        GroupSettingsButton.Visibility = visibleConversation?.Group == 1 && ConversationSidePane.Visibility == Visibility.Collapsed
            ? Visibility.Visible : Visibility.Collapsed;
        GroupSettingsButton.IsEnabled = GroupNoticeButton.IsEnabled = ContactsMenuButton.IsEnabled;
        GroupNoticeButton.Visibility = GroupSettingsButton.Visibility;
        ConversationMenuButton.Visibility = visibleConversation is { Group: 0 or 1 } ? Visibility.Visible : Visibility.Collapsed;
        ConversationMenuButton.IsEnabled = ContactsMenuButton.IsEnabled;
        var showCalls = visibleAccount is { CallsEnabled: true } && PrivateCallTarget() > 0;
        VoiceCallButton.Visibility = VideoCallButton.Visibility = showCalls ? Visibility.Visible : Visibility.Collapsed;
        VoiceCallButton.IsEnabled = VideoCallButton.IsEnabled =
            ActiveChatAccount is not null && activeCall is null && visibleAccount?.ConnectionStatus == "实时在线";
        MentionButton.Visibility = visibleConversation?.Group == 1 ? Visibility.Visible : Visibility.Collapsed;
        MentionButton.IsEnabled = Compose.IsEnabled;
        DirectoryAddButton.Content = ContactsMenuButton.Content = visibleAccount is { FriendRequests: > 0 } current
            ? $"＋ · {current.FriendRequests}" : "＋";
        ComposeHint.Text = visibleConversation?.Group == 2 ? "系统公告为只读" :
            visibleConversation is null ? "" : visibleAccount?.EnterToSend == true ? "Enter 发送 · Ctrl + Enter 换行" : "Ctrl + Enter 发送";
        SignOutButton.IsEnabled = visibleAccount is { SigningOut: false };
        Connection.Text = visibleAccount?.ConnectionStatus ?? "";
        OlderButton.Visibility = visibleConversation is null ? Visibility.Collapsed : Visibility.Visible;
        OlderButton.IsEnabled = hasOlderMessages && !loadingOlder && visibleAccount is { Expired: false, SigningOut: false };
        RefreshOlderButtonLabel(ActualWidth < 1050 && ConversationSidePane.Visibility == Visibility.Visible);
        var unread = accounts.Where(session => !session.Expired).Sum(session => session.Unread);
        var otherUnread = unread - (visibleAccount is { Expired: false } active ? active.Unread : 0);
        Title = unread > 0 ? $"IMGO（{unread} 条未读）" : "IMGO";
        TaskbarItemInfo.Description = Title;
        TaskbarItemInfo.Overlay = unread > 0 ? (ImageSource)FindResource("UnreadIndicator") : null;
        OtherUnread.Text = otherUnread > 0 ? $"其他账号 · {otherUnread} 条未读" : "";
    }

    private void ScheduleRefreshes()
    {
        UpdateButtons();
        foreach (var session in accounts.ToArray())
        {
            if (!session.Expired && !session.SigningOut && DateTimeOffset.UtcNow >= session.RefreshAfter && refreshing.Add(session))
                _ = RefreshAsync(session);
        }
    }

    private async Task RefreshAsync(AccountSession session)
    {
        var generation = conversationGeneration;
        try
        {
            var now = DateTimeOffset.UtcNow;
            var revision = session.Revision;
            var conversation = ChatWorkspace.Visibility == Visibility.Visible && visibleAccount == session && IsActive && WindowState != WindowState.Minimized
                && !IsDirectoryVisible && LoginPanel.Visibility != Visibility.Visible ? visibleConversation : null;
            var contactAge = now - session.LastContacts;
            var refreshContacts = (session.LastContactRevision != revision && contactAge >= TimeSpan.FromSeconds(2))
                || contactAge >= TimeSpan.FromSeconds(45);
            if (conversation is not null && (messageRevision == -1 ||
                (messageRevision != revision && now - lastMessages >= TimeSpan.FromSeconds(2)) ||
                now - lastMessages >= TimeSpan.FromSeconds(45)))
            {
                await LoadMessagesAsync(session, conversation, revision);
                refreshContacts = true;
                await RefreshNetworkProfileAsync(session, conversation);
            }
            if (refreshContacts)
            {
                await LoadContactsAsync(session, revision);
                await LoadFriendRequestCountAsync(session);
            }
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            session.RefreshAfter = DateTimeOffset.UtcNow.AddSeconds(10);
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session && conversationGeneration == generation && exception is ApiException { Code: 403 or 404 })
            {
                ClearMessages();
                await TryReloadContactsAsync(session);
            }
            if (visibleAccount == session && !session.Cancellation.IsCancellationRequested)
                Status.Text = ErrorText(exception);
        }
        finally { refreshing.Remove(session); }
    }

    private async Task TryReloadContactsAsync(AccountSession session)
    {
        try { await LoadContactsAsync(session, session.Revision); }
        catch (Exception exception) when (IsRequestFailure(exception)) { }
    }

    private async Task LoadContactsAsync(AccountSession session, long revision)
    {
        var data = await session.PostAsync("/enterprise/im/getcontacts", new { });
        var conversations = data.EnumerateArray()
            .OrderByDescending(item => ImgoApi.Number(item, "is_top"))
            .ThenByDescending(item => ImgoApi.Number(item, "lastSendTime"))
            .Select(item => new Conversation(ImgoApi.Field(item, "id"), ImgoApi.Field(item, "displayName"),
                ImgoApi.Number(item, "is_group"), ImgoApi.Number(item, "unread"),
                ImgoApi.MessageText(item, "lastContent"), ImgoApi.Field(item, "account"), ImgoApi.Number(item, "lastSendTime"),
                ImgoApi.Number(item, "is_at") > 0, ImgoApi.Number(item, "is_top") == 1,
                ImgoApi.Number(item, "is_notice", 1) == 0, ImgoApi.Number(item, "role"),
                ImgoApi.Number(ImgoApi.Property(item, "setting"), "manager_invite") == 1,
                ImgoApi.AvatarUri(session.Server, ImgoApi.Field(item, "avatar")),
                ImgoApi.Property(item, "is_online").ValueKind == JsonValueKind.True || ImgoApi.Number(item, "is_online") > 0)).ToList();
        session.LastContacts = DateTimeOffset.UtcNow;
        session.LastContactRevision = revision;
        if (session.Conversations.SequenceEqual(conversations) && !(visibleAccount == session && Status.Text == "已登录，正在读取会话…")) return;
        session.Conversations = conversations;
        session.UpdateUnread();
        if (visibleAccount == session)
        {
            ApplyContacts(session);
            if (Status.Text == "已登录，正在读取会话…") Status.Text = "";
        }
        UpdateButtons();
    }

    private async Task LoadMessagesAsync(AccountSession session, Conversation conversation, long revision)
    {
        var generation = conversationGeneration;
        var serverHistoryRevision = session.HistoryRevision;
        var messages = await FetchMessagesAsync(session, conversation, 0);
        if (conversationGeneration != generation || session.HistoryRevision != serverHistoryRevision) return;

        if (historyRevision != serverHistoryRevision)
        {
            ClearMessages();
            historyRevision = serverHistoryRevision;
        }
        List<ChatMessage> earlier = [];
        if (messageRevision == -1) hasOlderMessages = messages.Count == HistoryPageSize;
        else if (visibleMessages.Count > 0 && messages.Count == HistoryPageSize &&
                 messages[0].Id > visibleMessages[^1].Id)
        {
            // The latest page no longer overlaps what was visible. Start again from a contiguous page.
            hasOlderMessages = true;
        }
        else
        {
            var firstId = messages.FirstOrDefault()?.Id ?? 0;
            earlier = visibleMessages.Where(message => message.Id < firstId).ToList();
        }
        messageRevision = revision;
        lastMessages = DateTimeOffset.UtcNow;
        UpdateChatEmpty();
        var followLatest = IsAtBottom();
        RenderMessages([.. earlier, .. messages], followLatest);
        UpdateButtons();
    }

    private async void LoadOlder(object sender, RoutedEventArgs e)
    {
        if (!OlderButton.IsEnabled || visibleAccount is not { } session ||
            visibleConversation is not { } conversation || visibleMessages.Count == 0) return;

        var generation = conversationGeneration;
        var firstMessage = visibleMessages[0];
        var serverHistoryRevision = session.HistoryRevision;
        loadingOlder = true;
        UpdateButtons();
        try
        {
            var page = await FetchMessagesAsync(session, conversation, firstMessage.Id);
            if (conversationGeneration != generation || historyRevision != serverHistoryRevision ||
                session.HistoryRevision != serverHistoryRevision) return;
            var currentFirstId = visibleMessages.FirstOrDefault()?.Id ?? 0;
            if (currentFirstId != firstMessage.Id) return;
            hasOlderMessages = page.Count == HistoryPageSize;
            RenderMessages([.. page, .. visibleMessages], false, preserveOffset: false);
            if (Messages.Template?.FindName("PART_ScrollViewer", Messages) is ScrollViewer scroll)
            {
                var current = visibleMessages;
                var item = current.First(message => message.Id == firstMessage.Id);
                Messages.ScrollIntoView(item);
                _ = Dispatcher.BeginInvoke(() =>
                {
                    if (!ReferenceEquals(visibleMessages, current) ||
                        Messages.ItemContainerGenerator.ContainerFromItem(item) is not FrameworkElement row) return;
                    var position = row.TranslatePoint(new Point(0, 0), scroll).Y;
                    scroll.ScrollToVerticalOffset(Math.Max(0, scroll.VerticalOffset + position));
                }, DispatcherPriority.Loaded);
            }
            Status.Text = hasOlderMessages ? "" : "已到最早消息";
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (conversationGeneration == generation)
            {
                if (exception is ApiException { Code: 403 or 404 }) ClearMessages();
                Status.Text = ErrorText(exception);
            }
        }
        finally
        {
            if (conversationGeneration == generation) loadingOlder = false;
            UpdateButtons();
        }
    }

    private static async Task<List<ChatMessage>> FetchMessagesAsync(
        AccountSession session, Conversation conversation, long beforeId)
    {
        var data = await session.GetMessagesAsync(conversation, HistoryPageSize, beforeId);
        return data.EnumerateArray().Select(item => ReadChatMessage(item, session, conversation.Group)).ToList();
    }

    private static ChatMessage ReadChatMessage(JsonElement item, AccountSession session, long group)
    {
        var senderId = ImgoApi.Number(item, "from_user");
        var mine = senderId == session.UserId;
        var user = ImgoApi.Property(item, "fromUser");
        var name = ImgoApi.Field(user, "displayName");
        if (name.Length == 0) name = ImgoApi.Field(user, "realname");
        var time = ImgoApi.Number(item, "sendTime");
        var initialSource = name.Length > 0 ? name : mine ? session.Account : "";
        var initial = initialSource.Length == 0 ? "?" : System.Globalization.StringInfo.GetNextTextElement(initialSource);
        return new ChatMessage(ImgoApi.Number(item, "msg_id"), mine ? "我" : name,
            time is > 0 and <= 253402300799999 ? DateTimeOffset.FromUnixTimeMilliseconds(time).ToLocalTime().ToString("MM-dd HH:mm") : "",
            ImgoApi.MessageText(item, "content"), mine,
            mine && group == 0 ? ImgoApi.Number(item, "is_read") > 0 ? "已读" : "未读" : "",
            ImgoApi.Field(item, "type"), ImgoApi.Field(item, "fileName"), ImgoApi.Field(item, "download"),
            ImgoApi.Number(item, "pid"), FileId: ImgoApi.Number(item, "file_id"),
            Poster: ImgoApi.Field(ImgoApi.Property(item, "extends"), "poster"),
            SenderId: senderId, Timestamp: time, Avatar: ImgoApi.AvatarUri(session.Server, ImgoApi.Field(user, "avatar")), AvatarInitial: initial);
    }

    private bool IsAtBottom()
    {
        if (visibleMessages.Count == 0) return true;
        return Messages.Template?.FindName("PART_ScrollViewer", Messages) is not ScrollViewer scroll ||
            scroll.ScrollableHeight - scroll.VerticalOffset <= 2;
    }

    private void RenderMessages(List<ChatMessage> messages, bool followLatest, bool preserveOffset = true)
    {
        var scroll = Messages.Template?.FindName("PART_ScrollViewer", Messages) as ScrollViewer;
        var offset = scroll?.VerticalOffset ?? 0;
        var source = messages.ToLookup(message => message.Id);
        var rendered = new List<ChatMessage>(messages.Count);
        ChatMessage? previous = null;
        foreach (var message in messages)
        {
            bool validTime = message.Time.Length > 0;
            bool divider = validTime && (previous is null || previous.Time.Length == 0 ||
                message.Timestamp - previous.Timestamp >= TimeSpan.FromMinutes(5).TotalMilliseconds ||
                !message.Time.AsSpan(0, 5).SequenceEqual(previous.Time.AsSpan(0, 5)));
            bool sameSender = previous is not null && !divider && previous.Type != "event" &&
                previous.IsMine == message.IsMine && previous.SenderId == message.SenderId &&
                (message.SenderId != 0 || previous.Sender == message.Sender);
            bool showAvatar = !sameSender && message.Type != "event" && visibleConversation?.Group != 2;
            rendered.Add(message with
            {
                Quote = message.ParentId <= 0 ? "" : source[message.ParentId].FirstOrDefault() is { } parent
                    ? $"{parent.Sender} · {parent.Body}" : "引用了一条较早的消息",
                TimeDivider = divider ? message.Time : "",
                ShowAvatar = showAvatar,
                ShowSender = showAvatar && !message.IsMine && visibleConversation?.Group == 1 && message.Sender.Length > 0
            });
            previous = message;
        }
        if (visibleMessages.SequenceEqual(rendered)) return;
        visibleMessages = rendered;
        Messages.ItemsSource = rendered;
        UpdateChatEmpty();
        if (followLatest && messages.Count > 0)
        {
            Messages.ScrollIntoView(rendered[^1]);
            Dispatcher.BeginInvoke(() =>
            {
                if (ReferenceEquals(visibleMessages, rendered))
                    (Messages.Template?.FindName("PART_ScrollViewer", Messages) as ScrollViewer)?.ScrollToEnd();
            }, DispatcherPriority.Loaded);
        }
        else if (preserveOffset && scroll is not null)
            Dispatcher.BeginInvoke(() =>
            {
                if (ReferenceEquals(visibleMessages, rendered)) scroll.ScrollToVerticalOffset(offset);
            }, DispatcherPriority.Loaded);
    }

    private void UpdateChatEmpty()
    {
        ChatEmpty.Visibility = visibleMessages.Count == 0 ? Visibility.Visible : Visibility.Collapsed;
        ChatEmptyText.Text = visibleConversation is null ? "从左侧选择会话" :
            messageRevision < 0 ? "正在读取消息…" : "暂无消息";
    }

    private void ComposeChanged(object sender, TextChangedEventArgs e)
    {
        if (SendButton is not null) SendButton.IsEnabled = Compose.IsEnabled && !string.IsNullOrWhiteSpace(Compose.Text);
    }

    private void ChatShortcuts(object sender, KeyEventArgs e)
    {
        if (LoginPanel.Visibility != Visibility.Collapsed || ChatWorkspace.Visibility != Visibility.Visible) return;
        if (e.Key == Key.F11)
        {
            e.Handled = true;
            ToggleFullScreen(sender, e);
            return;
        }
        if (e.Key == Key.Escape && ConversationSidePane.Visibility == Visibility.Visible)
        {
            e.Handled = true;
            CloseConversationPane();
            return;
        }
        if (e.Key != Key.F || Keyboard.Modifiers != ModifierKeys.Control) return;
        e.Handled = true;
        var search = IsDirectoryVisible ? DirectorySearch : ConversationSearch;
        search.Focus();
        search.SelectAll();
    }

    private async void ShowQuickReplies(object sender, RoutedEventArgs e)
    {
        if (loadingQuickReplies || visibleAccount is not { Expired: false } session ||
            visibleConversation is not { Group: not 2 }) return;
        var generation = conversationGeneration;
        loadingQuickReplies = true;
        UpdateButtons();
        try
        {
            var result = await session.PostAsync("/enterprise/im/getquickreplies", new { });
            if (visibleAccount != session || generation != conversationGeneration ||
                ChatWorkspace.Visibility != Visibility.Visible) return;
            var entries = ImgoApi.Property(result, "items");
            var replies = entries.ValueKind == JsonValueKind.Array ?
                entries.EnumerateArray().Where(entry => entry.ValueKind == JsonValueKind.String)
                    .Select(entry => entry.GetString() ?? "").Where(text => text.Length > 0).ToArray() : [];
            var menu = new ContextMenu { Placement = PlacementMode.Top, PlacementTarget = QuickReplyButton,
                MaxHeight = 290, MaxWidth = 390 };
            if (ImgoApi.Property(result, "can_manage").ValueKind == JsonValueKind.True)
                AddMenuAction(menu, "编辑快捷用语…", () => EditQuickRepliesAsync(session, replies));
            if (replies.Length == 0)
                menu.Items.Add(new MenuItem { Header = "暂无快捷用语", IsEnabled = false });
            foreach (var reply in replies)
            {
                var summary = new TextBlock { Text = reply, TextTrimming = TextTrimming.CharacterEllipsis,
                    TextWrapping = TextWrapping.NoWrap, MaxWidth = 325 };
                var item = new MenuItem { Header = summary, ToolTip = reply };
                item.Click += (_, _) =>
                {
                    if (visibleAccount != session || generation != conversationGeneration || !Compose.IsEnabled) return;
                    var caret = Compose.SelectionStart;
                    Compose.SelectedText = reply;
                    Compose.CaretIndex = caret + reply.Length;
                    Compose.Focus();
                };
                menu.Items.Add(item);
            }
            menu.IsOpen = true;
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session && generation == conversationGeneration)
                Status.Text = ErrorText(exception);
        }
        finally
        {
            loadingQuickReplies = false;
            UpdateButtons();
        }
    }

    private async void PickAttachment(object sender, RoutedEventArgs e) => await PickAttachmentAsync(null);

    private async void PickTypedAttachment(object sender, RoutedEventArgs e)
    {
        if (sender is Button { Tag: string type }) await PickAttachmentAsync(type);
    }

    private async Task PickAttachmentAsync(string? type)
    {
        if (!CanSendAttachment || ActiveChatAccount is not { } session || visibleConversation is not { } conversation) return;
        var picker = new OpenFileDialog
        {
            Title = type switch { "image" => "发送图片", "voice" => "发送语音文件", "video" => "发送视频", _ => "选择要发送的文件" },
            Filter = type switch
            {
                "image" => ImageFileFilter,
                "voice" => "语音文件|*.mp3;*.wav;*.amr;*.wmv",
                "video" => "视频文件|*.mp4;*.mov;*.mkv;*.avi;*.3gp;*.webm",
                _ => "所有文件|*.*"
            },
            CheckFileExists = true,
            Multiselect = false
        };
        if (picker.ShowDialog(this) == true)
            await SendAttachmentAsync(session, conversation, picker.FileName, type);
    }

    private async Task SendAttachmentAsync(AccountSession session, Conversation conversation, string path, string? type = null)
    {
        transferringAttachment = true;
        UpdateButtons();
        var workingText = $"正在发送 {Path.GetFileName(path)}…";
        Status.Text = workingText;
        try
        {
            await session.UploadAttachmentAsync(path, conversation, Guid.NewGuid().ToString("N"), type);
            session.Invalidate();
            if (visibleAccount == session && Status.Text == workingText) Status.Text = "附件已发送";
            ScheduleRefreshes();
        }
        catch (Exception exception) when (IsAttachmentFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session) Status.Text = ErrorText(exception);
        }
        finally
        {
            transferringAttachment = false;
            UpdateButtons();
        }
    }

    private async void PreviewAttachment(object sender, RoutedEventArgs e)
    {
        if (sender is not Button { DataContext: ChatMessage { IsImage: true, HasAttachment: true } message } button ||
            visibleAccount is not { Expired: false } session) return;
        var generation = conversationGeneration;
        button.IsEnabled = false;
        try
        {
            await PreviewImageAsync(session, message, this, () => generation == conversationGeneration && session == visibleAccount);
        }
        catch (Exception exception) when (IsAttachmentFailure(exception) || exception is ArgumentException or NotSupportedException or FormatException)
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (session == visibleAccount && generation == conversationGeneration)
                Status.Text = exception is ArgumentException or NotSupportedException or FormatException
                    ? "无法预览此图片，可保存后查看。" : ErrorText(exception);
        }
        finally { button.IsEnabled = true; }
    }

    private async void SaveAttachment(object sender, RoutedEventArgs e)
    {
        if (sender is not Button { DataContext: ChatMessage { HasAttachment: true } message } button ||
            visibleAccount is not { Expired: false } session) return;
        button.IsEnabled = false;
        try
        {
            if (await SaveAttachmentAsync(session, message, this) && visibleAccount == session) Status.Text = "附件已保存";
        }
        catch (Exception exception) when (IsAttachmentFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session) Status.Text = ErrorText(exception);
        }
        finally { button.IsEnabled = true; }
    }

    private static bool IsAttachmentFailure(Exception exception) =>
        IsRequestFailure(exception) || exception is IOException or UnauthorizedAccessException;

    private async void SendMessage(object sender, RoutedEventArgs e) => await SendAsync();

    private async void ComposeKeyDown(object sender, KeyEventArgs e)
    {
        if (e.Key == Key.V && Keyboard.Modifiers == ModifierKeys.Control && await TryPasteImageAsync(e)) return;
        if (e.Key != Key.Enter || Keyboard.Modifiers.HasFlag(ModifierKeys.Shift)) return;
        var control = Keyboard.Modifiers.HasFlag(ModifierKeys.Control);
        if (visibleAccount?.EnterToSend == true && control)
        {
            e.Handled = true;
            var start = Compose.SelectionStart;
            Compose.SelectedText = Environment.NewLine;
            Compose.Select(Math.Min(start + Environment.NewLine.Length, Compose.Text.Length), 0);
            return;
        }
        if (visibleAccount?.EnterToSend != true && !control) return;
        e.Handled = true;
        await SendAsync();
    }

    private async Task SendAsync()
    {
        if (!SendButton.IsEnabled || visibleAccount is not { } session || visibleConversation is not { } conversation) return;
        var content = Compose.Text.Trim();
        if (content.Length == 0) return;
        var quoted = replyTo;
        var mentions = session.Mentions.TryGetValue(conversation.Id, out var names)
            ? names.Where(mention => content.Contains($"@{mention.Value} ", StringComparison.Ordinal) ||
                content.EndsWith($"@{mention.Value}", StringComparison.Ordinal)).Select(mention => mention.Key).ToArray()
            : [];
        sending = true;
        UpdateButtons();
        try
        {
            await session.PostAsync("/enterprise/im/sendmessage", new
            {
                toContactId = conversation.Id,
                is_group = conversation.Group,
                type = "text",
                content,
                pid = quoted?.Id ?? 0,
                at = mentions,
                id = Guid.NewGuid().ToString("N")
            });
            session.Invalidate();
            session.Drafts.Remove(conversation.Id);
            session.Mentions.Remove(conversation.Id);
            if (visibleAccount == session && visibleConversation?.Id == conversation.Id)
            {
                Compose.Clear();
                if (replyTo == quoted) SetQuote(null);
                Status.Text = "";
            }
            ScheduleRefreshes();
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session) Status.Text = ErrorText(exception);
        }
        finally
        {
            sending = false;
            UpdateButtons();
        }
    }

    private static bool IsRequestFailure(Exception exception) =>
        exception is ApiException or HttpRequestException or OperationCanceledException or JsonException or InvalidOperationException;

    private static string ErrorText(Exception exception) => exception switch
    {
        HttpRequestException => "无法连接服务器，请检查网络和 API 地址。",
        OperationCanceledException => "请求超时或已取消，请稍后重试。",
        _ => exception.Message
    };

    protected override void OnClosing(System.ComponentModel.CancelEventArgs e)
    {
        if (activeCall is { } call)
        {
            e.Cancel = true;
            call.CloseAppAfterEnd = true;
            if (!call.Ending) _ = Dispatcher.InvokeAsync(() => { _ = EndCall(call, call.EndCode); });
        }
        base.OnClosing(e);
    }

    protected override void OnClosed(EventArgs e)
    {
        timer.Stop();
        lifetime.Cancel();
        foreach (var session in accounts) session.Stop();
        base.OnClosed(e);
    }
}
