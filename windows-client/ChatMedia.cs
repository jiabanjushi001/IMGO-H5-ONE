using System.Runtime.CompilerServices;
using System.Diagnostics;
using System.IO;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using System.Windows.Threading;
using Microsoft.Win32;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private const string ImageFileFilter = "图片|*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp;*.ico";
    private bool CanSendAttachment => Compose.IsEnabled && !transferringAttachment;

    private readonly SemaphoreSlim thumbnailSlots = new(3, 3);
    private readonly ConditionalWeakTable<AccountSession, Dictionary<string, BitmapSource>> avatarCache = new();

    private async void LoadThumbnail(object sender, RoutedEventArgs e)
    {
        if (sender is Image image) await UpdateThumbnailAsync(image);
    }

    private async void ThumbnailDataChanged(object sender, DependencyPropertyChangedEventArgs e)
    {
        if (sender is Image { IsLoaded: true } image) await UpdateThumbnailAsync(image);
    }

    private async Task UpdateThumbnailAsync(Image image)
    {
        image.Source = null;
        var data = image.DataContext;
        bool avatar = image.Tag as string == "sender-avatar" || data is Conversation;
        if (avatar && data is ChatMessage { ShowAvatar: false }) return;
        var source = data switch
        {
            ChatMessage { Avatar: { } address } when avatar => address.PathAndQuery,
            ChatMessage { HasThumbnail: true } msg when !avatar => msg.Thumbnail,
            Conversation { Avatar: { } address } => address.PathAndQuery,
            _ => ""
        };
        if (source.Length == 0 || ActiveChatAccount is not { } session) return;
        var generation = conversationGeneration;
        var cachedAvatars = avatar && data is ChatMessage ? avatarCache.GetOrCreateValue(session) : null;
        try
        {
            await thumbnailSlots.WaitAsync(session.Cancellation);
            try
            {
                if (!Current()) return;
                if (cachedAvatars is not null && cachedAvatars.TryGetValue(source, out var cached))
                {
                    image.Source = cached;
                    return;
                }
                using var bytes = new MemoryStream();
                await session.CopyAttachmentAsync(source, bytes, 4L * 1024 * 1024);
                if (!Current()) return;
                bytes.Position = 0;
                var bitmap = new BitmapImage();
                bitmap.BeginInit();
                bitmap.CacheOption = BitmapCacheOption.OnLoad;
                bitmap.DecodePixelWidth = avatar ? 96 : 380;
                bitmap.StreamSource = bytes;
                bitmap.EndInit();
                if (cachedAvatars is not null)
                {
                    if (cachedAvatars.Count >= 256) cachedAvatars.Clear();
                    cachedAvatars[source] = bitmap;
                }
                image.Source = bitmap;
            }
            finally { thumbnailSlots.Release(); }
        }
        catch (Exception error) when (IsAttachmentFailure(error) || error is NotSupportedException or FormatException or ArgumentException)
        {
            // The filename and manual preview stay available if a thumbnail is unavailable.
        }

        bool Current() => image.IsLoaded && ReferenceEquals(image.DataContext, data) &&
            (data is Conversation || conversationGeneration == generation) && visibleAccount == session;
    }

    private void StartScreenshot(object sender, RoutedEventArgs e)
    {
        try
        {
            Process.Start(new ProcessStartInfo("ms-screenclip:") { UseShellExecute = true });
            Status.Text = "框选截图后，回到输入框按 Ctrl + V";
        }
        catch (System.ComponentModel.Win32Exception)
        {
            Status.Text = "无法打开 Windows 截图工具";
        }
    }

    private async Task<bool> TryPasteImageAsync(KeyEventArgs e)
    {
        if (!CanSendAttachment || ActiveChatAccount is not { } session || visibleConversation is not { } conversation) return false;
        try
        {
            if (!Clipboard.ContainsImage() || Clipboard.ContainsText() || Clipboard.GetImage() is not { } image) return false;
            e.Handled = true;
            var file = Path.Combine(Path.GetTempPath(), $"截图-{DateTime.Now:yyyyMMdd-HHmmss}-{Guid.NewGuid():N}.png");
            try
            {
                var preview = new Image { Source = image, Height = 235, Stretch = Stretch.Uniform };
                if (!EditChatFields("发送截图", "发送图片", ($"发送到 {conversation.Name}", preview))) return true;
                var encoder = new PngBitmapEncoder();
                encoder.Frames.Add(BitmapFrame.Create(image));
                using (var output = File.Create(file)) encoder.Save(output);
                await SendAttachmentAsync(session, conversation, file, "image");
            }
            finally { DeleteTemporary(file); }
            return true;
        }
        catch (Exception error) when (IsAttachmentFailure(error) || error is System.Runtime.InteropServices.ExternalException)
        {
            Status.Text = "无法读取剪贴板图片";
            return e.Handled;
        }
    }

    private void ComposeDragOver(object sender, DragEventArgs e)
    {
        if (!e.Data.GetDataPresent(DataFormats.FileDrop)) return;
        e.Effects = CanSendAttachment ? DragDropEffects.Copy : DragDropEffects.None;
        e.Handled = true;
    }

    private async void ComposeDrop(object sender, DragEventArgs e)
    {
        if (!e.Data.GetDataPresent(DataFormats.FileDrop)) return;
        e.Handled = true;
        if (!CanSendAttachment || ActiveChatAccount is not { } session || visibleConversation is not { } conversation ||
            e.Data.GetData(DataFormats.FileDrop) is not string[] paths) return;
        var files = paths.Where(File.Exists).ToArray();
        if (files.Length == 0) return;
        if (files.Length != 1)
        {
            Status.Text = "请一次拖入一个文件";
            return;
        }
        if (MessageBox.Show(this, $"发送 {Path.GetFileName(files[0])} 到 {conversation.Name}？",
            "发送文件", MessageBoxButton.YesNo, MessageBoxImage.Question) == MessageBoxResult.Yes)
            await SendAttachmentAsync(session, conversation, files[0]);
    }

    private async void OpenEmoji(object sender, RoutedEventArgs e)
    {
        if (!Compose.IsEnabled || ActiveChatAccount is not { } session || visibleConversation is not { } conversation) return;
        await RunChatActionAsync(async () =>
        {
            var generation = conversationGeneration;
            var collection = await session.PostAsync("/enterprise/emoji/index", new { });
            if (session != ActiveChatAccount || generation != conversationGeneration) return;
            var window = ChatDialog("表情", out var panel, out var buttons, 430);
            var grid = new WrapPanel();
            panel.Children.Add(grid);
            foreach (var emoji in "😀 😃 😄 😊 😂 🤣 🙂 🙃 😉 😍 😘 😎 😅 😭 😢 😡 😮 😴 🤔 🙄 🤝 🙏 👋 👍 👎 👏 🎉 ❤️ 🔥 ✅".Split(' '))
            {
                var item = new Button { Content = emoji, ToolTip = emoji, FontSize = 21, Width = 46, Height = 40,
                    Padding = new Thickness(0), Margin = new Thickness(0, 0, 5, 5) };
                item.Click += (_, _) =>
                {
                    window.Close();
                    var caret = Compose.SelectionStart;
                    Compose.SelectedText = emoji;
                    Compose.CaretIndex = caret + emoji.Length;
                    Compose.Focus();
                };
                grid.Children.Add(item);
            }
            var saved = new ListBox { Height = 95, DisplayMemberPath = nameof(ChatChoice.Label) };
            saved.ItemsSource = collection.ValueKind == System.Text.Json.JsonValueKind.Array ? collection.EnumerateArray()
                .Select(item => new ChatChoice(ImgoApi.Field(item, "name"), item, ImgoApi.Field(item, "file_id")))
                .Where(item => item.Id.Length > 0).ToList() : [];
            panel.Children.Add(new TextBlock { Text = "收藏表情", Margin = new Thickness(0, 10, 0, 4) });
            panel.Children.Add(saved);
            ChatChoice? chosen = null;
            var send = AddDialogButton(buttons, "发送表情", () => { chosen = saved.SelectedItem as ChatChoice; window.Close(); }, primary: true);
            send.IsEnabled = false;
            saved.SelectionChanged += (_, _) => send.IsEnabled = saved.SelectedItem is ChatChoice;
            AddDialogButton(buttons, "关闭", window.Close).IsCancel = true;
            window.ShowDialog();
            if (chosen is null) return;
            await session.PostAsync("/enterprise/im/sendmessage", new
            {
                toContactId = conversation.Id, is_group = conversation.Group, id = Guid.NewGuid().ToString("N"),
                type = "image", file_id = chosen.Id, content = ImgoApi.Field(chosen.Data, "src")
            });
            RefreshAfterAction(session, "表情已发送");
        });
    }

    private async void PlayAttachment(object sender, RoutedEventArgs e)
    {
        if (sender is not Button { DataContext: ChatMessage { IsPlayable: true, HasAttachment: true } message } button ||
            ActiveChatAccount is not { } session) return;
        button.IsEnabled = false;
        try { await PlayMediaAsync(session, message, this); }
        catch (Exception error) when (IsAttachmentFailure(error))
        {
            if (error is ApiException { SessionExpired: true }) session.Expire();
            Status.Text = ErrorText(error);
        }
        finally { button.IsEnabled = true; }
    }

    private async Task PlayMediaAsync(AccountSession session, ChatMessage message, Window owner)
    {
        var suffix = Path.GetExtension(message.FileName).ToLowerInvariant();
        if (suffix is not (".wav" or ".mp3" or ".mp4" or ".wmv" or ".amr" or ".mov" or ".webm" or ".mkv" or ".avi" or ".3gp"))
            suffix = message.Type == "voice" ? ".wav" : ".mp4";
        var temp = Path.Combine(Path.GetTempPath(), $"imgo-media-{Guid.NewGuid():N}{suffix}");
        try
        {
            await using (var output = new FileStream(temp, FileMode.CreateNew, FileAccess.Write, FileShare.None, 81920, true))
                await session.CopyAttachmentAsync(message.Download, output, 256L * 1024 * 1024);
            if (!owner.IsVisible) return;

            var player = new MediaElement { LoadedBehavior = MediaState.Manual, UnloadedBehavior = MediaState.Close,
                Stretch = Stretch.Uniform, Source = new Uri(temp) };
            var playerPanel = new DockPanel { Background = (Brush)FindResource("Canvas") };
            var controls = new StackPanel { Orientation = Orientation.Horizontal, HorizontalAlignment = HorizontalAlignment.Center, Margin = new Thickness(8) };
            DockPanel.SetDock(controls, Dock.Bottom);
            playerPanel.Children.Add(controls);
            playerPanel.Children.Add(player);
            var timeline = new Slider { Width = 220, Minimum = 0, Margin = new Thickness(0, 0, 10, 0), IsEnabled = false,
                VerticalAlignment = VerticalAlignment.Center, IsMoveToPointEnabled = true };
            bool updatingTimeline = false;
            var clock = new DispatcherTimer { Interval = TimeSpan.FromMilliseconds(350) };
            clock.Tick += (_, _) =>
            {
                if (!timeline.IsEnabled || timeline.IsMouseCaptureWithin) return;
                updatingTimeline = true;
                timeline.Value = player.Position.TotalSeconds;
                updatingTimeline = false;
            };
            timeline.ValueChanged += (_, _) =>
            {
                if (!updatingTimeline && timeline.IsEnabled) player.Position = TimeSpan.FromSeconds(timeline.Value);
            };
            player.MediaOpened += (_, _) =>
            {
                if (player.NaturalDuration.HasTimeSpan)
                {
                    timeline.Maximum = player.NaturalDuration.TimeSpan.TotalSeconds;
                    timeline.IsEnabled = true;
                }
            };
            AddDialogButton(controls, "播放", player.Play);
            AddDialogButton(controls, "暂停", player.Pause);
            controls.Children.Add(timeline);
            var window = new Window { Title = message.AttachmentLabel, Owner = owner,
                WindowStartupLocation = WindowStartupLocation.CenterOwner, Width = message.Type == "voice" ? 520 : 760,
                Height = message.Type == "voice" ? 170 : 520, ShowInTaskbar = false, Content = playerPanel };
            AddDialogButton(controls, "关闭", window.Close).IsCancel = true;
            player.MediaFailed += (_, _) =>
            {
                playerPanel.Children.Remove(player);
                playerPanel.Children.Add(new TextBlock { Text = "Windows 无法播放此格式，可保存后使用本地播放器。",
                    HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center, TextWrapping = TextWrapping.Wrap });
                foreach (var control in controls.Children.OfType<Control>().Take(3)) control.IsEnabled = false;
                clock.Stop();
                player.Close();
            };
            window.Loaded += (_, _) => { player.Play(); clock.Start(); };
            window.Closed += (_, _) => { clock.Stop(); player.Close(); };
            window.ShowDialog();
        }
        finally { DeleteTemporary(temp); }
    }

    private async Task PreviewImageAsync(AccountSession session, ChatMessage message, Window owner, Func<bool>? stillRelevant = null)
    {
        using var bytes = new MemoryStream();
        await session.CopyAttachmentAsync(message.Download, bytes, 16L * 1024 * 1024);
        if (!owner.IsVisible || stillRelevant?.Invoke() == false) return;
        bytes.Position = 0;
        var bitmap = new BitmapImage();
        bitmap.BeginInit();
        bitmap.CacheOption = BitmapCacheOption.OnLoad;
        bitmap.DecodePixelWidth = 1600;
        bitmap.StreamSource = bytes;
        bitmap.EndInit();
        new Window
        {
            Title = message.AttachmentLabel, Owner = owner, WindowStartupLocation = WindowStartupLocation.CenterOwner,
            Width = 800, Height = 620, ShowInTaskbar = false, Background = (Brush)FindResource("Canvas"),
            Content = new Image { Source = bitmap, Stretch = Stretch.Uniform, Margin = new Thickness(12) }
        }.ShowDialog();
    }

    private static async Task<bool> SaveAttachmentAsync(AccountSession session, ChatMessage message, Window owner)
    {
        var filename = string.Concat(message.AttachmentLabel.Select(character =>
            Array.IndexOf(Path.GetInvalidFileNameChars(), character) >= 0 ? '_' : character)).Trim().TrimEnd('.');
        var dialog = new SaveFileDialog { Title = "保存聊天附件", FileName = filename.Length == 0 ? "附件" : filename,
            DefaultExt = Path.GetExtension(filename), OverwritePrompt = true };
        if (dialog.ShowDialog(owner) != true) return false;
        var overwrite = File.Exists(dialog.FileName);
        var temp = Path.Combine(Path.GetDirectoryName(dialog.FileName)!, $".imgo-{Guid.NewGuid():N}.part");
        try
        {
            await using (var output = new FileStream(temp, FileMode.CreateNew, FileAccess.Write, FileShare.None, 81920, true))
                await session.CopyAttachmentAsync(message.Download, output);
            File.Move(temp, dialog.FileName, overwrite);
            return true;
        }
        finally { DeleteTemporary(temp); }
    }

    private static void DeleteTemporary(string path)
    {
        try { File.Delete(path); }
        catch (Exception error) when (error is IOException or UnauthorizedAccessException) { }
    }
}
