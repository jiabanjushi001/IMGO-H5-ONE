using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private bool fullScreen;
    private string networkChatSummary = "";

    private async Task RefreshNetworkProfileAsync(AccountSession session, Conversation conversation)
    {
        if (!session.HasBackendRole || conversation.Group != 0 || !long.TryParse(conversation.Id, out var userId)) return;
        var generation = conversationGeneration;
        try
        {
            var profile = await session.PostAsync("/enterprise/im/getuserinfo", new { user_id = userId });
            if (visibleAccount != session || conversationGeneration != generation) return;
            bool allowed = ImgoApi.Property(profile, "can_view_network_info").ValueKind == JsonValueKind.True;
            NetworkProfileButton.Visibility = allowed ? Visibility.Visible : Visibility.Collapsed;
            if (!allowed) { ChatKind.Text = conversation.Online ? "在线" : "离线"; return; }
            var ip = ImgoApi.Field(profile, "last_chat_ip");
            var region = ImgoApi.Field(profile, ip.Length > 0 ? "chat_location" : "location");
            if (ip.Length == 0) ip = ImgoApi.Field(profile, "last_login_ip");
            var time = NetworkDetail(profile, "last_chat_time", "", "");
            networkChatSummary = $"[{(ip.Length > 0 ? ip : "暂无")}] {region} · 最近聊天 {(time.Length > 0 ? time : "暂无")}";
            ChatKind.Text = networkChatSummary;
            ChatKind.ToolTip = networkChatSummary;
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            if (visibleAccount == session && generation == conversationGeneration) NetworkProfileButton.Visibility = Visibility.Collapsed;
        }
    }

    private async void OpenNetworkProfile(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is { } session && visibleConversation is { Group: 0 } conversation)
            await ShowContactSideAsync(session, conversation);
    }
    private WindowState restoredWindowState;
    private void ShowChatRail(object sender, RoutedEventArgs e)
    {
        SetAdminVisible(false);
        SetDirectoryVisible(false);
        ConversationSearch.Focus();
    }

    private void ToggleFullScreen(object sender, RoutedEventArgs e)
    {
        if (!fullScreen)
        {
            restoredWindowState = WindowState;
            WindowState = WindowState.Normal;
            WindowStyle = WindowStyle.None;
            WindowState = WindowState.Maximized;
        }
        else
        {
            WindowStyle = WindowStyle.SingleBorderWindow;
            WindowState = restoredWindowState;
        }
        fullScreen = !fullScreen;
    }

    private async void OpenMessageManager(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is { } session && visibleConversation is { } conversation)
            await ShowArchiveAsync(session, conversation);
    }

    private async void OpenFileLibrary(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is not { } session) return;
        SetAdminVisible(false);
        SetDirectoryVisible(false);
        await ShowArchiveAsync(session, null);
    }

    private async Task ShowArchiveAsync(AccountSession session, Conversation? conversation)
    {
        var fileLibrary = conversation is null;
        var title = fileLibrary ? "文件" : "消息记录";
        var side = OpenSidePage(title);
        var panel = side.Body;
        var buttons = side.Footer;
        if (conversation is not null)
            panel.Children.Add(new TextBlock { Text = conversation.Name, Margin = new Thickness(0, 0, 0, 8),
                FontWeight = FontWeights.SemiBold, TextTrimming = TextTrimming.CharacterEllipsis });
        var categories = fileLibrary ? new[] { "所有文件", "文档", "图片", "语音", "视频", "其他文件" }
            : new[] { "全部", "文本", "图片", "视频", "文件", "语音" };
        var search = new TextBox { ToolTip = fileLibrary ? "搜索文件名" : "搜索聊天文本" };
        var category = new ComboBox { ItemsSource = categories, SelectedIndex = 0, Width = 78,
            Margin = new Thickness(0, 0, 7, 0) };
        var toolbar = new DockPanel { Margin = new Thickness(0, 0, 0, 10) };
        var searchButton = new Button { Content = "搜索", Margin = new Thickness(7, 0, 0, 0) };
        DockPanel.SetDock(searchButton, Dock.Right);
        toolbar.Children.Add(searchButton);
        toolbar.Children.Add(category);
        toolbar.Children.Add(search);
        panel.Children.Add(toolbar);
        var list = new ListBox { DisplayMemberPath = nameof(ArchiveItem.Label) };
        var scroll = (ScrollViewer)side.Root.Children[^1];
        scroll.Content = null;
        side.Root.Children.Remove(scroll);
        DockPanel.SetDock(panel, Dock.Top);
        side.Root.Children.Add(panel);
        side.Root.Children.Add(list);
        var status = new TextBlock { TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 8, 0, 0) };
        panel.Children.Add(status);
        var previous = AddDialogButton(buttons, "上一页", () => { });
        var next = AddDialogButton(buttons, "下一页", () => { });
        var open = AddDialogButton(buttons, "查看", () => { });
        var save = AddDialogButton(buttons, "保存附件", () => { });
        int pageNumber = 1;
        const int limit = 40;
        int generation = 0;
        bool busy = false;
        bool canAdvance = false;
        string keyword = "";
        int selectedCategory = 0;
        void Update()
        {
            var selected = (list.SelectedItem as ArchiveItem)?.Message;
            previous.IsEnabled = !busy && pageNumber > 1;
            next.IsEnabled = !busy && canAdvance;
            open.IsEnabled = !busy && selected is not null;
            save.IsEnabled = !busy && selected?.HasAttachment == true;
        }

        async Task LoadAsync(int page)
        {
            var request = ++generation;
            busy = true;
            canAdvance = false;
            list.ItemsSource = null;
            status.Text = "正在读取…";
            Update();
            try
            {
                ApiPage result;
                if (conversation is null)
                    result = await session.PostPageAsync("/enterprise/files/index", new
                    {
                        page, limit, keywords = keyword, cate = new[] { 0, 1, 2, 3, 4, 9 }[selectedCategory]
                    });
                else
                    result = await session.PostPageAsync("/enterprise/im/getmessagelist", new
                    {
                        toContactId = conversation.Id, is_group = conversation.Group, page, limit, keywords = keyword,
                        type = new[] { "all", "text", "image", "video", "file", "voice" }[selectedCategory]
                    });
                if (!IsSidePageOpen(side) || ActiveChatAccount != session || generation != request) return;
                var entries = result.Data.EnumerateArray().Select(row => ArchiveRow(row, session, fileLibrary)).ToList();
                list.ItemsSource = entries;
                pageNumber = page;
                canAdvance = page * (long)limit < result.Count;
                status.Text = entries.Count > 0 ? $"第 {page} 页 · 共 {result.Count} 条" : "没有匹配的记录";
            }
            catch (Exception error) when (IsRequestFailure(error))
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                if (generation == request) status.Text = ErrorText(error);
            }
            finally
            {
                if (generation == request) { busy = false; Update(); }
            }
        }

        async Task SearchAsync()
        {
            keyword = search.Text.Trim();
            selectedCategory = category.SelectedIndex;
            if (!fileLibrary && keyword.Length > 0 && selectedCategory > 1)
            {
                category.SelectedIndex = 1;
                return;
            }
            await LoadAsync(1);
        }

        async Task OpenAsync(bool saveOnly)
        {
            if (busy || list.SelectedItem is not ArchiveItem { Message: var message }) return;
            busy = true;
            Update();
            try
            {
                if (saveOnly && message.HasAttachment) await SaveAttachmentAsync(session, message, this);
                else if (message.IsImage && message.HasAttachment) await PreviewImageAsync(session, message, this, () => IsSidePageOpen(side));
                else if (message.IsPlayable && message.HasAttachment) await PlayMediaAsync(session, message, this);
                else if (message.HasAttachment) await SaveAttachmentAsync(session, message, this);
                else
                {
                    var detail = OpenSidePage("消息内容", push: true);
                    AddDetail(detail.Body, "发送者", message.Sender);
                    AddDetail(detail.Body, "时间", message.Time);
                    detail.Body.Children.Add(new TextBox { IsReadOnly = true, TextWrapping = TextWrapping.Wrap,
                        Text = message.Body, MaxHeight = 310, VerticalScrollBarVisibility = ScrollBarVisibility.Auto });
                }
            }
            catch (Exception error) when (IsAttachmentFailure(error) || error is ArgumentException or NotSupportedException or FormatException)
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                status.Text = error is NotSupportedException or FormatException ? "无法预览，可保存后查看" : ErrorText(error);
            }
            finally { busy = false; Update(); }
        }
        searchButton.Click += async (_, _) => await SearchAsync();
        search.KeyDown += async (_, e) => { if (e.Key == Key.Enter) { e.Handled = true; await SearchAsync(); } };
        category.SelectionChanged += async (_, _) =>
        {
            if (!fileLibrary && category.SelectedIndex > 1) search.Clear();
            await SearchAsync();
        };
        previous.Click += async (_, _) => await LoadAsync(pageNumber - 1);
        next.Click += async (_, _) => await LoadAsync(pageNumber + 1);
        list.SelectionChanged += (_, _) => Update();
        open.Click += async (_, _) => await OpenAsync(false);
        save.Click += async (_, _) => await OpenAsync(true);
        list.MouseDoubleClick += async (_, _) => await OpenAsync(false);
        search.Focus();
        await LoadAsync(1);
    }

    private static ArchiveItem ArchiveRow(JsonElement row, AccountSession session, bool fileLibrary)
    {
        if (fileLibrary)
        {
            var filename = ImgoApi.Field(row, "name");
            var kind = ImgoApi.Number(row, "cate") == 3 ? "voice" : ImgoApi.Field(row, "msg_type");
            var message = new ChatMessage(ImgoApi.Number(row, "file_id"), "", "", "", false,
                Type: kind, FileName: filename, Download: ImgoApi.Field(row, "download"));
            return new ArchiveItem(filename, message);
        }
        var chat = ReadChatMessage(row, session, 2);
        var label = chat.HasAttachment ? $"{chat.Body} {chat.AttachmentLabel}" : chat.Body;
        label = label.Replace("\r", " ").Replace("\n", " ");
        if (label.Length > 100) label = label[..100] + "…";
        return new ArchiveItem($"{chat.Time}　{chat.Sender}　{label}", chat);
    }

    private sealed record ArchiveItem(string Label, ChatMessage Message);
}
