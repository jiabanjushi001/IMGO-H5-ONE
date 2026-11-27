using System.Diagnostics;
using System.IO;
using System.Runtime.InteropServices;
using System.Text;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Threading;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private void OpenVoiceMenu(object sender, RoutedEventArgs e)
    {
        if (sender is not Button anchor || !CanSendAttachment) return;
        var menu = MakeMenu(anchor);
        AddMenuAction(menu, "录制语音…", RecordVoiceAsync);
        AddMenuAction(menu, "选择语音文件…", () => PickAttachmentAsync("voice"));
        menu.IsOpen = true;
    }

    private async Task RecordVoiceAsync()
    {
        if (!CanSendAttachment || ActiveChatAccount is not { } session || visibleConversation is not { } conversation) return;
        var path = Path.Combine(Path.GetTempPath(), $"语音-{Guid.NewGuid():N}.wav");
        using var recorder = new WaveRecorder();
        var window = ChatDialog("录制语音", out var panel, out var buttons, 435);
        var status = new TextBlock { Text = $"发送到 {conversation.Name}", TextWrapping = TextWrapping.Wrap };
        panel.Children.Add(status);
        var duration = Stopwatch.StartNew();
        duration.Reset();
        var timer = new DispatcherTimer { Interval = TimeSpan.FromMilliseconds(250) };
        var preview = new MediaElement { LoadedBehavior = MediaState.Manual, UnloadedBehavior = MediaState.Close,
            Width = 1, Height = 1, IsHitTestVisible = false };
        panel.Children.Add(preview);

        var record = AddDialogButton(buttons, "开始录音", () => { });
        var listen = AddDialogButton(buttons, "试听", () =>
        {
            preview.Position = TimeSpan.Zero;
            preview.Play();
        });
        var send = AddDialogButton(buttons, "发送", () => window.DialogResult = true, primary: true);
        AddDialogButton(buttons, "取消", window.Close).IsCancel = true;
        listen.IsEnabled = send.IsEnabled = false;

        record.Click += (_, _) =>
        {
            if (duration.IsRunning) { Stop(); return; }
            try
            {
                preview.Close();
                DeleteTemporary(path);
                listen.IsEnabled = send.IsEnabled = false;
                recorder.Start();
                duration.Restart();
                timer.Start();
                record.Content = "停止录音";
                status.Text = "录音中 00:00 / 01:00";
            }
            catch (IOException error)
            {
                status.Text = error.Message;
            }
        };
        timer.Tick += (_, _) =>
        {
            if (duration.Elapsed >= TimeSpan.FromMinutes(1)) { Stop(); return; }
            status.Text = $"录音中 {duration.Elapsed:mm\\:ss} / 01:00";
        };
        preview.MediaFailed += (_, _) => status.Text = "无法试听这段录音，可以重新录制。";
        window.Closed += (_, _) => { timer.Stop(); recorder.Dispose(); preview.Close(); };

        try
        {
            if (window.ShowDialog() == true && session == ActiveChatAccount && visibleConversation?.Id == conversation.Id)
                await SendAttachmentAsync(session, conversation, path, "voice");
        }
        finally { DeleteTemporary(path); }

        void Stop()
        {
            duration.Stop();
            timer.Stop();
            record.Content = "重新录音";
            try
            {
                recorder.Save(path);
                if (duration.Elapsed < TimeSpan.FromSeconds(1) || new FileInfo(path).Length <= 44)
                {
                    status.Text = "语音至少录制 1 秒";
                    return;
                }
                preview.Source = new Uri(path);
                listen.IsEnabled = send.IsEnabled = true;
                status.Text = $"已录制 {(int)duration.Elapsed.TotalSeconds} 秒，可试听后发送";
            }
            catch (IOException error)
            {
                status.Text = error.Message;
            }
        }
    }

    private sealed class WaveRecorder : IDisposable
    {
        private readonly string alias = $"imgo{Guid.NewGuid():N}";
        private bool opened;

        public void Start()
        {
            Command($"open new type waveaudio alias {alias}");
            opened = true;
            try { Command($"record {alias}"); }
            catch { Dispose(); throw; }
        }

        public void Save(string path)
        {
            try
            {
                Command($"stop {alias}");
                Command($"save {alias} \"{path}\"");
            }
            finally { Dispose(); }
        }

        public void Dispose()
        {
            if (!opened) return;
            _ = MciSendString($"close {alias}", null, 0, IntPtr.Zero);
            opened = false;
        }

        private static void Command(string command)
        {
            uint error = MciSendString(command, null, 0, IntPtr.Zero);
            if (error == 0) return;
            var description = new StringBuilder(255);
            _ = MciGetErrorString(error, description, description.Capacity);
            throw new IOException($"录音设备操作失败（{error}）{description}");
        }

        [DllImport("winmm.dll", EntryPoint = "mciSendStringW", CharSet = CharSet.Unicode, ExactSpelling = true)]
        private static extern uint MciSendString(string command, StringBuilder? result, int resultLength, IntPtr callback);

        [DllImport("winmm.dll", EntryPoint = "mciGetErrorStringW", CharSet = CharSet.Unicode, ExactSpelling = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool MciGetErrorString(uint error, StringBuilder result, int resultLength);
    }
}
