using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private async Task ShowGroupInviteAsync(AccountSession session, Conversation? group, bool push = false, TextBlock? ownerStatus = null)
    {
        bool creating = group is null;
        var side = OpenSidePage(creating ? "创建群聊" : "邀请群成员", push);
        var panel = side.Body;
        var buttons = side.Footer;
        const int pageSize = 40;
        const int maxSelected = 100;
        int page = 1;
        long total = 0;
        string keyword = "";
        bool busy = false;
        bool pageLoaded = false;
        IReadOnlyList<ChatChoice>? legacyChoices = null;
        var selected = new HashSet<string>(StringComparer.Ordinal);
        var search = new TextBox { ToolTip = "输入昵称或账号，按回车搜索" };
        var searchButton = new Button { Content = "搜索", Margin = new Thickness(8, 0, 0, 0), MinWidth = 64 };
        var searchRow = new DockPanel();
        DockPanel.SetDock(searchButton, Dock.Right);
        searchRow.Children.Add(searchButton);
        searchRow.Children.Add(search);
        var name = new TextBox { MaxLength = 64, Style = (Style)FindResource("ChatInput") };
        if (creating)
        {
            panel.Children.Add(new Label { Content = "群名称", Target = name });
            panel.Children.Add(name);
            panel.Children.Add(new TextBlock { Text = "选择成员（选填）", Margin = new Thickness(0, 12, 0, 5) });
        }
        else
            AddDetail(panel, "群聊", group!.Name);
        panel.Children.Add(searchRow);

        var choices = new StackPanel();
        var scroll = new ScrollViewer { Content = choices, MaxHeight = 380,
            Margin = new Thickness(0, 10, 0, 0), VerticalScrollBarVisibility = ScrollBarVisibility.Auto };
        panel.Children.Add(scroll);
        var summary = new TextBlock { Margin = new Thickness(0, 8, 0, 0) };
        var status = new TextBlock { TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 5, 0, 0) };
        panel.Children.Add(summary);
        panel.Children.Add(status);

        var previous = AddDialogButton(buttons, "上一页", () => { });
        var next = AddDialogButton(buttons, "下一页", () => { });
        var submit = AddDialogButton(buttons, creating ? "创建群聊" : "邀请", () => { });
        name.TextChanged += (_, _) => UpdateControls();
        submit.Style = (Style)FindResource("ChatPrimaryAction");
        previous.Click += async (_, _) => await LoadPageAsync(page - 1);
        next.Click += async (_, _) => await LoadPageAsync(page + 1);
        searchButton.Click += async (_, _) => await LoadPageAsync(1, search.Text.Trim());
        search.KeyDown += async (_, args) =>
        {
            if (args.Key != Key.Enter) return;
            args.Handled = true;
            await LoadPageAsync(1, search.Text.Trim());
        };
        submit.Click += async (_, _) =>
        {
            if (busy || ActiveChatAccount != session || (creating ? string.IsNullOrWhiteSpace(name.Text) : selected.Count == 0)) return;
            busy = true;
            UpdateControls();
            status.Text = creating ? "正在创建…" : "正在邀请…";
            try
            {
                if (creating)
                {
                    var created = await session.PostAsync("/enterprise/group/add", new { name = name.Text.Trim(), user_ids = selected.ToArray() });
                    await OpenGroupAfterActionAsync(side, session, ImgoApi.Field(created, "id"), "群聊已创建");
                }
                else
                {
                    await session.PostAsync("/enterprise/group/addgroupuser", new { group_id = group!.Id, user_ids = selected.ToArray() });
                    if (ownerStatus is not null) ownerStatus.Text = "已添加群成员";
                    RefreshAfterAction(session, "已添加群成员");
                    if (IsSidePageOpen(side)) BackConversationPane();
                }
            }
            catch (Exception error) when (IsRequestFailure(error))
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                status.Text = ErrorText(error);
            }
            finally { busy = false; UpdateControls(); }
        };
        await LoadPageAsync(1);
        if (IsSidePageOpen(side)) (creating ? name : search).Focus();

        void UpdateControls()
        {
            summary.Text = $"已选 {selected.Count} / {maxSelected} · 第 {page} 页 · 共 {total} 人";
            search.IsEnabled = searchButton.IsEnabled = choices.IsEnabled = !busy;
            previous.IsEnabled = !busy && page > 1;
            next.IsEnabled = !busy && (long)page * pageSize < total;
            name.IsEnabled = !busy;
            submit.IsEnabled = !busy && (creating ? !string.IsNullOrWhiteSpace(name.Text) : selected.Count > 0);
        }

        async Task LoadPageAsync(int target, string? newKeyword = null)
        {
            if (busy || target < 1) return;
            busy = true;
            UpdateControls();
            status.Text = "读取候选成员…";
            string filter = newKeyword ?? keyword;
            try
            {
                IReadOnlyList<ChatChoice> matches;
                long count;
                if (legacyChoices is null)
                {
                    var response = await session.PostPageAsync("/enterprise/group/getalluser",
                        new { group_id = group?.Id ?? "", keywords = filter, page = target, limit = pageSize });
                    matches = ChatPeople(response.Data);
                    count = response.Count;
                    if (!pageLoaded && count == 0 && matches.Count > 0)
                        legacyChoices = matches;
                }
                else { matches = []; count = 0; }

                if (legacyChoices is not null)
                {
                    var allMatches = legacyChoices.Where(item => item.Label.Contains(filter, StringComparison.OrdinalIgnoreCase)).ToList();
                    count = allMatches.Count;
                    matches = allMatches.Skip((target - 1) * pageSize).Take(pageSize).ToList();
                }

                if (!IsSidePageOpen(side) || ActiveChatAccount != session) return;
                page = target;
                total = count;
                keyword = filter;
                pageLoaded = true;
                choices.Children.Clear();
                foreach (var item in matches)
                {
                    var check = new CheckBox { Content = item.Label, IsChecked = selected.Contains(item.Id),
                        Margin = new Thickness(0, 5, 0, 5) };
                    check.Checked += (_, _) =>
                    {
                        if (selected.Count >= maxSelected)
                        {
                            check.IsChecked = false;
                            status.Text = $"最多选择 {maxSelected} 人";
                            return;
                        }
                        selected.Add(item.Id);
                        UpdateControls();
                    };
                    check.Unchecked += (_, _) => { selected.Remove(item.Id); UpdateControls(); };
                    choices.Children.Add(check);
                }
                scroll.ScrollToTop();
                status.Text = matches.Count == 0 ? "没有匹配的候选成员" : "";
            }
            catch (Exception error) when (IsRequestFailure(error))
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                status.Text = ErrorText(error);
            }
            finally { busy = false; UpdateControls(); }
        }
    }
}
