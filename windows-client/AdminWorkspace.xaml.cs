using System.Diagnostics;
using System.Net.Http;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Controls.Primitives;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Shapes;

namespace Imgo.Desktop;

public partial class AdminWorkspace : UserControl
{
    private const int PageSize = 30;
    private AccountSession? account;
    private AdminPage? page;
    private int pageNumber = 1;
    private int generation;
    private int requestSequence;
    private long total;
    private JsonElement cardData;
    private bool mutating;

    public AdminWorkspace() => InitializeComponent();

    internal static bool CanOpen(AccountSession session) =>
        !session.Expired && AdminPages.All.Any(page => AdminPages.IsAllowed(page, session));

    internal void SetAccount(AccountSession? session)
    {
        generation++;
        account = session;
        Menu.ItemsSource = session is not null && !session.Expired
            ? AdminPages.All.Where(item => AdminPages.IsAllowed(item, session)).ToArray() : [];
        Table.ItemsSource = null;
        CardBody.Children.Clear();
        if (Menu.Items.Count > 0) Menu.SelectedIndex = 0;
    }

    private async void PageChanged(object sender, SelectionChangedEventArgs e)
    {
        generation++;
        page = Menu.SelectedItem as AdminPage;
        if (page is null) return;
        Heading.Text = page.Title;
        Keyword.Text = "";
        Keyword.Visibility = page.Searchable ? Visibility.Visible : Visibility.Collapsed;
        Table.Visibility = page.Columns.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
        Cards.Visibility = page.Columns.Length == 0 ? Visibility.Visible : Visibility.Collapsed;
        var filters = page.Title switch
        {
            "成员管理" => new[] { "不限层级", "直属下级", "全部下级" },
            "系统概况" => new[] { "今日在线", "近 7 天在线", "近 30 天在线" },
            "日志" => new[] { "全部等级", "低敏", "中敏", "高敏", "严重" },
            "提现订单" => new[] { "全部状态", "待处理", "已打款", "已拒绝", "已冻结" },
            "实名认证" or "绑卡管理" => new[] { "全部状态", "待审核", "已通过", "已拒绝" },
            _ => []
        };
        Filter.ItemsSource = filters;
        Filter.SelectedIndex = 0;
        Filter.Visibility = filters.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
        SearchButton.Content = page.Searchable ? "查询" : "应用";
        SearchButton.Visibility = filters.Length > 0 || page.Searchable ? Visibility.Visible : Visibility.Collapsed;
        Table.ItemsSource = null;
        CardBody.Children.Clear();
        Table.Columns.Clear();
        for (var i = 0; i < page.Columns.Length; i++)
        {
            var column = page.Columns[i];
            Table.Columns.Add(new DataGridTextColumn
            {
                Header = column.Label,
                Binding = new Binding($"Values[{i}]") { Mode = BindingMode.OneWay },
                MinWidth = column.Format == "date" ? 130 : 68,
                Width = column.Path is "content" or "remark" or "detail" or "name" ? new DataGridLength(1, DataGridLengthUnitType.Star) : DataGridLength.Auto,
                ElementStyle = (Style)FindResource("TableText"),
                MaxWidth = 520
            });
        }
        pageNumber = 1;
        cardData = default;
        total = 0;
        UpdateControls();
        await LoadAsync();
    }

    private Dictionary<string, object> Parameters(AdminPage definition)
    {
        var payload = new Dictionary<string, object>();
        if (definition.Paged)
        {
            payload["page"] = pageNumber;
            payload["limit"] = PageSize;
        }
        if (definition.Searchable) payload["keywords"] = Keyword.Text.Trim();
        if (definition.Title == "成员管理" && Filter.SelectedIndex > 0)
            payload["referral_scope"] = Filter.SelectedIndex == 1 ? "direct" : "all";
        if (definition.Title == "系统概况")
            payload["online_days"] = Filter.SelectedIndex switch { 1 => 7, 2 => 30, _ => 1 };
        if (definition.Title == "日志" && Filter.SelectedIndex > 0)
            payload["risk_level"] = new[] { "", "low", "medium", "high", "critical" }[Filter.SelectedIndex];
        if (Filter.SelectedIndex > 0 && definition.Title is "提现订单" or "实名认证" or "绑卡管理")
            payload["status"] = Filter.SelectedIndex - 1;
        if (definition.Title == "系统设置") payload["name"] = "sysInfo";
        return payload;
    }

