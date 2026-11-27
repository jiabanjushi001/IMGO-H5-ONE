using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private bool updatingDirectory;
    private int directoryRequest;
    private readonly Stack<SidePage> sideHistory = [];
    private bool IsDirectoryVisible => DirectoryListPane?.Visibility == Visibility.Visible;

    private sealed record SidePage(string Title, DockPanel Root, StackPanel Body, WrapPanel Footer);

    private void OpenContactDirectory(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is null) return;
        SetAdminVisible(false);
        SetDirectoryVisible(true);
        DirectorySearch.Focus();
    }

    private void SetDirectoryVisible(bool visible)
    {
        if (IsDirectoryVisible != visible) CloseConversationPane();
        DirectoryListPane.Visibility = DirectoryProfilePane.Visibility = visible ? Visibility.Visible : Visibility.Collapsed;
        ConversationListPane.Visibility = ConversationPane.Visibility = visible ? Visibility.Collapsed : Visibility.Visible;
        var active = (Brush)FindResource("ChatRailActive");
        DirectoryRailButton.Background = visible ? active : Brushes.Transparent;
        ChatRailButton.Background = visible ? Brushes.Transparent : active;
        if (visible) RefreshDirectory();
        else ScheduleRefreshes();
        AdjustChatWidth();
    }

    private void DirectoryFilterChanged(object sender, RoutedEventArgs e)
    {
        if (IsDirectoryVisible && DirectoryEntries is not null) RefreshDirectory();
    }

    private void RefreshDirectory()
    {
        var key = DirectorySearch.Text.Trim();
        var session = ActiveChatAccount;
        List<Conversation> entries = session?.Conversations.Where(item => item.Group is 0 or 1)
            .Where(item => DirectoryFriends.IsChecked == true ? item.Group == 0 : DirectoryGroups.IsChecked != true || item.Group == 1)
            .Where(item => item.Name.Contains(key, StringComparison.OrdinalIgnoreCase) ||
                item.Account.Contains(key, StringComparison.OrdinalIgnoreCase) ||
                item.Group == 1 && (item.Id.Equals($"group-{key}", StringComparison.OrdinalIgnoreCase) ||
                    item.Id.Equals(key, StringComparison.OrdinalIgnoreCase)))
            .OrderBy(item => item.Group).ThenBy(item => item.Name, StringComparer.CurrentCultureIgnoreCase).ToList() ?? [];
        DirectoryEmpty.Visibility = entries.Count == 0 ? Visibility.Visible : Visibility.Collapsed;
        DirectoryEmpty.Text = key.Length == 0 ? "暂无联系人" : "没有匹配的联系人或群聊";

        var previous = DirectoryEntries.SelectedItem as Conversation;
        if (!DirectoryEntries.Items.Cast<Conversation>().SequenceEqual(entries))
        {
            updatingDirectory = true;
            try
            {
                DirectoryEntries.ItemsSource = entries;
                DirectoryEntries.SelectedValue = previous?.Id;
            }
            finally { updatingDirectory = false; }
        }
        if (DirectoryEntries.SelectedItem is null || DirectoryProfile.Children.Count == 0)
            RenderDirectoryProfile();
    }

    private void DirectorySelectionChanged(object sender, SelectionChangedEventArgs e)
    {
        if (updatingDirectory || !IsDirectoryVisible) return;
        CloseConversationPane();
        RenderDirectoryProfile();
    }

    private async void RenderDirectoryProfile()
    {
        int request = ++directoryRequest;
        DirectoryProfile.Children.Clear();
        if (DirectoryEntries.SelectedItem is not Conversation selected || ActiveChatAccount is not { } session)
        {
            DirectoryProfile.Children.Add(new TextBlock { Text = "选择联系人或群聊", FontSize = 16,
                Foreground = (Brush)FindResource("ChatMuted"), HorizontalAlignment = HorizontalAlignment.Center,
                Margin = new Thickness(0, 70, 0, 0) });
            return;
        }

        DirectoryProfile.Children.Add(ContactHeading(selected));
        var details = new StackPanel { Margin = new Thickness(0, 10, 0, 10) };
        DirectoryProfile.Children.Add(details);
        if (selected.Group == 1) AddDetail(details, "群号", selected.Id.Replace("group-", "", StringComparison.Ordinal));
        else AddDetail(details, "账号", selected.Account);

        var actions = new WrapPanel { Margin = new Thickness(0, 8, 0, 0) };
        DirectoryProfile.Children.Add(actions);
        var start = AddDialogButton(actions, selected.Group == 1 ? "进入群聊" : "发消息",
            () => ActivateDirectoryConversation(selected));
        start.Style = (Style)FindResource("ChatPrimaryAction");

        try
        {
            var profile = selected.Group == 1
                ? await session.PostAsync("/enterprise/group/groupinfo", new { group_id = selected.Id })
                : await session.PostAsync("/enterprise/im/getuserinfo", new { user_id = selected.Id });
            if (request != directoryRequest || ActiveChatAccount != session || !IsDirectoryVisible) return;
            if (selected.Group == 1)
            {
                AddDetail(details, "群主", ImgoApi.Field(profile, "ownerName"));
                AddDetail(details, "成员数", ImgoApi.Field(profile, "groupUserCount"));
            }
            else
            {
                if (selected.Account.Length == 0) AddDetail(details, "账号", ImgoApi.Field(profile, "account"));
                var name = ImgoApi.Field(profile, "realname");
                if (name != selected.Name) AddDetail(details, "昵称", name);
                AddDetail(details, "签名", ImgoApi.Field(profile, "motto"));
            }
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            if (request == directoryRequest && IsDirectoryVisible && visibleAccount == session)
                AddDetail(details, "资料读取失败", ErrorText(error));
        }
    }

    private void DirectoryEntryDoubleClick(object sender, MouseButtonEventArgs e)
    {
        if (e.OriginalSource is DependencyObject element &&
            ItemsControl.ContainerFromElement(DirectoryEntries, element) is ListBoxItem { DataContext: Conversation selected })
            ActivateDirectoryConversation(selected);
    }

    private void DirectoryEntryKeyDown(object sender, KeyEventArgs e)
    {
        if (e.Key != Key.Enter || DirectoryEntries.SelectedItem is not Conversation selected) return;
        e.Handled = true;
        ActivateDirectoryConversation(selected);
    }

    private void ActivateDirectoryConversation(Conversation selected)
    {
        if (ActiveChatAccount is not { } session) return;
        SetDirectoryVisible(false);
        OnlyAll.IsChecked = true;
        ConversationSearch.Clear();
        session.SelectedConversation = selected.Id;
        ApplyContacts(session);
        ScheduleRefreshes();
        Compose.Focus();
    }

    private DockPanel ContactHeading(Conversation selected)
    {
        var heading = new DockPanel { Margin = new Thickness(0, 0, 0, 12) };
        var avatar = new Border { Width = 48, Height = 48, CornerRadius = new CornerRadius(8),
            Margin = new Thickness(0, 0, 12, 0), Background = (Brush)FindResource("ChatTint"), ClipToBounds = true };
        var layers = new Grid();
        layers.Children.Add(new TextBlock { Text = selected.Initial, FontSize = 19,
            VerticalAlignment = VerticalAlignment.Center, HorizontalAlignment = HorizontalAlignment.Center });
        var photo = new Image { DataContext = selected, Stretch = Stretch.UniformToFill };
        photo.Loaded += LoadThumbnail;
        layers.Children.Add(photo);
        avatar.Child = layers;
        DockPanel.SetDock(avatar, Dock.Left);
        heading.Children.Add(avatar);
        heading.Children.Add(new TextBlock { Text = selected.Name, FontSize = 19, FontWeight = FontWeights.SemiBold,
            TextWrapping = TextWrapping.Wrap, VerticalAlignment = VerticalAlignment.Center });
        return heading;
    }

    private SidePage OpenSidePage(string title, bool push = false)
    {
        if (push && ConversationSideContent.Tag is SidePage current) sideHistory.Push(current);
        else sideHistory.Clear();
        var root = new DockPanel { Margin = new Thickness(12, 10, 12, 10) };
        var footer = new WrapPanel { Margin = new Thickness(0, 8, 0, 0) };
        DockPanel.SetDock(footer, Dock.Bottom);
        root.Children.Add(footer);
        var body = new StackPanel();
        root.Children.Add(new ScrollViewer { Content = body, VerticalScrollBarVisibility = ScrollBarVisibility.Auto });
        var page = new SidePage(title, root, body, footer);
        ShowSidePage(page);
        return page;
    }

    private void ShowSidePage(SidePage page)
    {
        ConversationSideContent.Tag = page;
        ConversationSideContent.Content = page.Root;
        ConversationSideTitle.Text = page.Title;
        ConversationSideBack.Visibility = sideHistory.Count > 0 ? Visibility.Visible : Visibility.Collapsed;
        ConversationSidePane.Visibility = Visibility.Visible;
        if (IsDirectoryVisible) DirectoryProfilePane.Visibility = Visibility.Collapsed;
        AdjustChatWidth();
        UpdateButtons();
    }

    private bool IsSidePageOpen(SidePage page) => ReferenceEquals(ConversationSideContent.Tag, page);

    private void CloseConversationPane(object sender, RoutedEventArgs e) => CloseConversationPane();

    private void BackConversationPane(object sender, RoutedEventArgs e) => BackConversationPane();

    private void BackConversationPane()
    {
        if (sideHistory.TryPop(out var page)) ShowSidePage(page);
        else CloseConversationPane();
    }

    private void CloseConversationPane()
    {
        if (ConversationSidePane is null) return;
        sideHistory.Clear();
        ConversationSideContent.Tag = null;
        ConversationSideContent.Content = null;
        ConversationSidePane.Visibility = Visibility.Collapsed;
        if (IsDirectoryVisible) DirectoryProfilePane.Visibility = Visibility.Visible;
        AdjustChatWidth();
        UpdateButtons();
    }

    private void AdjustChatWidth()
    {
        bool narrow = ActualWidth < 1050;
        bool compact = narrow && ConversationSidePane.Visibility == Visibility.Visible;
        ChatListColumn.Width = new GridLength(narrow ? 188 : 240);
        bool directoryPage = IsDirectoryVisible && ConversationSidePane.Visibility == Visibility.Visible;
        Grid.SetColumn(ConversationSidePane, directoryPage ? 2 : 3);
        Grid.SetColumnSpan(ConversationSidePane, directoryPage ? 2 : 1);
        ConversationSidePane.Width = directoryPage ? double.NaN : narrow ? 280 : 320;
        if (ConversationSideContent.Tag is SidePage page)
            page.Root.MaxWidth = directoryPage ? 540 : double.PositiveInfinity;
        RefreshOlderButtonLabel(compact);
        ComposeHint.Visibility = compact ? Visibility.Collapsed : Visibility.Visible;
    }

    private void RefreshOlderButtonLabel(bool compact)
    {
        OlderButton.Content = compact ? (loadingOlder ? "…" : "↑") : (loadingOlder ? "读取中…" : "加载更早消息");
    }

    private async Task ShowContactSideAsync(AccountSession session, Conversation conversation)
    {
        var page = OpenSidePage("联系人资料");
        page.Body.Children.Add(ContactHeading(conversation));
        var status = new TextBlock { Text = "正在读取资料…", TextWrapping = TextWrapping.Wrap };
        page.Body.Children.Add(status);
        try
        {
            var profile = await session.PostAsync("/enterprise/im/getuserinfo", new { user_id = conversation.Id });
            if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
            page.Body.Children.Remove(status);
            AddDetail(page.Body, "账号", ImgoApi.Field(profile, "account"));
            AddDetail(page.Body, "昵称", ImgoApi.Field(profile, "realname"));
            AddDetail(page.Body, "签名", ImgoApi.Field(profile, "motto"));
            AddDetail(page.Body, "备注", ImgoApi.Field(ImgoApi.Property(profile, "friendInfo"), "nickname"));
            if (ImgoApi.Property(profile, "can_view_network_info").ValueKind == System.Text.Json.JsonValueKind.True)
            {
                AddDetail(page.Body, "注册", NetworkDetail(profile, "create_time", "register_ip", "reg_location"));
                AddDetail(page.Body, "上次登录", NetworkDetail(profile, "last_login_time", "last_login_ip", "location"));
                AddDetail(page.Body, "最后一次聊天", NetworkDetail(profile, "last_chat_time", "last_chat_ip", "chat_location"));
            }
            AddConversationSideActions(page.Body, session, conversation);
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            status.Text = ErrorText(error);
        }
    }

    private void AddConversationSideActions(StackPanel panel, AccountSession session, Conversation conversation)
    {
        var actions = new WrapPanel { Margin = new Thickness(0, 8, 0, 5) };
        panel.Children.Add(actions);
        var pinned = conversation.Pinned;
        var muted = conversation.Muted;
        var pin = AddDialogButton(actions, pinned ? "取消置顶" : "置顶会话", () => { });
        pin.Click += async (_, _) => await RunChatActionAsync(async () =>
        {
            await ChangeConversationAsync(session, "setchattop", new { id = conversation.Id, is_group = conversation.Group, is_top = !pinned ? 1 : 0 });
            pinned = !pinned;
            pin.Content = pinned ? "取消置顶" : "置顶会话";
        });
        var mute = AddDialogButton(actions, muted ? "恢复消息提醒" : "消息免打扰", () => { });
        mute.Click += async (_, _) => await RunChatActionAsync(async () =>
        {
            await ChangeConversationAsync(session, "isnotice", new { id = conversation.Id, is_group = conversation.Group, is_notice = muted ? 1 : 0 });
            muted = !muted;
            mute.Content = muted ? "恢复消息提醒" : "消息免打扰";
        });
        if (conversation.Group == 0)
        {
            var remark = AddDialogButton(actions, "设置备注", () => { });
            remark.Click += async (_, _) => await EditRemarkAsync(session, conversation, push: true);
        }
        var clear = AddDialogButton(actions, "清空我的记录…", () => { });
        clear.Click += async (_, _) => await RunChatActionAsync(() => ClearConversationAsync(session, conversation));
        if (conversation.Group == 0 || conversation.Role > 1)
        {
            var remove = AddDialogButton(actions, conversation.Group == 0 ? "删除好友…" : "退出群聊…", () => { });
            remove.Click += async (_, _) => await RunChatActionAsync(() => RemoveContactAsync(session, conversation));
        }
    }
}
