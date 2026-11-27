using System.IO;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using Microsoft.Win32;

namespace Imgo.Desktop;

public partial class MainWindow
{
    private const string PreparedVideoRoute = "/enterprise/callVideo/index";
    private const int VideoPageSize = 50;

    private sealed record PreparedVideo(long Id, string Name, string Url, string Extension, long Size)
    {
        public string Label => Size switch
        {
            <= 0 => Name,
            < 1048576 => $"{Name} · {Math.Max(1, Size / 1024d):0} KB",
            _ => $"{Name} · {Size / 1048576d:0.0} MB"
        };
    }

    private void StartVideoCall(object sender, RoutedEventArgs e)
    {
        if (activeCall is not null || ActiveChatAccount is not { CallsEnabled: true } session || PrivateCallTarget() == 0)
            return;
        if (!session.IsSuperAdmin && !session.IsAgent)
        {
            StartOutgoingCall(true);
            return;
        }
        var recipient = PrivateCallTarget();
        var page = OpenSidePage("选择视频通话方式");
        var live = AddDialogButton(page.Body, "使用实时摄像头", () => Dial());
        live.HorizontalAlignment = HorizontalAlignment.Left;
        page.Body.Children.Add(new TextBlock
        {
            Text = "预录视频", FontWeight = FontWeights.SemiBold, Margin = new Thickness(0, 14, 0, 5)
        });
        var status = new TextBlock { TextWrapping = TextWrapping.Wrap };
        page.Body.Children.Add(status);
        var videos = new ListBox
        {
            DisplayMemberPath = nameof(PreparedVideo.Label), MaxHeight = 300, Margin = new Thickness(0, 8, 0, 0)
        };
        page.Body.Children.Add(videos);
        var videoPage = 1;
        var fetching = false;
        var upload = AddDialogButton(page.Footer, "上传视频", () => { });
        var previous = AddDialogButton(page.Footer, "上一页", () => { });
        var next = AddDialogButton(page.Footer, "下一页", () => { });
        var start = AddDialogButton(page.Footer, "发起预录通话", () =>
        {
            if (videos.SelectedItem is PreparedVideo video) Dial(video);
        }, primary: true);
        previous.Click += (_, _) => _ = LoadVideos(videoPage - 1);
        next.Click += (_, _) => _ = LoadVideos(videoPage + 1);
        start.IsEnabled = false;
        videos.SelectionChanged += (_, _) => start.IsEnabled = !fetching && videos.SelectedItem is PreparedVideo;
        upload.Click += async (_, _) =>
        {
            if (fetching || !IsSidePageOpen(page) || ActiveChatAccount != session) return;
            var picker = new OpenFileDialog
            {
                Title = "上传预录视频",
                Filter = "视频文件 (*.mp4;*.mov;*.webm;*.m4v)|*.mp4;*.mov;*.webm;*.m4v"
            };
            if (picker.ShowDialog(this) != true || ActiveChatAccount != session || !IsSidePageOpen(page)) return;
            SetLoading(true);
            status.Text = "正在上传视频…";
            try
            {
                var result = await session.UploadAsync(picker.FileName, "uploadfile");
                if (ImgoApi.Number(result, "file_id") <= 0) throw new IOException("服务器未确认视频上传。");
            }
            catch (Exception error) when (IsAttachmentFailure(error))
            {
                SetContactActionError(session, status, error);
                return;
            }
            finally { SetLoading(false); }
            await LoadVideos(1);
        };
        _ = LoadVideos(1);

        void Dial(PreparedVideo? video = null)
        {
            if (!IsSidePageOpen(page) || ActiveChatAccount != session || PrivateCallTarget() != recipient || activeCall is not null)
                return;
            CloseConversationPane();
            StartOutgoingCall(true, video);
        }

        async Task LoadVideos(int number)
        {
            if (fetching || number <= 0 || !IsSidePageOpen(page) || ActiveChatAccount != session) return;
            SetLoading(true);
            status.Text = "正在读取视频…";
            videos.ItemsSource = null;
            try
            {
                var result = await session.PostPageAsync(PreparedVideoRoute, new { page = number, limit = VideoPageSize });
                if (!IsSidePageOpen(page) || ActiveChatAccount != session) return;
                var items = result.Data.ValueKind == JsonValueKind.Array
                    ? result.Data.EnumerateArray().Select(ReadPreparedVideo).OfType<PreparedVideo>().ToArray()
                    : [];
                videoPage = number;
                videos.ItemsSource = items;
                previous.Tag = number > 1;
                next.Tag = result.Count >= (long)number * VideoPageSize;
                status.Text = items.Length > 0 ? "接通后将播放所选视频。" : "此页没有可用于通话的视频。";
            }
            catch (Exception error) when (IsRequestFailure(error))
            {
                SetContactActionError(session, status, error);
                previous.Tag = videoPage > 1;
                next.Tag = false;
            }
            finally { SetLoading(false); }
        }

        void SetLoading(bool loading)
        {
            fetching = loading;
            upload.IsEnabled = !loading;
            start.IsEnabled = !loading && videos.SelectedItem is PreparedVideo;
            previous.IsEnabled = !loading && previous.Tag is true;
            next.IsEnabled = !loading && next.Tag is true;
        }
    }

    private static PreparedVideo? ReadPreparedVideo(JsonElement item)
    {
        var id = ImgoApi.Number(item, "file_id");
        var url = ImgoApi.Field(item, "url");
        var extension = ImgoApi.Field(item, "ext").TrimStart('.').ToLowerInvariant();
        if (id <= 0 || url.Length == 0 || extension is not ("mp4" or "mov" or "webm" or "m4v")) return null;
        var name = ImgoApi.Field(item, "name");
        return new PreparedVideo(id, name.Length > 0 ? name : "未命名视频", url, extension, ImgoApi.Number(item, "size"));
    }

    private static async Task<string> StagePreparedVideo(Call call, long limitMb)
    {
        var video = call.Prepared!;
        if (!Uri.TryCreate(call.Session.Server, video.Url, out var source) ||
            !source.AbsolutePath.StartsWith("/storage/video/", StringComparison.Ordinal))
            throw new IOException("预录视频地址无效。");

        var folder = Directory.CreateTempSubdirectory("imgo-call-").FullName;
        try
        {
            var assets = Path.Combine(AppContext.BaseDirectory, "CallMedia");
            foreach (var file in new[] { "index.html", "call.js", "call.css" })
                File.Copy(Path.Combine(assets, file), Path.Combine(folder, file));
            var target = Path.Combine(folder, "prepared." + video.Extension);
            await using (var stream = new FileStream(target, FileMode.CreateNew, FileAccess.Write, FileShare.None, 81920, useAsync: true))
            {
                var limit = (limitMb is >= 1 and <= 200 ? limitMb : 200) * 1048576L;
                await call.Session.CopyAttachmentAsync(video.Url, stream, limit, call.MediaCancellation.Token);
                if (stream.Length == 0) throw new IOException("预录视频是空文件。");
            }
            return folder;
        }
        catch
        {
            await RemovePreparedVideo(folder);
            throw;
        }
    }

    private static async Task RemovePreparedVideo(string folder)
    {
        for (var attempt = 0; attempt < 5; attempt++)
        {
            try
            {
                Directory.Delete(folder, recursive: true);
                return;
            }
            catch (DirectoryNotFoundException) { return; }
            catch (Exception error) when (error is IOException or UnauthorizedAccessException)
            {
                await Task.Delay(125 * (attempt + 1)).ConfigureAwait(false);
            }
        }
    }
}