    private async Task LoadAsync()
    {
        if (account is not { Expired: false } session || page is not { } definition || mutating) return;
        var stamp = generation;
        var request = ++requestSequence;
        Footer.Text = "正在读取…";
        Previous.IsEnabled = Next.IsEnabled = false;
        try
        {
            var result = await session.GetAdminPageAsync(definition.Api, Parameters(definition));
            if (stamp != generation || request != requestSequence) return;
            if (definition.Columns.Length == 0)
            {
                cardData = result.Data;
                RenderCards(definition, result.Data);
            }
            else
                Table.ItemsSource = result.Data.EnumerateArray()
                    .Select(row => new AdminRow(row, definition.Columns.Select(column => Format(row, column)).ToArray())).ToList();
            total = definition.Paged ? result.Count : (Table.ItemsSource as IReadOnlyCollection<AdminRow>)?.Count ?? 0;
            Footer.Text = definition.Paged ? $"第 {pageNumber} 页 · 共 {total} 条" : "";
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (stamp == generation && request == requestSequence)
            {
                Table.ItemsSource = null;
                cardData = default;
                CardBody.Children.Clear();
                Footer.Text = FailureText(exception);
            }
        }
        finally
        {
            if (stamp == generation && request == requestSequence) UpdateControls();
        }
    }

    private void RowChanged(object sender, SelectionChangedEventArgs e) => UpdateControls();

    private void UpdateControls()
    {
        if (Primary is null) return;
        var title = page?.Title;
        var selected = (Table.SelectedItem as AdminRow)?.Data ?? default;
        var primaryLabel = title switch
        {
            "成员管理" => "编辑备注",
            "群聊管理" => "群号加入设置",
            "系统公告" => "发布公告",
            "角色" => "添加角色",
            "系统设置" when account?.IsSuperAdmin == true => "编辑基础设置",
            "签到设置" => "编辑签到设置",
            _ => ""
        };
        Primary.Content = primaryLabel;
        Primary.Visibility = primaryLabel.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
        Primary.IsEnabled = !mutating && account is { Expired: false } &&
            (title is "系统公告" or "角色" || title is "系统设置" or "签到设置" && cardData.ValueKind == JsonValueKind.Object || selected.ValueKind == JsonValueKind.Object &&
                (title != "群聊管理" || ImgoApi.Property(selected, "can_set_number_join").ValueKind == JsonValueKind.True));
        var secondaryLabel = title switch
        {
            "角色" => "编辑角色",
            "成员管理" when account?.IsSuperAdmin == true => "设置角色",
            "日志" => "查看详情",
            "系统公告" => "删除公告",
            _ => ""
        };
        Secondary.Content = secondaryLabel;
        Secondary.Visibility = secondaryLabel.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
        Secondary.IsEnabled = !mutating && selected.ValueKind == JsonValueKind.Object &&
            (title != "角色" || ImgoApi.Number(selected, "role_id") > 0) &&
            (title != "成员管理" || ImgoApi.Number(selected, "user_id") > 1);
        var extraLabel = title switch { "成员管理" => "添加成员", "群聊管理" => "添加群成员", _ => "" };
        Extra.Content = extraLabel;
        Extra.Visibility = extraLabel.Length > 0 ? Visibility.Visible : Visibility.Collapsed;
        Extra.IsEnabled = !mutating && account is { Expired: false } &&
            (title == "成员管理" || selected.ValueKind == JsonValueKind.Object);
        Previous.Visibility = Next.Visibility = page is { Paged: true } ? Visibility.Visible : Visibility.Collapsed;
        Previous.IsEnabled = !mutating && pageNumber > 1;
        Next.IsEnabled = !mutating && pageNumber * (long)PageSize < total;
    }

    private async void Refresh(object sender, RoutedEventArgs e) => await LoadAsync();
    private async void Search(object sender, RoutedEventArgs e) => await SearchAsync();
    private async void SearchKey(object sender, KeyEventArgs e) { if (e.Key == Key.Enter) await SearchAsync(); }
    private async Task SearchAsync() { pageNumber = 1; await LoadAsync(); }
    private async void PreviousPage(object sender, RoutedEventArgs e) { if (pageNumber > 1) { pageNumber--; await LoadAsync(); } }
    private async void NextPage(object sender, RoutedEventArgs e) { if (pageNumber * (long)PageSize < total) { pageNumber++; await LoadAsync(); } }

