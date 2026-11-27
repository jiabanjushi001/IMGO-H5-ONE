using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private Task AddFriendAsync(AccountSession session)
    {
        var page = OpenSidePage("添加好友");
        var account = new TextBox { MaxLength = 100, Style = (Style)FindResource("ChatInput") };
        page.Body.Children.Add(new Label { Content = "对方的账号", Target = account });
        page.Body.Children.Add(account);
        var result = new StackPanel { Margin = new Thickness(0, 16, 0, 0) };
        var status = new TextBlock { TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 8, 0, 0) };
        page.Body.Children.Add(result);
        page.Body.Children.Add(status);
        var search = AddDialogButton(page.Footer, "搜索", () => { });
        search.Style = (Style)FindResource("ChatPrimaryAction");
        search.Click += async (_, _) => await SearchAsync();
        account.KeyDown += async (_, args) =>
        {
            if (args.Key != Key.Enter) return;
            args.Handled = true;
            await SearchAsync();
        };
        account.TextChanged += (_, _) => { result.Children.Clear(); status.Text = ""; };
        account.Focus();
        return Task.CompletedTask;

        async Task SearchAsync()
        {
            if (!search.IsEnabled || ActiveChatAccount != session) return;
            var value = account.Text.Trim();
            if (value.Length == 0) { status.Text = "请输入对方的账号"; return; }
            result.Children.Clear();
            search.IsEnabled = account.IsEnabled = false;
            status.Text = "正在搜索…";
            try
            {
                var users = await session.PostAsync("/enterprise/im/searchuser", new { account = value });
                if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
                if (users.ValueKind != JsonValueKind.Array || users.GetArrayLength() == 0)
                {
                    status.Text = "未找到这个账号";
                    return;
                }

                var person = users[0];
                var userId = ImgoApi.Number(person, "user_id");
                status.Text = "";
                result.Children.Add(new TextBlock { Text = PersonName(person), FontSize = 16,
                    FontWeight = FontWeights.SemiBold, TextWrapping = TextWrapping.Wrap });
                var remark = new TextBox { MaxLength = 100, Style = (Style)FindResource("ChatInput") };
                result.Children.Add(new Label { Content = "验证消息（选填）", Target = remark });
                result.Children.Add(remark);
                var actions = new WrapPanel { Margin = new Thickness(0, 8, 0, 0) };
                result.Children.Add(actions);
                var send = AddDialogButton(actions, "发送申请", () => { });
                send.Style = (Style)FindResource("ChatPrimaryAction");
                send.Click += async (_, _) =>
                {
                    if (ActiveChatAccount != session || userId <= 0) return;
                    send.IsEnabled = search.IsEnabled = account.IsEnabled = remark.IsEnabled = false;
                    try
                    {
                        await session.PostAsync("/enterprise/friend/add", new { user_id = userId, remark = remark.Text.Trim() });
                        status.Text = "好友申请已发送";
                        RefreshAfterAction(session, status.Text);
                    }
                    catch (Exception error) when (IsRequestFailure(error))
                    {
                        SetContactActionError(session, status, error);
                        send.IsEnabled = remark.IsEnabled = true;
                    }
                    finally { search.IsEnabled = account.IsEnabled = true; }
                };
            }
            catch (Exception error) when (IsRequestFailure(error)) { SetContactActionError(session, status, error); }
            finally { search.IsEnabled = account.IsEnabled = true; }
        }
    }

    private async Task FriendRequestsAsync(AccountSession session)
    {
        var page = OpenSidePage("好友申请");
        var requests = new StackPanel();
        var status = new TextBlock { TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 8, 0, 0) };
        page.Body.Children.Add(requests);
        page.Body.Children.Add(status);
        var previous = AddDialogButton(page.Footer, "上一页", () => { });
        var next = AddDialogButton(page.Footer, "下一页", () => { });
        const int limit = 30;
        int current = 1;
        long count = 0;
        bool busy = false;
        previous.Click += async (_, _) => await LoadAsync(current - 1);
        next.Click += async (_, _) => await LoadAsync(current + 1);
        await LoadAsync(1);

        void Update()
        {
            requests.IsEnabled = !busy;
            previous.IsEnabled = !busy && current > 1;
            next.IsEnabled = !busy && current * (long)limit < count;
        }

        async Task LoadAsync(int target, string feedback = "")
        {
            if (busy || target < 1 || ActiveChatAccount != session) return;
            busy = true;
            Update();
            status.Text = "正在读取申请…";
            try
            {
                var response = await session.PostPageAsync("/enterprise/friend/index", new { page = target, limit });
                if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
                current = target;
                count = response.Count;
                requests.Children.Clear();
                foreach (var item in response.Data.EnumerateArray())
                {
                    var id = ImgoApi.Field(item, "friend_id");
                    var state = ImgoApi.Number(item, "status");
                    var row = new StackPanel { Margin = new Thickness(0, 0, 0, 10) };
                    row.Children.Add(new TextBlock { Text = PersonName(ImgoApi.Property(item, "create_user_info")),
                        FontWeight = FontWeights.SemiBold, TextWrapping = TextWrapping.Wrap });
                    var label = state switch { 1 => "已接受", 2 => "待处理", _ => "已拒绝" };
                    var remark = ImgoApi.Field(item, "remark");
                    row.Children.Add(new TextBlock { Text = remark.Length == 0 ? label : $"{label} · {remark}", TextWrapping = TextWrapping.Wrap });
                    if (state == 2 && id.Length > 0)
                    {
                        var actions = new WrapPanel { Margin = new Thickness(0, 5, 0, 0) };
                        row.Children.Add(actions);
                        AddDialogButton(actions, "同意", () => { }).Click += async (_, _) => await DecideAsync(id, 1);
                        AddDialogButton(actions, "拒绝", () => { }).Click += async (_, _) => await DecideAsync(id, 0);
                    }
                    requests.Children.Add(row);
                }
                status.Text = requests.Children.Count == 0 ? "暂无好友申请" : $"第 {current} 页";
                if (count > 0) status.Text += $" · 共 {count} 条";
                if (feedback.Length > 0) status.Text = $"{feedback} · {status.Text}";
            }
            catch (Exception error) when (IsRequestFailure(error)) { SetContactActionError(session, status, error); }
            finally { busy = false; Update(); }
        }

        async Task DecideAsync(string id, int decision)
        {
            if (busy || ActiveChatAccount != session) return;
            busy = true;
            Update();
            try
            {
                await session.PostAsync("/enterprise/friend/update", new { friend_id = id, status = decision });
                var feedback = decision == 1 ? "已同意好友申请" : "已拒绝好友申请";
                RefreshAfterAction(session, feedback);
                await LoadFriendRequestCountAsync(session);
                busy = false;
                if (IsSidePageOpen(page)) await LoadAsync(current, feedback);
            }
            catch (Exception error) when (IsRequestFailure(error)) { SetContactActionError(session, status, error); }
            finally { busy = false; Update(); }
        }
    }

    private Task JoinGroupAsync(AccountSession session)
    {
        var page = OpenSidePage("按群号加入");
        var entry = new TextBox { MaxLength = 19, Style = (Style)FindResource("ChatInput") };
        var status = new TextBlock { TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 8, 0, 0) };
        page.Body.Children.Add(new Label { Content = "群号", Target = entry });
        page.Body.Children.Add(entry);
        page.Body.Children.Add(status);
        var join = AddDialogButton(page.Footer, "加入群聊", () => { });
        join.Style = (Style)FindResource("ChatPrimaryAction");
        join.Click += async (_, _) => await JoinAsync();
        entry.KeyDown += async (_, args) =>
        {
            if (args.Key != Key.Enter) return;
            args.Handled = true;
            await JoinAsync();
        };
        entry.Focus();
        return Task.CompletedTask;

        async Task JoinAsync()
        {
            if (!join.IsEnabled || ActiveChatAccount != session) return;
            if (!long.TryParse(entry.Text.Trim(), out var id) || id <= 0)
            {
                status.Text = "请输入有效的数字群号";
                return;
            }
            join.IsEnabled = entry.IsEnabled = false;
            status.Text = "正在加入…";
            try
            {
                await session.PostAsync("/enterprise/group/joingroup", new { group_id = id });
                await OpenGroupAfterActionAsync(page, session, $"group-{id}", "已加入群聊");
            }
            catch (Exception error) when (IsRequestFailure(error)) { SetContactActionError(session, status, error); }
            finally { join.IsEnabled = entry.IsEnabled = true; }
        }
    }

    private async Task OpenGroupAfterActionAsync(SidePage page, AccountSession session, string id, string feedback)
    {
        RefreshAfterAction(session, feedback);
        if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
        await TryReloadContactsAsync(session);
        if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
        CloseConversationPane();
        if (session.Conversations.Find(item => item.Id == id) is { } conversation)
            ActivateDirectoryConversation(conversation);
    }

    private void SetContactActionError(AccountSession session, TextBlock status, Exception error)
    {
        if (error is ApiException { SessionExpired: true }) session.Expire();
        status.Text = ErrorText(error);
    }
}
