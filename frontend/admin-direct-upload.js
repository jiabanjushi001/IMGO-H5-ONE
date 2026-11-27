/* IMGO_ADMIN_DIRECT_UPLOAD_BEGIN */
function imgoAdminDirectUploadEnabled() {
  var config = imgoNormalizeConfig(o().get('globalConfig'));
  var capability = config && config.fileUpload && config.fileUpload.directUpload;
  return !!(capability && typeof capability === 'object' && capability.enabled === true && capability.provider === 'aliyun');
}

function imgoAdminFileExtension(file) {
  var match = String(file && file.name || '').match(/\.([a-z0-9]+)$/i);
  if (match) return match[1].toLowerCase();
  var byType = {'image/jpeg':'jpg','image/png':'png','image/gif':'gif','image/webp':'webp','video/mp4':'mp4','audio/wav':'wav'};
  return byType[String(file && file.type || '').toLowerCase()] || '';
}

function imgoAdminOSSPost(prepared, file) {
  return new Promise(function (resolve, reject) {
    var form = new FormData();
    Object.keys(prepared.fields || {}).forEach(function (key) { form.append(key, prepared.fields[key]); });
    form.append('file', file, file.name || ('upload.' + imgoAdminFileExtension(file)));
    var xhr = new XMLHttpRequest();
    xhr.open('POST', prepared.url, true);
    xhr.timeout = 30 * 60 * 1000;
    xhr.onload = function () {
      if (xhr.status >= 200 && xhr.status < 300) resolve();
      else reject(Error('OSS 上传失败（HTTP ' + (xhr.status || '未知') + '）'));
    };
    xhr.onerror = function () { reject(Error('无法连接 OSS，请检查 OSS 跨域设置')); };
    xhr.ontimeout = function () { reject(Error('OSS 上传超时')); };
    xhr.send(form);
  });
}

async function imgoAdminUploadOne(file, uploadAction, message) {
  var ext = imgoAdminFileExtension(file);
  var prepare = await Ti({url:'common/upload/prepareDirect',method:'post',data:{
    uploadAction: uploadAction,
    fileName: file.name || ('upload.' + ext),
    fileSize: Number(file.size || 0),
    ext: ext,
    messageType: message && message.type || '',
    voice: message && message.type === 'voice' ? 1 : 0
  }});
  if (!prepare || Number(prepare.code) !== 0 || !prepare.data) {
    var unavailable = Error(prepare && prepare.msg || '无法获取 OSS 上传凭证');
    unavailable.imgoPrepareUnavailable = [404,409,410].includes(Number(prepare && prepare.code));
    throw unavailable;
  }
  var prepared = prepare.data, transferError = null;
  for (var attempt = 0; attempt < 2; attempt += 1) {
    try { await imgoAdminOSSPost(prepared, file); transferError = null; }
    catch (error) { transferError = error; }
    var complete = await Ti({url:'common/upload/completeDirect',method:'post',data:{ticket:prepared.ticket,message:message || undefined}});
    if (complete && Number(complete.code) === 0) return complete;
    if (!transferError) transferError = Error(complete && complete.msg || 'OSS 文件确认失败');
  }
  throw transferError || Error('OSS 上传失败');
}

async function imgoAdminUpload(form, uploadAction, legacyUpload) {
  if (!imgoAdminDirectUploadEnabled() || !form || typeof form.get !== 'function') return legacyUpload();
  var file = form.get('file');
  if (!file) return legacyUpload();
  var message = null, rawMessage = form.get('message');
  if (rawMessage) {
    try { message = typeof rawMessage === 'string' ? JSON.parse(rawMessage) : rawMessage; }
    catch (_) { message = null; }
  }
  try {
    var poster = form.get('poster');
    if (poster && message) {
      var posterResult = await imgoAdminUploadOne(poster, 'uploadFile', null);
      if (posterResult.data && posterResult.data.url) {
        message.extends = Object.assign({}, message.extends || {}, {poster:posterResult.data.url});
      }
    }
    return await imgoAdminUploadOne(file, uploadAction, message);
  } catch (error) {
    if (error && error.imgoPrepareUnavailable) return legacyUpload();
    throw error;
  }
}

// Element UI's upload component is also used by the system settings page for
// the site logo and navigation icons. Keep its callback contract while routing
// the file through the same direct-upload/fallback path as chat and avatars.
if (typeof window !== 'undefined') {
  window.imgoAdminElementUpload = function (request, uploadAction) {
    var form = new FormData();
    form.append('file', request.file);
    Object.keys(request.data || {}).forEach(function (key) {
      form.append(key, request.data[key]);
    });
    var legacyUpload = function () {
      return Ti({
        url: 'common/upload/' + uploadAction,
        method: 'post',
        data: form,
        timeout: 0,
        headers: {'Content-Type': 'multipart/form-data'}
      });
    };
    var task = imgoAdminUpload(form, uploadAction, legacyUpload);
    task.then(function (result) {
      if (request.onSuccess) request.onSuccess(result);
    }).catch(function (error) {
      if (request.onError) request.onError(error);
    });
    return task;
  };
}
/* IMGO_ADMIN_DIRECT_UPLOAD_END */