    private void OpenWeb(object sender, RoutedEventArgs e)
    {
        if (account is null || page is null) return;
        var target = new Uri(account.Server, "/#" + page.Web);
        try { Process.Start(new ProcessStartInfo(target.AbsoluteUri) { UseShellExecute = true }); }
        catch (System.ComponentModel.Win32Exception) { Footer.Text = "无法打开默认浏览器。"; }
    }

    private async void PrimaryAction(object sender, RoutedEventArgs e)
    {
        if (account is not { Expired: false } session || page is not { } definition) return;
        var row = (Table.SelectedItem as AdminRow)?.Data ?? default;
        switch (definition.Title)
        {
            case "成员管理" when row.ValueKind == JsonValueKind.Object:
                var remark = Input(ImgoApi.Field(row, "remark"), 191, true);
                if (ShowDialog($"编辑备注 · {ImgoApi.Field(row, "account")}", [("备注", remark)], "保存"))
                    await SaveAsync(session, "/manage/user/setremark", new
                    {
                        user_id = ImgoApi.Number(row, "user_id"),
                        remark = remark.Text.Trim()
                    });
                break;
            case "群聊管理" when ImgoApi.Property(row, "can_set_number_join").ValueKind == JsonValueKind.True:
                var join = new CheckBox
                {
                    Content = "输入群号即可加入私有群",
                    IsChecked = ImgoApi.Number(row, "number_join") == 1,
                    Margin = new Thickness(0, 8, 0, 8)
                };
                if (ShowDialog($"群号加入设置 · {ImgoApi.Field(row, "name")}", [("", join)], "保存"))
                    await SaveAsync(session, "/manage/group/setnumberjoin", new
                    {
                        group_id = ImgoApi.Number(row, "group_id"),
                        enabled = join.IsChecked == true ? 1 : 0
                    });
                break;
            case "系统公告":
                var title = Input("", 200);
                var content = Input("", 100000, true);
                if (ShowDialog("发布系统公告", [("标题", title), ("内容", content)], "发布",
                    () => title.Text.Trim().Length > 0 && content.Text.Trim().Length > 0 ? "" : "请填写标题和内容。"))
                    await SaveAsync(session, "/manage/index/publishnotice", new { title = title.Text.Trim(), content = content.Text.Trim() });
                break;
            case "系统设置" when session.IsSuperAdmin:
                await EditBasicAsync(session);
                break;
            case "签到设置":
                await EditCheckInAsync(session);
                break;
            case "角色":
                await EditRoleAsync(session, 0);
                break;
        }
    }

    private async void SecondaryAction(object sender, RoutedEventArgs e)
    {
        if (account is not { Expired: false } session || page is not { } definition ||
            Table.SelectedItem is not AdminRow row) return;
        if (definition.Title == "成员管理" && session.IsSuperAdmin)
            await AssignRoleAsync(session, row.Data);
        else if (definition.Title == "角色")
            await EditRoleAsync(session, ImgoApi.Number(row.Data, "role_id"));
        else if (definition.Title == "系统公告")
        {
            var title = ImgoApi.Field(row.Data, "title");
            if (MessageBox.Show(Window.GetWindow(this), $"删除公告「{title}」？", "系统公告",
                MessageBoxButton.YesNo, MessageBoxImage.Warning, MessageBoxResult.No) == MessageBoxResult.Yes)
                await SaveAsync(session, "/manage/index/delnotice", new { id = ImgoApi.Number(row.Data, "msg_id") });
        }
        else if (definition.Title == "日志")
        {
            var fields = new[] { ("请求方法", "request_method"), ("路径", "request_path"), ("操作说明", "detail"),
                ("请求头", "request_headers"), ("请求数据", "request_data"), ("错误", "error_message") };
            var details = Input(string.Join(Environment.NewLine + Environment.NewLine, fields.Select(field =>
            {
                var value = ImgoApi.Property(row.Data, field.Item2);
                return $"{field.Item1}{Environment.NewLine}{(value.ValueKind == JsonValueKind.String ? value.GetString() : value.ToString())}";
            })), 0, true);
            details.IsReadOnly = true;
            details.Height = 270;
            ShowDialog("日志详情", [("", details)], "关闭");
        }
    }

