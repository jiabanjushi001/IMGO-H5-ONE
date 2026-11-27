using Microsoft.Win32;
using System.IO;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;
using System.Windows.Media.Imaging;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private async void OpenGroupSettings(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is { } session && visibleConversation is { Group: 1 } conversation)
            await EditGroupAsync(session, conversation);
    }

    private async void OpenGroupNotice(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is not { } session || visibleConversation is not { Group: 1 } group) return;
        var page = OpenSidePage("群公告");
        var info = await LoadGroupSideInfoAsync(page, session, group);
        if (!IsSidePageOpen(page) || info.ValueKind != JsonValueKind.Object) return;
        page.Body.Children.Add(ContactHeading(group));
        var notice = ImgoApi.Field(info, "notice");
        page.Body.Children.Add(new TextBlock { Text = string.IsNullOrWhiteSpace(notice) ? "暂无群公告" : notice,
            TextWrapping = TextWrapping.Wrap });
        if (ImgoApi.Number(info, "isJoin") is 1 or 2)
        {
            var edit = AddDialogButton(page.Footer, "编辑公告", () => { });
            edit.Click += async (_, _) => await EditGroupAsync(session, group, focusNotice: true, push: true);
        }
    }

    private async Task<JsonElement> LoadGroupSideInfoAsync(SidePage page, AccountSession session, Conversation group)
    {
        var status = new TextBlock { Text = "正在读取群资料…", TextWrapping = TextWrapping.Wrap };
        page.Body.Children.Add(status);
        try
        {
            var info = await session.PostAsync("/enterprise/group/groupinfo", new { group_id = group.Id });
            page.Body.Children.Remove(status);
            return ActiveChatAccount == session ? info : default;
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            status.Text = ErrorText(error);
            return default;
        }
    }

    private async Task EditGroupAsync(AccountSession session, Conversation conversation, bool focusNotice = false, bool push = false)
    {
        var page = OpenSidePage("群设置", push);
        var info = await LoadGroupSideInfoAsync(page, session, conversation);
        if (!IsSidePageOpen(page) || info.ValueKind != JsonValueKind.Object) return;
        var role = ImgoApi.Number(info, "isJoin");
        bool owner = role == 1;
        bool canEdit = role is 1 or 2;
        bool canEditAvatar = ImgoApi.Property(info, "canEditAvatar").ValueKind == JsonValueKind.True;
        bool canSetNumberJoin = session.IsSuperAdmin || owner && session.IsAgent && session.HasPermission("manage.groups");
        var current = ImgoApi.Property(info, "setting");
        var panel = page.Body;
        var footer = page.Footer;

        var groupTitle = ImgoApi.Field(info, "name");
        if (groupTitle.Length == 0) groupTitle = conversation.Name;
        var header = new DockPanel { Margin = new Thickness(0, 0, 0, 12) };
        var avatar = new Border { Width = 48, Height = 48, CornerRadius = new CornerRadius(8),
            Background = (Brush)FindResource("ChatTint"), Margin = new Thickness(0, 0, 12, 0), ClipToBounds = true };
        var portrait = new Grid { ClipToBounds = true };
        portrait.Children.Add(new TextBlock { Text = conversation.Initial, FontWeight = FontWeights.SemiBold,
            FontSize = 18, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center });
        var image = new Image { Stretch = Stretch.UniformToFill, DataContext = conversation with
            { Avatar = ImgoApi.AvatarUri(session.Server, ImgoApi.Field(info, "avatar")) ?? conversation.Avatar } };
        image.Loaded += LoadThumbnail;
        portrait.Children.Add(image);
        avatar.Child = portrait;
        DockPanel.SetDock(avatar, Dock.Left);
        header.Children.Add(avatar);
        var title = new StackPanel { VerticalAlignment = VerticalAlignment.Center };
        title.Children.Add(new TextBlock { Text = groupTitle, FontSize = 20, FontWeight = FontWeights.SemiBold, TextWrapping = TextWrapping.Wrap });
        string groupNumber = ImgoApi.Field(info, "group_id");
        var count = ImgoApi.Field(info, "groupUserCount");
        var ownerName = ImgoApi.Field(info, "ownerName");
        title.Children.Add(new TextBlock { Text = $"群号 {groupNumber}　·　群主 {ownerName}" +
            (count.Length > 0 ? $"　·　{count} 人" : ""), Foreground = (Brush)FindResource("ChatMuted"),
            TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 4, 0, 0) });
        header.Children.Add(title);
        panel.Children.Add(header);

        var common = new WrapPanel { Margin = new Thickness(0, 0, 0, 13) };
        var localStatus = new TextBlock { MinHeight = 18, Margin = new Thickness(0, 9, 0, 0), TextWrapping = TextWrapping.Wrap,
            Foreground = (Brush)FindResource("ChatAccent") };
        panel.Children.Add(common);
        AddDialogButton(common, "复制群号", () => CopyGroupText(groupNumber, localStatus, "群号已复制"));
        var members = AddDialogButton(common, "群成员", () => { });
        members.Click += async (_, _) => await ShowGroupMembersAsync(session, conversation, info, push: true);
        var invite = AddDialogButton(common, "群二维码", () => { });
        invite.Click += async (_, _) =>
        {
            invite.IsEnabled = false;
            try { await ShowGroupQrAsync(session, conversation); }
            catch (Exception error) when (IsAttachmentFailure(error) || error is FormatException or NotSupportedException)
            {
                if (error is ApiException { SessionExpired: true }) session.Expire();
                localStatus.Text = ErrorText(error);
            }
            finally { invite.IsEnabled = true; }
        };
        if (owner || role == 2 && ImgoApi.Number(current, "manager_invite") == 1)
        {
            var add = AddDialogButton(common, "邀请成员", () => { });
            add.Click += async (_, _) => await ShowGroupInviteAsync(session, conversation, push: true, ownerStatus: localStatus);
        }

        string? avatarPath = null;
        if (canEditAvatar)
        {
            var avatarButton = AddDialogButton(common, "更换头像", () => { });
            avatarButton.Click += (_, _) =>
            {
                var picker = new OpenFileDialog { Title = "选择群头像", Filter = ImageFileFilter };
                if (picker.ShowDialog(this) != true) return;
                try
                {
                    if (new FileInfo(picker.FileName).Length > 5 * 1024 * 1024)
                    {
                        localStatus.Text = "群头像不能超过 5 MB";
                        return;
                    }
                    avatarPath = picker.FileName;
                    localStatus.Text = $"已选择 {Path.GetFileName(avatarPath)}，保存后生效";
                }
                catch (Exception error) when (IsAttachmentFailure(error)) { localStatus.Text = ErrorText(error); }
            };
        }

        var name = new TextBox { Text = groupTitle, MaxLength = 64, IsReadOnly = !canEdit };
        var notice = new TextBox { Text = ImgoApi.Field(info, "notice"), MaxLength = 4000, IsReadOnly = !canEdit,
            AcceptsReturn = true, TextWrapping = TextWrapping.Wrap, Height = 64, VerticalScrollBarVisibility = ScrollBarVisibility.Auto };
        var mute = new ComboBox { ItemsSource = new[] { "全员可发言", "仅群主和管理员可发言", "仅群主可发言" },
            SelectedIndex = (int)Math.Clamp(ImgoApi.Number(current, "nospeak"), 0, 2), IsEnabled = canEdit };
        AddField("群名称", name);
        AddField("群公告", notice);
        AddField("全群禁言", mute);
        var flags = new Dictionary<string, CheckBox>(StringComparer.Ordinal);
        AddFlag("manage", "仅群主和管理员可管理", owner);
        AddFlag("invite", "允许邀请", owner, 1);
        AddFlag("manager_invite", "允许群管理员邀请成员", owner);
        AddFlag("profile", "允许查看群成员资料并添加为好友", owner);
        AddFlag("history", "新成员可查看历史消息", owner, 1);
        var numberJoin = AddFlag("number_join", "输入群号即可加入私有群", canSetNumberJoin);

        if (owner || role > 0 && session.IsSuperAdmin)
        {
            var actions = new WrapPanel { Margin = new Thickness(0, 12, 0, 0) };
            panel.Children.Add(actions);
            if (owner)
            {
                var transfer = AddDialogButton(actions, "转让群主…", () => { });
                transfer.Click += async (_, _) => await ShowGroupMembersAsync(session, conversation, info, push: true, transferOnly: true);
                var dissolve = AddDialogButton(actions, "解散群聊…", () => { });
                dissolve.Click += async (_, _) => await RunChatActionAsync(() => ApplyGroupDestructionAsync(session, conversation, removeGroup: true));
            }
            var clear = AddDialogButton(actions, "清空全群聊天记录…", () => { });
            clear.Click += async (_, _) => await RunChatActionAsync(() => ApplyGroupDestructionAsync(session, conversation, removeGroup: false));
        }
        AddConversationSideActions(panel, session, conversation);
        DockPanel.SetDock(localStatus, Dock.Bottom);
        page.Root.Children.Insert(1, localStatus);
        var savedSettings = new Dictionary<string, int>(StringComparer.Ordinal);
        if (canEdit) savedSettings["nospeak"] = OriginalFlag("nospeak");
        foreach (var key in flags.Keys) savedSettings[key] = OriginalFlag(key);
        var savedName = groupTitle;
        var savedNotice = ImgoApi.Field(info, "notice");

        if (canEdit || canEditAvatar || canSetNumberJoin)
        {
            var save = AddDialogButton(footer, "保存", () => { });
            save.Style = (Style)FindResource("ChatPrimaryAction");
            save.Click += async (_, _) =>
            {
                var newName = name.Text.Trim();
                if (canEdit && newName.Length == 0) { localStatus.Text = "请输入群名称"; name.Focus(); return; }
                panel.IsEnabled = footer.IsEnabled = false;
                localStatus.Text = "保存中…";
                bool succeeded = false;
                var groupId = conversation.Id;
                try
                {
                    if (canEdit)
                    {
                        var changes = new Dictionary<string, int>();
                        if (mute.SelectedIndex != savedSettings["nospeak"]) changes["nospeak"] = mute.SelectedIndex;
                        if (owner)
                        {
                            foreach (var (key, check) in flags)
                            {
                                int value = check.IsChecked == true ? 1 : 0;
                                if (key != "number_join" && value != savedSettings[key]) changes[key] = value;
                            }
                        }
                        if (changes.Count > 0)
                        {
                            await session.PostAsync("/enterprise/group/groupsetting", new { group_id = groupId, setting = changes });
                            foreach (var (key, value) in changes) savedSettings[key] = value;
                        }
                        if (newName != savedName)
                        {
                            await session.PostAsync("/enterprise/group/editgroupname", new { group_id = groupId, displayName = newName });
                            savedName = newName;
                        }
                        if (notice.Text != savedNotice)
                        {
                            await session.PostAsync("/enterprise/group/setnotice", new { group_id = groupId, notice = notice.Text });
                            savedNotice = notice.Text;
                        }
                    }
                    int join = numberJoin.IsChecked == true ? 1 : 0;
                    if (canSetNumberJoin && join != savedSettings["number_join"])
                    {
                        await session.PostAsync("/manage/group/setnumberjoin", new { group_id = groupId, enabled = join });
                        savedSettings["number_join"] = join;
                    }
                    if (avatarPath is not null)
                    {
                        var uploaded = await session.UploadAsync(avatarPath, "uploadfile");
                        var fileId = ImgoApi.Number(uploaded, "file_id");
                        if (fileId <= 0) throw new ApiException("服务器未返回头像文件编号", 0);
                        var updated = await session.PostAsync("/enterprise/group/editgroupavatar", new { group_id = groupId, file_id = fileId });
                        RefreshGroupAvatar(session, groupId, ImgoApi.Field(updated, "avatar"));
                        avatarPath = null;
                    }
                    succeeded = true;
                }
                catch (Exception error) when (IsAttachmentFailure(error))
                {
                    if (error is ApiException { SessionExpired: true }) session.Expire();
                    localStatus.Text = $"保存失败。{ErrorText(error)}";
                }
                finally
                {
                    panel.IsEnabled = footer.IsEnabled = true;
                    session.Invalidate();
                    ScheduleRefreshes();
                }
                if (succeeded)
                {
                    localStatus.Text = "群设置已保存";
                    RefreshAfterAction(session, "群设置已保存");
                    if (push && IsSidePageOpen(page) && sideHistory.TryPeek(out var previous) && previous.Title == "群公告")
                    {
                        var announcement = previous.Body.Children.OfType<TextBlock>().LastOrDefault();
                        if (announcement is not null) announcement.Text = string.IsNullOrWhiteSpace(savedNotice) ? "暂无群公告" : savedNotice;
                    }
                }
            };
        }

        if (focusNotice) { notice.Focus(); notice.BringIntoView(); }

        int OriginalFlag(string key) => (int)ImgoApi.Number(current, key, key is "invite" or "history" ? 1 : 0);
        void AddField(string label, Control control)
        {
            panel.Children.Add(new TextBlock { Text = label, Margin = new Thickness(0, 8, 0, 4) });
            panel.Children.Add(control);
        }
        CheckBox AddFlag(string key, string label, bool enabled, int fallback = 0)
        {
            var check = new CheckBox { Content = new TextBlock { Text = label, TextWrapping = TextWrapping.Wrap },
                IsChecked = ImgoApi.Number(current, key, fallback) == 1,
                IsEnabled = enabled, Margin = new Thickness(0, 10, 0, 0) };
            flags.Add(key, check);
            panel.Children.Add(check);
            return check;
        }
    }

    private void RefreshGroupAvatar(AccountSession session, string groupId, string path)
    {
        var updated = ImgoApi.AvatarUri(session.Server, path);
        if (updated is null) return;
        var existing = session.Conversations.FindIndex(entry => entry.Id == groupId);
        if (existing < 0) return;
        session.Conversations[existing] = session.Conversations[existing] with { Avatar = updated };
        if (visibleAccount == session) ApplyContacts(session);
    }

    private async Task ApplyGroupDestructionAsync(AccountSession session, Conversation conversation, bool removeGroup)
    {
        var label = removeGroup ? "解散群聊" : "清空全群聊天记录";
        string question = removeGroup ? $"解散 {conversation.Name}？所有成员都会退出该群。"
            : $"清空 {conversation.Name} 的全部聊天记录？此操作对所有成员生效。";
        if (MessageBox.Show(this, question, label, MessageBoxButton.YesNo, MessageBoxImage.Warning,
            MessageBoxResult.No) != MessageBoxResult.Yes) return;

        await session.PostAsync($"/enterprise/group/{(removeGroup ? "removegroup" : "clearmessage")}", new { group_id = conversation.Id });
        session.InvalidateHistory();
        if (removeGroup)
        {
            session.Conversations.RemoveAll(group => group.Id == conversation.Id);
            if (session.SelectedConversation == conversation.Id) session.SelectedConversation = "";
            if (visibleAccount == session) ApplyContacts(session);
        }
        RefreshAfterAction(session, removeGroup ? "群聊已解散" : "已清空全群聊天记录");
    }

    private async Task ShowGroupQrAsync(AccountSession session, Conversation conversation)
    {
        var page = OpenSidePage("群二维码", push: true);
        var panel = page.Body;
        var buttons = page.Footer;
        var loading = new TextBlock { Text = "正在读取二维码…", TextWrapping = TextWrapping.Wrap };
        panel.Children.Add(loading);
        JsonElement info;
        try
        {
            info = await session.PostAsync("/enterprise/group/groupinfo", new { group_id = conversation.Id, include_qr_image = 1 });
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            loading.Text = ErrorText(error);
            return;
        }
        if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
        panel.Children.Clear();
        var image = ImgoApi.Field(info, "qrImage");
        var link = ImgoApi.Field(info, "qrUrl");
        const string prefix = "data:image/png;base64,";
        if (!image.StartsWith(prefix, StringComparison.Ordinal))
        {
            panel.Children.Add(new TextBlock { Text = ImgoApi.Property(info, "canShareQr").ValueKind == JsonValueKind.False ?
                "当前群二维码入口已关闭" : "当前 Go 服务尚未返回群二维码图片",
                TextWrapping = TextWrapping.Wrap });
        }
        else
        {
            using var bytes = new MemoryStream(Convert.FromBase64String(image[prefix.Length..]));
            var decoded = BitmapFrame.Create(bytes, BitmapCreateOptions.None, BitmapCacheOption.OnLoad);
            var code = new Image { Source = decoded, Width = 220, Height = 220, HorizontalAlignment = HorizontalAlignment.Center };
            RenderOptions.SetBitmapScalingMode(code, BitmapScalingMode.NearestNeighbor);
            panel.Children.Add(code);
            var expiry = ImgoApi.Field(info, "qrExpire");
            if (expiry.Length > 0) AddDetail(panel, "有效期", $"{expiry}前");
            var status = new TextBlock { Foreground = (Brush)FindResource("ChatAccent") };
            panel.Children.Add(status);
            if (link.Length > 0) AddDialogButton(buttons, "复制邀请链接", () => CopyGroupText(link, status, "邀请链接已复制"));
        }
    }

    private static void CopyGroupText(string value, TextBlock status, string success)
    {
        if (value.Length == 0) return;
        try { Clipboard.SetText(value); status.Text = success; }
        catch (System.Runtime.InteropServices.ExternalException) { status.Text = "剪贴板暂时无法使用"; }
    }

    private async Task ViewGroupMemberAsync(AccountSession session, Conversation group, ChatChoice member)
    {
        var page = OpenSidePage("群成员资料", push: true);
        var status = new TextBlock { Text = "正在读取资料…", TextWrapping = TextWrapping.Wrap };
        page.Body.Children.Add(status);
        try
        {
            var profile = await session.PostAsync("/enterprise/im/getuserinfo", new { user_id = member.Id, group_id = group.Id });
            if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
            status.Text = "";
            AddDetail(page.Body, "昵称", ImgoApi.Field(profile, "realname"));
            AddDetail(page.Body, "签名", ImgoApi.Field(profile, "motto"));
            if (ImgoApi.Number(ImgoApi.Property(profile, "friendInfo"), "status") == 1) return;

            var note = new TextBox { MaxLength = 100 };
            page.Body.Children.Add(new TextBlock { Text = "验证消息（选填）", Margin = new Thickness(0, 7, 0, 4) });
            page.Body.Children.Add(note);
            var add = AddDialogButton(page.Footer, "添加好友", () => { });
            add.Style = (Style)FindResource("ChatPrimaryAction");
            add.Click += async (_, _) =>
            {
                add.IsEnabled = false;
                try
                {
                    await session.PostAsync("/enterprise/friend/add", new { user_id = member.Id, group_id = group.Id, remark = note.Text.Trim() });
                    status.Text = "好友申请已发送";
                }
                catch (Exception error) when (IsRequestFailure(error))
                {
                    if (error is ApiException { SessionExpired: true }) session.Expire();
                    status.Text = ErrorText(error);
                    add.IsEnabled = true;
                }
            };
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            status.Text = ErrorText(error);
        }
    }
}
