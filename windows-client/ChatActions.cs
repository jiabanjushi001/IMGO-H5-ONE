using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Input;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private ChatMessage? replyTo;
    private bool runningChatAction;

    private AccountSession? ActiveChatAccount =>
        visibleAccount is { Expired: false, SigningOut: false } session ? session : null;

    private void OpenContactsMenu(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is not { } session) return;
        var menu = MakeMenu((FrameworkElement)sender);
        AddMenuAction(menu, "添加好友", () => AddFriendAsync(session));
        AddMenuAction(menu, session.FriendRequests > 0 ? $"好友申请（{session.FriendRequests}）" : "好友申请", () => FriendRequestsAsync(session));
        AddMenuAction(menu, "创建群聊", () => ShowGroupInviteAsync(session, null));
        if (session.HasPermission("manage.groups"))
            AddMenuAction(menu, "按群号加入", () => JoinGroupAsync(session));
        menu.IsOpen = true;
    }

    private async void OpenCurrentConversationMenu(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is not { } session || visibleConversation is not { } conversation) return;
        if (ConversationSidePane.Visibility == Visibility.Visible) { CloseConversationPane(); return; }
        if (conversation.Group == 1) await EditGroupAsync(session, conversation);
        else await ShowContactSideAsync(session, conversation);
    }

    private void OpenConversationRowMenu(object sender, MouseButtonEventArgs e)
    {
        if (sender is not ListBoxItem { DataContext: Conversation conversation } row) return;
        e.Handled = true;
        OpenConversationMenu(row, conversation, PlacementMode.MousePoint);
    }

    private void OpenConversationMenu(FrameworkElement anchor, Conversation conversation, PlacementMode placement = PlacementMode.Bottom)
    {
        if (ActiveChatAccount is not { } session || conversation.Group is not (0 or 1)) return;
        var menu = MakeMenu(anchor, placement);
        async Task ForSelectedConversation(Func<Task> show)
        {
            session.SelectedConversation = conversation.Id;
            ApplyContacts(session);
            ScheduleRefreshes();
            await show();
        }
        AddMenuAction(menu, conversation.Group == 1 ? "群资料与成员" : "查看资料",
            () => ForSelectedConversation(() => conversation.Group == 1 ? ShowGroupMemberSideAsync(session, conversation) :
                ShowContactSideAsync(session, conversation)));
        if (conversation.Group == 0)
            AddMenuAction(menu, "设置备注", () => ForSelectedConversation(() => EditRemarkAsync(session, conversation)));
        if (conversation is { Group: 1, Role: 1 } or { Group: 1, Role: 2, ManagerCanInvite: true })
            AddMenuAction(menu, "邀请成员", () => ForSelectedConversation(() => ShowGroupInviteAsync(session, conversation)));
        if (conversation.Group == 1)
            AddMenuAction(menu, "群设置", () => ForSelectedConversation(() => EditGroupAsync(session, conversation)));
        AddMenuAction(menu, conversation.Pinned ? "取消置顶" : "置顶会话", () =>
            ChangeConversationAsync(session, "setchattop", new { id = conversation.Id, is_group = conversation.Group, is_top = conversation.Pinned ? 0 : 1 }));
        AddMenuAction(menu, conversation.Muted ? "恢复消息提醒" : "消息免打扰", () =>
            ChangeConversationAsync(session, "isnotice", new { id = conversation.Id, is_group = conversation.Group, is_notice = conversation.Muted ? 1 : 0 }));
        menu.Items.Add(new Separator());
        AddMenuAction(menu, "清空我的会话记录…", () => ClearConversationAsync(session, conversation));
        if (conversation.Group == 0)
            AddMenuAction(menu, "删除好友…", () => RemoveContactAsync(session, conversation));
        else if (conversation.Role > 1)
            AddMenuAction(menu, "退出群聊…", () => RemoveContactAsync(session, conversation));
        menu.IsOpen = true;
    }

    private async Task ClearConversationAsync(AccountSession session, Conversation conversation)
    {
        if (MessageBox.Show(this, $"清空自己在 {conversation.Name} 的聊天记录？", "清空聊天记录",
            MessageBoxButton.YesNo, MessageBoxImage.Question) != MessageBoxResult.Yes) return;
        await session.PostAsync("/enterprise/im/delchat", new { id = conversation.Id, is_group = conversation.Group });
        session.InvalidateHistory();
        RefreshAfterAction(session, "聊天记录已清空");
    }

    private async Task RemoveContactAsync(AccountSession session, Conversation conversation)
    {
        bool group = conversation.Group == 1;
        var action = group ? "退出群聊" : "删除好友";
        if (MessageBox.Show(this, $"确定{action} {conversation.Name}？", action,
            MessageBoxButton.YesNo, MessageBoxImage.Question) != MessageBoxResult.Yes) return;
        if (group)
            await session.PostAsync("/enterprise/group/removeuser", new { group_id = conversation.Id, user_id = session.UserId });
        else
            await session.PostAsync("/enterprise/friend/del", new { user_id = conversation.Id });
        if (session.SelectedConversation == conversation.Id)
        {
            session.SelectedConversation = "";
            if (ActiveChatAccount == session) ShowConversation(null);
        }
        RefreshAfterAction(session, group ? "已退出群聊" : "已删除好友");
    }

    private static ContextMenu MakeMenu(FrameworkElement anchor, PlacementMode placement = PlacementMode.Bottom) =>
        new() { PlacementTarget = anchor, Placement = placement };

    private void AddMenuAction(ContextMenu menu, string label, Func<Task> action)
    {
        var item = new MenuItem { Header = label };
        item.Click += async (_, _) => await RunChatActionAsync(action);
        menu.Items.Add(item);
    }

    private async Task RunChatActionAsync(Func<Task> action)
    {
        if (runningChatAction || ActiveChatAccount is not { } session) return;
        runningChatAction = true;
        try { await action(); }
        catch (Exception exception) when (IsAttachmentFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session) Status.Text = ErrorText(exception);
        }
        finally { runningChatAction = false; }
    }

    private async Task ChangeConversationAsync(AccountSession session, string action, object body)
    {
        await session.PostAsync($"/enterprise/im/{action}", body);
        RefreshAfterAction(session, "会话设置已保存");
    }

    private void RefreshAfterAction(AccountSession session, string message)
    {
        session.Invalidate();
        if (visibleAccount == session) Status.Text = message;
        ScheduleRefreshes();
    }

    private async Task EditRemarkAsync(AccountSession session, Conversation conversation, bool push = false)
    {
        if (!long.TryParse(conversation.Id, out var userId)) return;
        var page = OpenSidePage("设置好友备注", push);
        var status = new TextBlock { Text = "正在读取备注…", TextWrapping = TextWrapping.Wrap };
        page.Body.Children.Add(ContactHeading(conversation));
        page.Body.Children.Add(status);
        JsonElement profile;
        try { profile = await session.PostAsync("/enterprise/im/getuserinfo", new { user_id = userId }); }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            status.Text = ErrorText(error);
            return;
        }
        if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
        var friend = ImgoApi.Property(profile, "friendInfo");
        if (ImgoApi.Number(friend, "status") != 1)
        {
            status.Text = "添加为好友后可以设置备注";
            return;
        }

        status.Text = "备注（留空可清除）";
        var remark = new TextBox { Text = ImgoApi.Field(friend, "nickname"), MaxLength = 100 };
        page.Body.Children.Add(remark);
        var save = AddDialogButton(page.Footer, "保存", () => { });
        save.Style = (Style)FindResource("ChatPrimaryAction");
        save.Click += async (_, _) =>
        {
            if (ActiveChatAccount != session) return;
            save.IsEnabled = remark.IsEnabled = false;
            try
            {
                await session.PostAsync("/enterprise/friend/setnickname", new { user_id = userId, nickname = remark.Text.Trim() });
                RefreshAfterAction(session, "备注已保存");
                if (IsSidePageOpen(page))
                {
                    if (push) await ShowContactSideAsync(session, conversation);
                    else CloseConversationPane();
                }
            }
            catch (Exception error) when (IsRequestFailure(error))
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                status.Text = ErrorText(error);
            }
            finally { save.IsEnabled = remark.IsEnabled = true; }
        };
        remark.Focus();
    }

    private async Task LoadFriendRequestCountAsync(AccountSession session)
    {
        try
        {
            var data = await session.PostAsync("/enterprise/friend/getapplymsg", new { });
            if (data.ValueKind == JsonValueKind.Number && data.TryGetInt64(out var count)) session.FriendRequests = count;
            if (visibleAccount == session) UpdateButtons();
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
        }
    }

    private async void ChooseMention(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is not { } session || visibleConversation is not { Group: 1 } conversation) return;
        await RunChatActionAsync(async () =>
        {
            var generation = conversationGeneration;
            var list = await session.PostAsync("/enterprise/group/groupuserlist", new { group_id = conversation.Id });
            if (visibleAccount != session || generation != conversationGeneration) return;
            var people = list.EnumerateArray().Select(row => ImgoApi.Property(row, "userInfo"))
                .Where(user => ImgoApi.Number(user, "user_id") != session.UserId).ToArray();
            var choices = ChatPeople(JsonSerializer.SerializeToElement(people));
            var selected = ChooseChatItems("提及群成员", choices, 30);
            if (selected is null || !Compose.IsEnabled) return;
            if (!session.Mentions.TryGetValue(conversation.Id, out var names))
            {
                names = new Dictionary<string, string>(StringComparer.Ordinal);
                session.Mentions[conversation.Id] = names;
            }
            var mentions = choices.Where(choice => selected.Contains(choice.Id)).Select(choice =>
            {
                names[choice.Id] = choice.Label;
                return $"@{choice.Label} ";
            });
            var content = string.Concat(mentions);
            var start = Compose.SelectionStart;
            Compose.SelectedText = content;
            Compose.CaretIndex = start + content.Length;
            Compose.Focus();
        });
    }

    private async Task ShowGroupMemberSideAsync(AccountSession session, Conversation conversation)
    {
        var page = OpenSidePage("群成员");
        var info = await LoadGroupSideInfoAsync(page, session, conversation);
        if (IsSidePageOpen(page) && info.ValueKind == JsonValueKind.Object)
            await ShowGroupMembersAsync(session, conversation, info, existingPage: page);
    }

    private async Task ShowGroupMembersAsync(AccountSession session, Conversation conversation, JsonElement info,
        bool push = false, bool transferOnly = false, SidePage? existingPage = null)
    {
        var side = existingPage ?? OpenSidePage(transferOnly ? "选择接任群主" : "群成员", push);
        var panel = side.Body;
        var buttons = side.Footer;
        AddDetail(panel, "群名", ImgoApi.Field(info, "name"));
        if (!transferOnly)
        {
            AddDetail(panel, "群号", ImgoApi.Field(info, "group_id"));
            AddDetail(panel, "群主", ImgoApi.Field(info, "ownerName"));
        }
        var search = new TextBox { Margin = new Thickness(0, 8, 0, 6), ToolTip = "输入成员关键词后按回车" };
        var list = new ListBox { Height = 215, DisplayMemberPath = nameof(ChatChoice.Label) };
        var status = new TextBlock { Foreground = Foreground, Margin = new Thickness(0, 6, 0, 0) };
        panel.Children.Add(new TextBlock { Text = "搜索群成员（按回车）" });
        panel.Children.Add(search);
        panel.Children.Add(list);
        panel.Children.Add(status);
        int pageNumber = 1;
        const int limit = 40;
        string keyword = "";
        bool loading = false;
        var previous = AddDialogButton(buttons, "上一页", () => { });
        var next = AddDialogButton(buttons, "下一页", () => { });
        previous.Click += async (_, _) => await LoadAsync(pageNumber - 1);
        next.Click += async (_, _) => await LoadAsync(pageNumber + 1);
        long myRole = ImgoApi.Number(info, "isJoin");
        bool changingMember = false;
        bool canViewMember = myRole is 1 or 2 || session.IsSuperAdmin || session.CanAddGroupFriends ||
            ImgoApi.Number(ImgoApi.Property(info, "setting"), "profile") == 1;
        if (myRole is 1 or 2 || canViewMember)
        {
            var manage = AddDialogButton(buttons, transferOnly ? "转让给此成员" : "成员操作", () => { });
            bool CanOperate(ChatChoice member) => member.Id.Length > 0 && member.Id != session.UserId.ToString() &&
                (transferOnly ? myRole == 1 && ImgoApi.Number(member.Data, "role") > 1 : canViewMember);
            manage.IsEnabled = false;
            list.SelectionChanged += (_, _) => manage.IsEnabled = list.SelectedItem is ChatChoice item && CanOperate(item);
            manage.Click += async (_, _) =>
            {
                if (loading || changingMember || list.SelectedItem is not ChatChoice member) return;
                long role = ImgoApi.Number(member.Data, "role");
                if (!CanOperate(member)) return;
                async Task ApplyMemberActionAsync(string label, string path, object body, bool confirm)
                {
                    if (changingMember || loading || !IsSidePageOpen(side)) return;
                    if (confirm && MessageBox.Show(this, $"{label} {member.Label}？", "成员操作",
                        MessageBoxButton.YesNo, MessageBoxImage.Question, MessageBoxResult.No) != MessageBoxResult.Yes) return;
                    changingMember = true;
                    manage.IsEnabled = false;
                    try
                    {
                        await session.PostAsync($"/enterprise/group/{path}", body);
                        session.Invalidate();
                        if (!IsSidePageOpen(side)) return;
                        if (path == "changeowner")
                        {
                            CloseConversationPane();
                            RefreshAfterAction(session, "群主已转让");
                            return;
                        }
                        await LoadAsync(pageNumber);
                    }
                    catch (Exception error) when (IsRequestFailure(error))
                    {
                        if (error is ApiException { SessionExpired: true }) session.Expire();
                        status.Text = ErrorText(error);
                    }
                    finally
                    {
                        changingMember = false;
                        manage.IsEnabled = list.SelectedItem is ChatChoice selected && CanOperate(selected);
                    }
                }
                if (transferOnly)
                {
                    await ApplyMemberActionAsync("转让群主给", "changeowner",
                        new { group_id = conversation.Id, user_id = member.Id }, confirm: true);
                    return;
                }
                var menu = MakeMenu(manage);
                void Add(ItemCollection destination, string label, string path, object body, bool confirm = false)
                {
                    var entry = new MenuItem { Header = label };
                    entry.Click += async (_, _) => await ApplyMemberActionAsync(label, path, body, confirm);
                    destination.Add(entry);
                }
                var groupId = conversation.Id;
                var userId = member.Id;
                if (canViewMember)
                {
                    var details = new MenuItem { Header = "查看资料与添加好友…" };
                    details.Click += async (_, _) => await ViewGroupMemberAsync(session, conversation, member);
                    menu.Items.Add(details);
                }
                if (myRole is 1 or 2 && role > myRole)
                {
                    if (myRole == 1)
                    {
                        Add(menu.Items, role == 2 ? "取消管理员" : "设为管理员", "setmanager", new { group_id = groupId, user_id = userId, role = role == 2 ? 3 : 2 });
                        Add(menu.Items, "转让群主", "changeowner", new { group_id = groupId, user_id = userId }, confirm: true);
                    }
                    var mute = new MenuItem { Header = "禁言…" };
                    foreach (var (label, timer) in new[] { ("10 分钟", 1), ("1 小时", 2), ("3 小时", 3), ("1 天", 4) })
                        Add(mute.Items, label, "setnospeak", new { group_id = groupId, user_id = userId, noSpeakTimer = timer });
                    var custom = new MenuItem { Header = "自定义天数…" };
                    custom.Click += async (_, _) =>
                    {
                        if (changingMember || loading) return;
                        var days = new TextBox { Text = "1" };
                        if (!EditChatFields("设置禁言天数", "确定", this, ("天数（1–365）", days))) return;
                        if (!int.TryParse(days.Text.Trim(), out int dayCount) || dayCount is < 1 or > 365)
                        {
                            status.Text = "请输入 1–365 天";
                            return;
                        }
                        changingMember = true;
                        try
                        {
                            await session.PostAsync("/enterprise/group/setnospeak", new { group_id = groupId, user_id = userId, noSpeakDay = dayCount });
                            await LoadAsync(pageNumber);
                        }
                        catch (Exception error) when (IsRequestFailure(error))
                        {
                            if (error is ApiException { SessionExpired: true }) session.Expire();
                            status.Text = ErrorText(error);
                        }
                        finally { changingMember = false; }
                    };
                    mute.Items.Add(custom);
                    menu.Items.Add(mute);
                    Add(menu.Items, "解除禁言", "setnospeak", new { group_id = groupId, user_id = userId, noSpeakTimer = 0 });
                    Add(menu.Items, "移出群聊", "removeuser", new { group_id = groupId, user_id = userId }, confirm: true);
                }
                menu.IsOpen = true;
            };
        }
        search.KeyDown += async (_, args) =>
        {
            if (args.Key != Key.Enter || loading) return;
            args.Handled = true;
            keyword = search.Text.Trim();
            await LoadAsync(1);
        };
        search.Focus();
        await LoadAsync(1);

        async Task LoadAsync(int page)
        {
            if (loading || page < 1) return;
            loading = true;
            previous.IsEnabled = next.IsEnabled = false;
            status.Text = "读取成员中…";
            try
            {
                var result = await session.PostPageAsync("/enterprise/group/groupuserlist",
                    new { group_id = conversation.Id, keywords = keyword, page, limit });
                if (!IsSidePageOpen(side)) return;
                pageNumber = page;
                list.ItemsSource = result.Data.EnumerateArray()
                    .Where(row => !transferOnly || ImgoApi.Number(row, "user_id") != session.UserId)
                    .Select(row =>
                {
                    var user = ImgoApi.Property(row, "userInfo");
                    var role = ImgoApi.Number(row, "role") switch { 1 => " · 群主", 2 => " · 管理员", _ => "" };
                    long until = ImgoApi.Number(row, "no_speak_time");
                    var muted = until > DateTimeOffset.UtcNow.ToUnixTimeSeconds() && until <= 253402300799
                        ? $" · 禁言至 {DateTimeOffset.FromUnixTimeSeconds(until).ToLocalTime():MM-dd HH:mm}" : "";
                    return new ChatChoice(PersonName(user) + role + muted, row, ImgoApi.Field(row, "user_id"));
                }).ToList();
                status.Text = $"第 {page} 页 · 共 {result.Count} 人";
                previous.IsEnabled = page > 1;
                next.IsEnabled = page * (long)limit < result.Count;
            }
            catch (Exception error) when (IsRequestFailure(error))
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                status.Text = ErrorText(error);
            }
            finally { loading = false; }
        }
    }

    private static string NetworkDetail(JsonElement data, string time, string ip, string location)
    {
        long seconds = ImgoApi.Number(data, time);
        if (seconds is <= 0 or > 253402300799) return "";
        return string.Join(" · ", new[] { DateTimeOffset.FromUnixTimeSeconds(seconds).ToLocalTime().ToString("yyyy-MM-dd HH:mm:ss"),
            ImgoApi.Field(data, ip), ImgoApi.Field(data, location) }.Where(part => part.Length > 0));
    }

    private static void AddDetail(StackPanel panel, string label, string value)
    {
        if (value.Length == 0) return;
        panel.Children.Add(new TextBlock { Text = $"{label}　{value}", TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0,0,0,8) });
    }

    private static string PersonName(JsonElement person)
    {
        var nickname = ImgoApi.Field(person, "displayName");
        if (nickname.Length == 0) nickname = ImgoApi.Field(person, "realname");
        var account = ImgoApi.Field(person, "account");
        if (nickname.Length == 0) return account.Length > 0 ? account : $"用户 #{ImgoApi.Number(person, "user_id")}";
        return account.Length == 0 || account == nickname ? nickname : $"{nickname}（{account}）";
    }

    private static IReadOnlyList<ChatChoice> ChatPeople(JsonElement data) => data.EnumerateArray()
        .Select(person => new ChatChoice(PersonName(person), person, ImgoApi.Field(person, "user_id")))
        .Where(person => person.Id.Length > 0).ToList();

    private string[]? ChooseChatItems(string title, IReadOnlyList<ChatChoice> choices, int maximum, bool allowEmpty = false)
    {
        var window = ChatDialog(title, out var panel, out var buttons, 485);
        var search = new TextBox { ToolTip = "筛选名称或账号" };
        var summary = new TextBlock { Margin = new Thickness(0, 6, 0, 0) };
        var selected = new HashSet<string>(StringComparer.Ordinal);
        var choicePanel = new StackPanel();
        panel.Children.Add(new TextBlock { Text = "筛选名称或账号", Margin = new Thickness(0,0,0,4) });
        panel.Children.Add(search);
        panel.Children.Add(new ScrollViewer { Height = 240, Margin = new Thickness(0, 8, 0, 0),
            VerticalScrollBarVisibility = ScrollBarVisibility.Auto, Content = choicePanel });
        panel.Children.Add(summary);

        AddDialogButton(buttons, "取消", window.Close).IsCancel = true;
        var accept = AddDialogButton(buttons, "确定", () => window.DialogResult = true, primary: true);
        void Update()
        {
            summary.Text = $"已选 {selected.Count} / {maximum}";
            accept.IsEnabled = (allowEmpty || selected.Count > 0) && selected.Count <= maximum;
        }
        void Filter()
        {
            choicePanel.Children.Clear();
            var text = search.Text.Trim();
            foreach (var item in choices.Where(item => item.Label.Contains(text, StringComparison.OrdinalIgnoreCase) ||
                item.Id.Equals(text, StringComparison.Ordinal)).Take(100))
            {
                var check = new CheckBox { Content = item.Label, IsChecked = selected.Contains(item.Id), Margin = new Thickness(0, 5, 0, 5) };
                check.Checked += (_, _) => { selected.Add(item.Id); Update(); };
                check.Unchecked += (_, _) => { selected.Remove(item.Id); Update(); };
                choicePanel.Children.Add(check);
            }
            if (choicePanel.Children.Count == 0) choicePanel.Children.Add(new TextBlock { Text = "没有匹配的成员或会话" });
            if (choicePanel.Children.Count == 100) choicePanel.Children.Add(new TextBlock { Text = "使用搜索缩小范围" });
        }
        window.Loaded += (_, _) => search.Focus();
        search.TextChanged += (_, _) => Filter();
        Filter();
        Update();
        return window.ShowDialog() == true ? selected.ToArray() : null;
    }

    private void ConfigureMessageMenu(object sender, RoutedEventArgs e)
    {
        if (sender is not ContextMenu { PlacementTarget: FrameworkElement { DataContext: ChatMessage message } } menu) return;
        foreach (var item in menu.Items.OfType<MenuItem>())
            item.IsEnabled = (item.Tag as string) switch
            {
                "copy" => message.Body.Length > 0,
                "emoji" => message.IsImage && message.FileId > 0,
                "reply" => Compose.IsEnabled && message.Id > 0 && message.Type != "event",
                "forward" => message.Id > 0 && visibleConversation?.Group != 2 && message.Type is not ("event" or "webrtc"),
                "undo" => message.Id > 0 && visibleConversation?.Group != 2 && message.Type != "event" &&
                    (message.IsMine || visibleConversation is { Group: 1, Role: 1 or 2 }),
                _ => message.Id > 0 && visibleConversation?.Group != 2
            };
    }

    private async void MessageMenuAction(object sender, RoutedEventArgs e)
    {
        if (sender is not MenuItem { Tag: string action } item ||
            ItemsControl.ItemsControlFromItemContainer(item) is not ContextMenu { PlacementTarget: FrameworkElement { DataContext: ChatMessage message } })
            return;

        if (action == "copy")
        {
            try { Clipboard.SetText(message.Body); }
            catch (System.Runtime.InteropServices.ExternalException) { Status.Text = "剪贴板暂时无法使用"; }
            return;
        }
        if (ActiveChatAccount is not { } session || visibleConversation is not { Group: not 2 } conversation) return;
        if (action == "reply")
        {
            SetQuote(message);
            Compose.Focus();
            return;
        }
        await RunChatActionAsync(async () =>
        {
            if (action == "emoji")
            {
                await session.PostAsync("/enterprise/emoji/add", new { file_id = message.FileId });
                Status.Text = "已收藏为表情";
                return;
            }
            if (action == "forward")
            {
                var targets = session.Conversations.Where(target => target.Group is 0 or 1 or 3)
                    .Select(target => new ChatChoice($"{target.Name} · {target.Kind}", default, target.Id)).ToList();
                var recipients = ChooseChatItems("转发给", targets, 5);
                if (recipients is null) return;
                await session.PostAsync("/enterprise/im/forwardmessage", new { msg_id = message.Id, user_ids = recipients });
                RefreshAfterAction(session, "消息已转发");
                return;
            }
            var undo = action == "undo";
            var label = undo ? "撤回消息" : "从我的记录移除消息";
            if (MessageBox.Show(this, $"确定{label}？", label, MessageBoxButton.YesNo, MessageBoxImage.Question) != MessageBoxResult.Yes) return;
            await session.PostAsync($"/enterprise/im/{(undo ? "undomessage" : "removemessage")}", new { msg_id = message.Id });
            session.InvalidateHistory();
            RefreshAfterAction(session, undo ? "消息已撤回" : "消息已移除");
        });
    }

    private void ClearQuote(object sender, RoutedEventArgs e) => SetQuote(null);

    private void SetQuote(ChatMessage? message)
    {
        replyTo = message;
        QuoteBar.Visibility = message is null ? Visibility.Collapsed : Visibility.Visible;
        QuoteText.Text = message is null ? "" : $"回复 {message.Sender} · {message.Body}";
    }

    private async Task EditQuickRepliesAsync(AccountSession session, string[] replies)
    {
        var input = new TextBox { Text = string.Join(Environment.NewLine, replies), AcceptsReturn = true,
            Height = 180, TextWrapping = TextWrapping.Wrap, VerticalScrollBarVisibility = ScrollBarVisibility.Auto };
        if (!EditChatFields("编辑快捷用语", "保存", ("每行一条快捷用语", input))) return;
        var items = input.Text.Split(['\r','\n'], StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries);
        await session.PostAsync("/enterprise/im/savequickreplies", new { items });
        RefreshAfterAction(session, "快捷用语已保存");
    }

    private bool EditChatFields(string title, string action, params (string Label, FrameworkElement Input)[] fields) =>
        EditChatFields(title, action, this, fields);

    private bool EditChatFields(string title, string action, Window owner, params (string Label, FrameworkElement Input)[] fields)
    {
        var window = ChatDialog(title, out var panel, out var buttons, owner: owner);
        foreach (var field in fields)
        {
            panel.Children.Add(new TextBlock { Text = field.Label, Margin = new Thickness(0, 0, 0, 5) });
            panel.Children.Add(field.Input);
        }
        AddDialogButton(buttons, "取消", window.Close).IsCancel = true;
        AddDialogButton(buttons, action, () => window.DialogResult = true, primary: true).IsDefault = true;
        window.Loaded += (_, _) => fields[0].Input.Focus();
        return window.ShowDialog() == true;
    }

    private Window ChatDialog(string title, out StackPanel panel, out StackPanel buttons, double width = 420, Window? owner = null)
    {
        panel = new StackPanel();
        buttons = new StackPanel { Orientation = Orientation.Horizontal, HorizontalAlignment = HorizontalAlignment.Right, Margin = new Thickness(0, 16, 0, 0) };
        var content = new DockPanel { Margin = new Thickness(20) };
        DockPanel.SetDock(buttons, Dock.Bottom);
        content.Children.Add(buttons);
        content.Children.Add(panel);
        return new Window { Title = title, Owner = owner ?? this, Width = width, SizeToContent = SizeToContent.Height,
            ResizeMode = ResizeMode.NoResize, WindowStartupLocation = WindowStartupLocation.CenterOwner, ShowInTaskbar = false,
            Background = Background, Foreground = Foreground, FontFamily = FontFamily, FontSize = FontSize, Content = content };
    }

    private Button AddDialogButton(Panel buttons, string label, Action action, bool primary = false)
    {
        var button = new Button { Content = label, MinWidth = 64 };
        if (primary) button.Style = (Style)FindResource("PrimaryButton");
        button.Click += (_, _) => action();
        buttons.Children.Add(button);
        return button;
    }

    private sealed record ChatChoice(string Label, JsonElement Data, string Id);
}