    private async void ExtraAction(object sender, RoutedEventArgs e)
    {
        if (account is not { Expired: false } session) return;
        if (page?.Title == "成员管理")
        {
            var username = Input("", 32);
            var name = Input("", 100);
            var password = new PasswordBox { MinHeight = 32, MaxLength = 72 };
            var inviter = Input("", 6);
            if (!ShowDialog("添加成员", [("用户名", username), ("昵称", name), ("密码", password),
                ("上级邀请码（选填）", inviter)], "添加",
                () => username.Text.Trim().Length >= 3 && password.Password.Length > 0 ? "" : "请填写用户名和密码。")) return;
            var newPassword = password.Password;
            password.Clear();
            await SaveAsync(session, "/manage/user/add", new
            {
                account = username.Text.Trim(), realname = name.Text.Trim(), password = newPassword,
                parent_invite_code = inviter.Text.Trim()
            });
        }
        else if (page?.Title == "群聊管理" && Table.SelectedItem is AdminRow row)
        {
            var input = Input("", 2000, true);
            var ids = Array.Empty<long>();
            if (ShowDialog($"添加群成员 · {ImgoApi.Field(row.Data, "name")}",
                [("用户 ID，可用逗号、空格或换行分隔", input)], "添加",
                () =>
                {
                    var parts = input.Text.Split([',', '，', ' ', (char)9, (char)10, (char)13],
                        StringSplitOptions.RemoveEmptyEntries);
                    ids = parts.Select(part => long.TryParse(part, out var value) ? value : 0).Distinct().ToArray();
                    return ids.Length > 0 && ids.All(id => id > 0) ? "" : "请输入有效的用户 ID。";
                }))
                await SaveAsync(session, "/manage/group/addgroupuser", new
                {
                    group_id = ImgoApi.Number(row.Data, "group_id"), user_ids = ids
                });
        }
    }

    private async Task AssignRoleAsync(AccountSession session, JsonElement user)
    {
        var stamp = generation;
        var userId = ImgoApi.Number(user, "user_id");
        if (userId <= 1) return;
        try
        {
            var catalog = await session.PostAsync("/manage/role/index", new { });
            if (stamp != generation) return;
            var options = catalog.EnumerateArray()
                .Where(item => ImgoApi.Number(item, "role_id") >= 0 && ImgoApi.Number(item, "status") == 1)
                .Select(item => new RoleOption(ImgoApi.Number(item, "role_id"), ImgoApi.Field(item, "name"))).ToArray();
            var roles = new ComboBox { ItemsSource = options, DisplayMemberPath = nameof(RoleOption.Name), MinHeight = 32 };
            roles.SelectedItem = options.FirstOrDefault(item => item.Id == ImgoApi.Number(user, "admin_role_id"));
            if (ShowDialog($"设置角色 · {ImgoApi.Field(user, "account")}", [("角色", roles)], "保存",
                    () => roles.SelectedItem is RoleOption ? "" : "请选择角色。") && roles.SelectedItem is RoleOption option)
                await SaveAsync(session, "/manage/user/setrole", new { user_id = userId, admin_role_id = option.Id });
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (stamp == generation) Footer.Text = FailureText(exception);
        }
    }

    private async Task EditBasicAsync(AccountSession session)
    {
        var name = Input(ImgoApi.Field(cardData, "name"), 100);
        var description = Input(ImgoApi.Field(cardData, "description"), 500, true);
        var download = Input(ImgoApi.Field(cardData, "clientDownloadUrl"), 2048);
        if (!ShowDialog("编辑基础设置", [("系统名称", name), ("系统描述", description), ("客户端地址", download)],
                "保存", () => name.Text.Trim().Length > 0 ? "" : "请输入系统名称。")) return;
        var stamp = generation;
        try
        {
            var latest = await session.PostAsync("/manage/config/getconfig", new { name = "sysInfo" });
            if (stamp != generation) return;
            var values = System.Text.Json.Nodes.JsonNode.Parse(latest.GetRawText())?.AsObject();
            if (values is null) throw new InvalidOperationException("读取系统设置失败。");
            values["name"] = name.Text.Trim();
            values["description"] = description.Text.Trim();
            values["clientDownloadUrl"] = download.Text.Trim();
            await SaveAsync(session, "/manage/config/setconfig", new { name = "sysInfo", value = values });
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (stamp == generation) Footer.Text = FailureText(exception);
        }
    }

    private static readonly (string Label, string Key)[] SignFields =
    [
        ("签到周期（天）", "cycleDays"), ("高额奖励天数", "highDays"),
        ("高额奖励下限（元）", "highMinYuan"), ("高额奖励上限（元）", "highMaxYuan"),
        ("普通奖励下限（元）", "normalMinYuan"), ("普通奖励上限（元）", "normalMaxYuan")
    ];

