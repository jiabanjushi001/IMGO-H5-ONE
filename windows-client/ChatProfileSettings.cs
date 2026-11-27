using System.Globalization;
using System.Security;
using System.Security.Cryptography;
using System.IO;
using System.Text;
using System.Text.Json;
using System.Windows;
using System.Windows.Automation;
using System.Windows.Controls;
using System.Windows.Media;
using Microsoft.Win32;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private async void OpenChatSettings(object sender, RoutedEventArgs e)
    {
        if (ActiveChatAccount is not { } session) return;
        var page = OpenSidePage("个人资料与聊天设置");
        var status = new TextBlock { Text = "正在读取个人资料…", TextWrapping = TextWrapping.Wrap };
        page.Body.Children.Add(status);
        try
        {
            var profile = await session.PostAsync("/enterprise/im/getuserinfo", new { user_id = session.UserId });
            if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
            PopulateChatSettings(page, session, profile, status);
        }
        catch (Exception error) when (IsRequestFailure(error))
        {
            SetContactActionError(session, status, error);
        }
    }

    private void PopulateChatSettings(SidePage page, AccountSession session, JsonElement profile, TextBlock status)
    {
        page.Body.Children.Clear();
        status.Text = "";
        status.Margin = new Thickness(0, 12, 0, 0);
        status.Foreground = (Brush)FindResource("ChatMuted");

        var heading = new DockPanel { Margin = new Thickness(0, 2, 0, 14) };
        var fallback = new TextBlock
        {
            HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center,
            FontSize = 19, FontWeight = FontWeights.SemiBold, Foreground = (Brush)FindResource("ChatAccent")
        };
        var image = new Image { Stretch = Stretch.UniformToFill, OpacityMask = (Brush)FindResource("ChatAvatarMask") };
        image.Loaded += LoadThumbnail;
        image.DataContextChanged += ThumbnailDataChanged;
        var layers = new Grid();
        layers.Children.Add(fallback);
        layers.Children.Add(image);
        var avatar = new Border
        {
            Width = 52, Height = 52, CornerRadius = new CornerRadius(13), Child = layers,
            Background = (Brush)FindResource("ChatTint"), Margin = new Thickness(0, 0, 10, 0)
        };
        DockPanel.SetDock(avatar, Dock.Left);
        heading.Children.Add(avatar);
        var accountLabel = new TextBlock
        {
            Text = session.Account, VerticalAlignment = VerticalAlignment.Center, FontWeight = FontWeights.SemiBold,
            TextTrimming = TextTrimming.CharacterEllipsis, ToolTip = session.Account
        };
        heading.Children.Add(accountLabel);
        page.Body.Children.Add(heading);

        var portrait = ImgoApi.Field(profile, "avatar");
        var name = new TextBox { Text = ImgoApi.Field(profile, "realname"), MaxLength = 100 };
        SetAvatar(portrait);
        var avatarButton = new Button { Content = "更换头像", HorizontalAlignment = HorizontalAlignment.Left };
        page.Body.Children.Add(avatarButton);
        avatarButton.Click += async (_, _) =>
        {
            if (ActiveChatAccount != session) return;
            var picker = new OpenFileDialog { Title = "选择个人头像", Filter = ImageFileFilter };
            if (picker.ShowDialog(this) != true) return;
            avatarButton.IsEnabled = false;
            try
            {
                var size = new FileInfo(picker.FileName).Length;
                if (size == 0 || size > 5 * 1024 * 1024)
                {
                    status.Text = "请选择不超过 5 MB 的非空图片。";
                    return;
                }

                status.Text = "正在上传头像…";
                var result = await session.UploadAsync(picker.FileName, "uploadavatar");
                if (result.ValueKind == JsonValueKind.String && result.GetString() is { Length: > 0 } url)
                {
                    portrait = url;
                    SetAvatar(portrait);
                }
                RefreshAfterAction(session, "头像已更新");
                status.Text = "头像已更新";
            }
            catch (Exception error) when (IsAttachmentFailure(error)) { SetContactActionError(session, status, error); }
            finally { avatarButton.IsEnabled = true; }
        };

        var motto = new TextBox { Text = ImgoApi.Field(profile, "motto"), MaxLength = 1000, TextWrapping = TextWrapping.Wrap };
        var email = new TextBox { Text = session.Email, MaxLength = 254 };
        AddSettingsField(page.Body, "昵称", name);
        AddSettingsField(page.Body, "签名", motto);
        AddSettingsField(page.Body, "邮箱", email);
        var fields = new[] { ("realname", name), ("motto", motto), ("email", email) };
        var saved = fields.ToDictionary(field => field.Item1, field => field.Item2.Text);

        var shortcut = new CheckBox
        {
            Content = new TextBlock { Text = "按 Enter 发送，Ctrl + Enter 换行", TextWrapping = TextWrapping.Wrap },
            IsChecked = session.EnterToSend, Margin = new Thickness(0, 15, 0, 0),
            ToolTip = "按当前服务器和账号保存在这台电脑上"
        };
        page.Body.Children.Add(shortcut);
        page.Body.Children.Add(status);
        var save = new Button { Content = "保存设置", Style = (Style)FindResource("ChatPrimaryAction"), MinWidth = 82 };
        page.Footer.Children.Add(save);
        save.Click += async (_, _) =>
        {
            if (ActiveChatAccount != session) return;
            var changes = new Dictionary<string, string>();
            foreach (var (field, input) in fields)
                if (input.Text.Trim() != saved[field])
                    changes[field] = input.Text.Trim();

            var enterToSend = shortcut.IsChecked == true;
            save.IsEnabled = false;
            try
            {
                if (changes.Count > 0)
                {
                    await session.PostAsync("/enterprise/im/updateuserinfo", changes);
                    foreach (var (field, value) in changes)
                        saved[field] = value;
                }
                if (changes.TryGetValue("email", out var emailValue)) session.Email = emailValue;
                var shortcutSaved = session.EnterToSend == enterToSend || SaveSendShortcut(session, enterToSend);
                session.EnterToSend = enterToSend;
                if (changes.TryGetValue("realname", out var nameValue))
                {
                    session.UpdateName(nameValue);
                    SetAvatar(portrait);
                }
                RefreshAfterAction(session, "聊天设置已保存");
                status.Text = shortcutSaved ? "已保存" : "快捷键暂时只在本次登录生效";
                if (visibleAccount == session) UpdateButtons();
            }
            catch (Exception error) when (IsRequestFailure(error)) { SetContactActionError(session, status, error); }
            finally { save.IsEnabled = true; }
        };
        AddDialogButton(page.Footer, "修改密码", () => ShowPasswordSettings(session));

        void SetAvatar(string value)
        {
            var label = name.Text.Trim();
            if (label.Length == 0) label = session.Account;
            fallback.Text = StringInfo.GetNextTextElement(label);
            image.DataContext = new Conversation(session.UserId.ToString(), label, 0, 0, "",
                Avatar: ImgoApi.AvatarUri(session.Server, value));
        }
    }

    private const string SendShortcutKey = @"HKEY_CURRENT_USER\Software\Imgo\SendShortcuts";

    private static string SendShortcutAccount(AccountSession session) =>
        Convert.ToHexString(SHA256.HashData(Encoding.UTF8.GetBytes($"{session.Server.AbsoluteUri}|{session.UserId}")));

    private static bool ReadSendShortcut(AccountSession session, JsonElement profile)
    {
        try
        {
            if (Registry.GetValue(SendShortcutKey, SendShortcutAccount(session), null) is int saved)
                return saved == 1;
        }
        catch (Exception error) when (error is IOException or UnauthorizedAccessException or SecurityException) { }
        return ImgoApi.Number(ImgoApi.Property(profile, "setting"), "sendKey") == 1;
    }

    private static bool SaveSendShortcut(AccountSession session, bool enterToSend)
    {
        try
        {
            Registry.SetValue(SendShortcutKey, SendShortcutAccount(session), enterToSend ? 1 : 0, RegistryValueKind.DWord);
            return true;
        }
        catch (Exception error) when (error is IOException or UnauthorizedAccessException or SecurityException)
        {
            return false;
        }
    }

    private void ShowPasswordSettings(AccountSession session)
    {
        if (ActiveChatAccount != session) return;
        var page = OpenSidePage("修改密码", push: true);
        var current = new PasswordBox();
        var password = new PasswordBox { MaxLength = 30 };
        var confirmation = new PasswordBox { MaxLength = 30 };
        AddSettingsField(page.Body, "原密码", current);
        AddSettingsField(page.Body, "新密码", password);
        AddSettingsField(page.Body, "确认新密码", confirmation);
        var status = new TextBlock { Text = "保存后需要重新登录。", TextWrapping = TextWrapping.Wrap, Margin = new Thickness(0, 12, 0, 0) };
        page.Body.Children.Add(status);

        var save = new Button { Content = "保存密码", Style = (Style)FindResource("ChatPrimaryAction"), MinWidth = 82 };
        page.Footer.Children.Add(save);
        save.Click += async (_, _) =>
        {
            if (ActiveChatAccount != session) return;
            if (string.IsNullOrEmpty(current.Password))
            {
                status.Text = "请输入原密码。";
                current.Focus();
                return;
            }
            if (Encoding.UTF8.GetByteCount(password.Password) is < 6 or > 30)
            {
                status.Text = "新密码长度须为 6–30 字节。";
                password.Focus();
                return;
            }
            if (!string.Equals(password.Password, confirmation.Password, StringComparison.Ordinal))
            {
                status.Text = "两次输入的新密码不一致。";
                confirmation.Focus();
                return;
            }

            save.IsEnabled = false;
            try
            {
                await session.PostAsync("/enterprise/im/editpassword",
                    new { originalPassword = current.Password, password = password.Password });
                current.Clear();
                password.Clear();
                confirmation.Clear();
                session.Expire();
                if (visibleAccount != session) return;
                CloseConversationPane();
                ServerAddress.Text = session.Server.ToString();
                LoginAccount.Text = session.Account;
                AddAccount(this, new RoutedEventArgs());
                LoginPassword.Focus();
                Status.Text = "密码已修改，请重新登录";
                UpdateButtons();
            }
            catch (Exception error) when (IsRequestFailure(error)) { SetContactActionError(session, status, error); }
            finally { save.IsEnabled = true; }
        };
        AddDialogButton(page.Footer, "返回", BackConversationPane);
        current.Focus();
    }

    private void AddSettingsField(Panel panel, string label, Control input)
    {
        AutomationProperties.SetName(input, label);
        if (input is TextBox) input.Style = (Style)FindResource("ChatInput");
        panel.Children.Add(new TextBlock
        {
            Text = label, Foreground = (Brush)FindResource("ChatMuted"), Margin = new Thickness(0, 10, 0, 4)
        });
        panel.Children.Add(input);
    }
}
