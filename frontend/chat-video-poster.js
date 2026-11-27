function imgoExtractVideoFirstFrame(file) {
  return new Promise(function (resolve, reject) {
    var video = document.createElement('video');
    var objectURL = URL.createObjectURL(file);
    var finished = false;
    var timer = setTimeout(function () { finish(new Error('视频首帧读取超时')); }, 10000);

    function cleanup() {
      clearTimeout(timer);
      video.removeAttribute('src');
      video.load();
      URL.revokeObjectURL(objectURL);
    }
    function finish(error, blob) {
      if (finished) return;
      finished = true;
      cleanup();
      if (error) reject(error); else resolve(blob);
    }
    video.muted = true;
    video.playsInline = true;
    video.preload = 'auto';
    video.addEventListener('error', function () { finish(new Error('无法读取视频')); }, { once: true });
    video.addEventListener('loadeddata', function () {
      try {
        var width = Number(video.videoWidth || 0);
        var height = Number(video.videoHeight || 0);
        if (!width || !height) throw new Error('视频尺寸无效');
        var maxWidth = 1280;
        var scale = Math.min(1, maxWidth / width);
        var canvas = document.createElement('canvas');
        canvas.width = Math.max(1, Math.round(width * scale));
        canvas.height = Math.max(1, Math.round(height * scale));
        var context = canvas.getContext('2d');
        if (!context) throw new Error('无法生成视频封面');
        context.drawImage(video, 0, 0, canvas.width, canvas.height);
        canvas.toBlob(function (blob) {
          if (!blob) return finish(new Error('无法生成视频封面'));
          finish(null, blob);
        }, 'image/jpeg', 0.86);
      } catch (error) {
        finish(error);
      }
    }, { once: true });
    video.src = objectURL;
    video.load();
  });
}

var imgoOriginalUploadVideo = component.methods.uploadVideo;
component.methods.uploadVideo = function (event) {
  var self = this;
  var input = event && (event.target || event.srcElement);
  var file = input && input.files && input.files[0];
  if (!file) return imgoOriginalUploadVideo.call(this, event);
  return imgoExtractVideoFirstFrame(file).then(function (poster) {
    Object.defineProperty(file, '__imgoPoster', { value: poster, configurable: true });
  }).catch(function () {
    // The server still has its FFmpeg/placeholder fallback for unreadable videos.
  }).then(function () {
    return imgoOriginalUploadVideo.call(self, event);
  });
};