    private async Task EditCheckInAsync(AccountSession session)
    {
        var enabled = new CheckBox { Content = "开启签到奖励", IsChecked = ImgoApi.Number(cardData, "rewardEnabled") == 1 };
        var inputs = SignFields.Select(field => Input(ImgoApi.Field(cardData, field.Key), 18)).ToArray();
        var controls = SignFields.Select((field, index) => (field.Label, (FrameworkElement)inputs[index])).ToList();
        controls.Insert(0, ("", enabled));
        if (!ShowDialog("编辑签到设置", controls.ToArray(), "保存",
                () => inputs.All(input => decimal.TryParse(input.Text, out _)) ? "" : "请填写有效数字。")) return;
        var values = SignFields.Select((field, index) => (field.Key, Value: (object)decimal.Parse(inputs[index].Text)))
            .ToDictionary(field => field.Key, field => field.Value);
        values["rewardEnabled"] = enabled.IsChecked == true ? 1 : 0;
        await SaveAsync(session, "/manage/config/setcheckinsettings", values);
    }

    private sealed record RoleOption(long Id, string Name);

    private async Task EditRoleAsync(AccountSession session, long roleId)
    {
        var stamp = generation;
        try
        {
            var catalogTask = session.PostAsync("/manage/role/permissions", new { });
            var roleTask = roleId > 0 ? session.PostAsync("/manage/role/detail", new { role_id = roleId })
                : Task.FromResult(default(JsonElement));
            await Task.WhenAll(catalogTask, roleTask);
            if (stamp != generation) return;
            var role = await roleTask;
            var current = ImgoApi.Property(role, "permissions");
            var keys = current.ValueKind == JsonValueKind.Array
                ? current.EnumerateArray().Select(item => item.GetString()).ToHashSet() : [];
            var name = Input(ImgoApi.Field(role, "name"), 64);
            var remark = Input(ImgoApi.Field(role, "remark"), 255);
            var active = new CheckBox { Content = "启用", IsChecked = ImgoApi.Number(role, "status", 1) == 1 };
            var agent = new CheckBox { Content = "代理模式", IsChecked = ImgoApi.Number(role, "agent_mode") == 1 };
            var permissions = new WrapPanel();
            var boxes = new List<(string Key, CheckBox Box)>();
            foreach (var item in (await catalogTask).EnumerateArray())
            {
                var key = ImgoApi.Field(item, "permission_key");
                var box = new CheckBox { Content = ImgoApi.Field(item, "name"), IsChecked = keys.Contains(key),
                    MinWidth = 112, Margin = new Thickness(0, 5, 4, 5) };
                boxes.Add((key, box));
                permissions.Children.Add(box);
            }
            if (ShowDialog(roleId > 0 ? "编辑角色" : "添加角色",
                [("角色名称", name), ("备注", remark), ("", active), ("", agent), ("菜单权限", permissions)], "保存",
                () => name.Text.Trim().Length > 0 ? "" : "请输入角色名称。"))
                await SaveAsync(session, "/manage/role/save", new
                {
                    role_id = roleId,
                    name = name.Text.Trim(),
                    remark = remark.Text.Trim(),
                    status = active.IsChecked == true ? 1 : 0,
                    agent_mode = agent.IsChecked == true ? 1 : 0,
                    permissions = boxes.Where(pair => pair.Box.IsChecked == true).Select(pair => pair.Key).ToArray()
                });
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (stamp == generation) Footer.Text = FailureText(exception);
        }
    }

    private async Task SaveAsync(AccountSession session, string path, object body)
    {
        if (mutating || session != account || session.Expired) return;
        var stamp = generation;
        mutating = true;
        requestSequence++;
        Footer.Text = "正在提交…";
        UpdateControls();
        try
        {
            await session.PostAsync(path, body);
        }
        catch (Exception exception) when (IsRequestFailure(exception))
        {
            if (exception is ApiException { SessionExpired: true }) session.Expire();
            if (stamp == generation) Footer.Text = FailureText(exception);
            return;
        }
        finally
        {
            mutating = false;
            UpdateControls();
            if (stamp != generation) await LoadAsync();
        }
        if (stamp == generation) await LoadAsync();
    }

    private static TextBox Input(string value, int limit, bool multiline = false) => new()
    {
        Text = value, MaxLength = limit, AcceptsReturn = multiline, MinHeight = multiline ? 110 : 32,
        TextWrapping = multiline ? TextWrapping.Wrap : TextWrapping.NoWrap,
        VerticalScrollBarVisibility = multiline ? ScrollBarVisibility.Auto : ScrollBarVisibility.Disabled
    };

    private bool ShowDialog(string title, (string Label, FrameworkElement Control)[] fields, string action,
        Func<string>? validate = null)
    {
        var body = new StackPanel { Margin = new Thickness(23) };
        body.Children.Add(new TextBlock { Text = title, FontSize = 21, FontWeight = FontWeights.Bold,
            TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 0, 0, 10) });
        foreach (var (label, control) in fields)
        {
            if (label.Length > 0) body.Children.Add(new TextBlock { Text = label, Foreground = (Brush)FindResource("Muted"), Margin = new Thickness(0, 9, 0, 5) });
            body.Children.Add(control);
        }
        var error = new TextBlock { Foreground = Brushes.Firebrick, TextWrapping = TextWrapping.Wrap };
        body.Children.Add(error);
        var buttons = new StackPanel { Orientation = Orientation.Horizontal, HorizontalAlignment = HorizontalAlignment.Right,
            Margin = new Thickness(0, 12, 0, 0) };
        var dialog = new Window
        {
            Owner = Window.GetWindow(this), Title = title, Width = 495, SizeToContent = SizeToContent.Height, MaxHeight = 600,
            Background = Brushes.White, Foreground = (Brush)FindResource("Ink"), FontSize = 13,
            ResizeMode = ResizeMode.NoResize, ShowInTaskbar = false, WindowStartupLocation = WindowStartupLocation.CenterOwner,
            Content = new ScrollViewer { Content = body, VerticalScrollBarVisibility = ScrollBarVisibility.Auto }
        };
        buttons.Children.Add(new Button { Content = "取消", IsCancel = true, MinWidth = 65 });
        var confirm = new Button { Content = action, IsDefault = true, MinWidth = 74, Margin = new Thickness(0),
            Style = (Style)FindResource("PrimaryButton") };
        confirm.Click += (_, _) =>
        {
            var text = validate?.Invoke() ?? "";
            if (text.Length == 0) dialog.DialogResult = true;
            else error.Text = text;
        };
        buttons.Children.Add(confirm);
        body.Children.Add(buttons);
        return dialog.ShowDialog() == true;
    }

    private static AdminColumn C(string label, string path, string format = "") => new(label, path, format);

    private static JsonElement Value(JsonElement row, string path)
    {
        foreach (var key in path.Split('.')) row = ImgoApi.Property(row, key);
        return row;
    }

    private static string Format(JsonElement row, AdminColumn column)
    {
        var value = Value(row, column.Path);
        var text = value.ValueKind == JsonValueKind.String ? value.GetString() ?? "" :
            value.ValueKind is JsonValueKind.Number or JsonValueKind.True or JsonValueKind.False ? value.GetRawText() : "";
        if (column.Format == "message")
            return ImgoApi.MessageText(row, column.Path);
        if (column.Format == "switch") return text is "1" or "true" ? "开启" : "关闭";
        if (column.Format == "review") return text switch { "0" => "待审核", "1" => "已通过", "2" => "已拒绝", _ => text };
        if (column.Format == "withdrawal") return text switch { "0" => "待处理", "1" => "已打款", "2" => "已拒绝", "3" => "已冻结", _ => text };
        if (column.Format == "result") return text switch { "1" => "成功", "0" => "失败", _ => text };
        if (!long.TryParse(text, out var number)) return text;
        return column.Format switch
        {
            "date" when number is > 0 and <= 253402300799999 =>
                DateTimeOffset.FromUnixTimeMilliseconds(number < 10_000_000_000 ? number * 1000 : number)
                    .ToLocalTime().ToString("yyyy-MM-dd HH:mm"),
            "date" => "—",
            "money" => $"¥{number / 100m:N2}",
            "bytes" => $"{number / 1024d:N1} KB",
            _ => text
        };
    }

    private void RenderCards(AdminPage definition, JsonElement result)
    {
        CardBody.Children.Clear();
        if (definition.Title != "系统概况")
        {
            var columns = definition.Title == "签到设置" ? new[] { C("签到奖励", "rewardEnabled", "switch") }
                .Concat(SignFields.Select(field => C(field.Label, field.Key))).ToArray() :
                new[] { C("系统名称", "name"), C("系统描述", "description"), C("客户端地址", "clientDownloadUrl") };
            foreach (var column in columns)
            {
                var value = Format(result, column);
                if (value.Length > 0) CardBody.Children.Add(CardSurface(Card(column.Label, value)));
            }
            CardBody.Children.Add(new TextBlock { Text = "完整配置可在网页中打开。", Margin = new Thickness(10) });
            return;
        }

        var totals = ImgoApi.Property(result, "totals");
        var today = ImgoApi.Property(result, "today");
        var summary = new UniformGrid { Columns = 4 };
        foreach (var (label, key, icon, hint) in new[]
        {
            ("用户总数", "users", "\uE77B", MetricHint(today, "users", "今日新增")),
            ("在线用户", "online_users", "\uE968", MetricHint(today, "peak_users", "今日峰值")),
            ("在线设备", "online_devices", "\uE7F4", MetricHint(today, "peak_devices", "今日峰值")),
            ("今日聊天用户", "today_online_users", "\uE8BD", ""),
            ("群聊总数", "groups", "\uE902", MetricHint(today, "groups", "今日创建")),
            ("消息总数", "messages", "\uE8F2", MetricHint(today, "messages", "今日发送")),
            ("文件总数", "files", "\uE8B7", MetricHint(today, "files", "今日上传"))
        })
        {
            var content = Card(label, ImgoApi.Field(totals, key) is { Length: > 0 } value ? value : "—");
            var heading = (TextBlock)content.Children[0];
            content.Children.RemoveAt(0);
            var header = new DockPanel { Margin = new Thickness(0, 0, 0, 6), LastChildFill = true };
            var symbol = new TextBlock { Text = icon, FontFamily = new FontFamily("Segoe MDL2 Assets"),
                Foreground = (Brush)FindResource("Accent"), FontSize = 14, HorizontalAlignment = HorizontalAlignment.Right };
            DockPanel.SetDock(symbol, Dock.Right);
            header.Children.Add(symbol);
            header.Children.Add(heading);
            content.Children.Insert(0, header);
            if (hint.Length > 0) content.Children.Add(ChartLabel(hint));
            summary.Children.Add(CardSurface(content));
        }
        CardBody.Children.Add(summary);

        var charts = new UniformGrid { Columns = 2, Margin = new Thickness(0, 3, 0, 0) };
        charts.Children.Add(BuildChart("用户注册趋势", ImgoApi.Property(result, "registration_month"), "value"));
        charts.Children.Add(BuildChart("在线趋势", ImgoApi.Property(result, "online"), "users", "devices",
            ImgoApi.Number(result, "online_bucket_seconds")));
        charts.Children.Add(BuildChart("消息发送趋势", ImgoApi.Property(result, "messages"), "value"));
        charts.Children.Add(BuildChart("群聊增长趋势", ImgoApi.Property(result, "groups"), "value"));
        charts.Children.Add(BuildChart("文件增长趋势", ImgoApi.Property(result, "files"), "value"));
        CardBody.Children.Add(charts);
    }

    private static string MetricHint(JsonElement data, string field, string label)
    {
        var value = ImgoApi.Field(data, field);
        return value.Length == 0 ? "" : $"{label} {value}";
    }

    private Border BuildChart(string title, JsonElement source, string first, string? second = null, long bucketSeconds = 0)
    {
        var card = Card(title, "");
        var surface = CardSurface(card);
        if (source.ValueKind != JsonValueKind.Array || source.GetArrayLength() == 0)
        {
            card.Children.Add(new TextBlock { Text = "暂无数据", Foreground = (Brush)FindResource("Muted"),
                VerticalAlignment = VerticalAlignment.Center, HorizontalAlignment = HorizontalAlignment.Center,
                Margin = new Thickness(0, 54, 0, 54) });
            return surface;
        }
        var items = source.EnumerateArray().ToArray();
        var max = Math.Max(1, items.Max(item => Math.Max(ImgoApi.Number(item, first),
            second is null ? 0 : ImgoApi.Number(item, second))));
        var canvas = new Canvas { Width = 400, Height = 117, Margin = new Thickness(0, 4, 0, 0) };
        foreach (var y in new[] { 10, 36, 62, 88, 114 })
            canvas.Children.Add(new Line { X1 = 24, X2 = 398, Y1 = y, Y2 = y,
                Stroke = (Brush)FindResource("Line"), StrokeThickness = 1 });
        var maximum = ChartLabel(max.ToString());
        Canvas.SetTop(maximum, 1);
        canvas.Children.Add(maximum);
        var zero = ChartLabel("0");
        Canvas.SetTop(zero, 100);
        canvas.Children.Add(zero);

        Draw(first, (Brush)FindResource("Accent"));
        if (second is not null) Draw(second, Brushes.SeaGreen);
        card.Children.Add(new Viewbox { Stretch = Stretch.Uniform, MaxHeight = 140, Child = canvas });
        var start = SampleLabel(items[0]);
        var end = SampleLabel(items[^1]);
        var labels = new DockPanel { Margin = new Thickness(17, 2, 0, 0) };
        var endLabel = ChartLabel(end);
        DockPanel.SetDock(endLabel, Dock.Right);
        labels.Children.Add(endLabel);
        if (items.Length > 1) labels.Children.Add(ChartLabel(start));
        card.Children.Add(labels);
        if (second is not null) card.Children.Add(ChartLabel("蓝色 在线用户   ·   绿色 在线设备"));
        return surface;

        static string SampleLabel(JsonElement item)
        {
            var label = ImgoApi.Field(item, "label");
            var timestamp = ImgoApi.Number(item, "time");
            return label.Length > 0 || timestamp is <= 0 or > 253402300799
                ? label : DateTimeOffset.FromUnixTimeSeconds(timestamp).ToLocalTime().ToString("MM-dd HH:mm");
        }

        void Draw(string field, Brush color)
        {
            var startTime = ImgoApi.Number(items[0], "time");
            var endTime = ImgoApi.Number(items[^1], "time");
            var previousTime = startTime;
            var points = new PointCollection();
            for (var i = 0; i < items.Length; i++)
            {
                var time = ImgoApi.Number(items[i], "time");
                if (bucketSeconds > 0 && i > 0 && time - previousTime > bucketSeconds)
                {
                    AddSeries(points);
                    points = new PointCollection();
                }
                double fraction = endTime > startTime ? (time - startTime) / (double)(endTime - startTime)
                    : i / (double)Math.Max(1, items.Length - 1);
                points.Add(new Point(24 + 374 * fraction, 114 - 104d * ImgoApi.Number(items[i], field) / max));
                previousTime = time;
            }
            AddSeries(points);

            void AddSeries(PointCollection segment)
            {
                if (segment.Count > 1)
                    canvas.Children.Add(new Polyline { Stroke = color, StrokeThickness = 2.4,
                        StrokeLineJoin = PenLineJoin.Round, Points = segment });
                else if (segment.Count == 1)
                {
                    var dot = new Ellipse { Width = 6, Height = 6, Fill = color };
                    Canvas.SetLeft(dot, segment[0].X - 3);
                    Canvas.SetTop(dot, segment[0].Y - 3);
                    canvas.Children.Add(dot);
                }
            }
        }
    }

    private TextBlock ChartLabel(string text) => new() { Text = text, Foreground = (Brush)FindResource("Muted"), FontSize = 11 };

    private Border CardSurface(StackPanel content) => new()
    {
        Child = content, Margin = new Thickness(5), Padding = new Thickness(12, 10, 12, 10), Background = Brushes.White,
        BorderBrush = (Brush)FindResource("Line"), BorderThickness = new Thickness(1), CornerRadius = new CornerRadius(9)
    };

    private StackPanel Card(string title, string value)
    {
        var card = new StackPanel();
        card.Children.Add(new TextBlock { Text = title, Foreground = (Brush)FindResource("Muted"),
            FontWeight = FontWeights.SemiBold, TextWrapping = TextWrapping.Wrap });
        if (value.Length > 0)
            card.Children.Add(new TextBlock { Text = value, FontSize = 24, FontWeight = FontWeights.SemiBold,
                Margin = new Thickness(0, 1, 0, 3), TextWrapping = TextWrapping.Wrap });
        return card;
    }

    internal sealed record AdminRow(JsonElement Data, string[] Values);

    private static bool IsRequestFailure(Exception error) =>
        error is ApiException or HttpRequestException or OperationCanceledException or JsonException or InvalidOperationException;

    private static string FailureText(Exception error) => error switch
    {
        HttpRequestException => "无法连接 Go API。",
        OperationCanceledException => "请求超时或已取消。",
        _ => error.Message
    };
}
