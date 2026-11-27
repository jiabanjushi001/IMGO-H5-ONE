/* IMGO_MEDIA_AUTH_RECOVERY_BEGIN */
var imgoMediaOriginalMounted = component.mounted;
var imgoMediaOriginalBeforeDestroy = component.beforeDestroy;

function imgoMediaCallHook(hook, instance) {
  if (Array.isArray(hook)) {
    hook.forEach(function (item) {
      if (typeof item === 'function') item.call(instance);
    });
    return;
  }
  if (typeof hook === 'function') hook.call(instance);
}

function imgoProtectedMediaURL(value) {
  if (!value || typeof window === 'undefined') return null;
  try {
    var url = new URL(value, window.location.href);
    if (url.origin !== window.location.origin) return null;
    if (!/^\/(?:storage|image|file|video|voice|emoji|cover|avatar)\//.test(url.pathname)) return null;
    return url;
  } catch (_) {
    return null;
  }
}

component.methods.imgoSyncMediaCookie = function () {
  if (this._imgoMediaCookieSyncPromise) return this._imgoMediaCookieSyncPromise;
  var api = this.$api && this.$api.imApi && this.$api.imApi.getContactsAPI;
  if (typeof api !== 'function') return Promise.reject(new Error('media auth api unavailable'));
  var self = this;
  this._imgoMediaCookieSyncPromise = Promise.resolve().then(function () {
    return api.call(self.$api.imApi);
  }).then(function (result) {
    if (!result || Number(result.code) !== 0) {
      throw new Error(result && result.msg ? result.msg : 'media auth failed');
    }
    return true;
  });
  this._imgoMediaCookieSyncPromise = this._imgoMediaCookieSyncPromise.then(function (result) {
    self._imgoMediaCookieSyncPromise = null;
    return result;
  }, function (error) {
    self._imgoMediaCookieSyncPromise = null;
    throw error;
  });
  return this._imgoMediaCookieSyncPromise;
};

component.methods.imgoRetryProtectedImage = function (image) {
  if (!image || image.tagName !== 'IMG' || image.getAttribute('data-imgo-media-retry') === '1') return;
  var source = image.currentSrc || image.getAttribute('src') || image.src;
  var url = imgoProtectedMediaURL(source);
  if (!url) return;
  image.setAttribute('data-imgo-media-retry', '1');
  this.imgoSyncMediaCookie().then(function () {
    url.searchParams.set('_imgo_media_retry', String(Date.now()));
    image.src = url.toString();
  }).catch(function () {
    // Keep the original browser error state. A later navigation can try again.
  });
};

component.mounted = function () {
  imgoMediaCallHook(imgoMediaOriginalMounted, this);
  if (typeof document === 'undefined') return;
  this._imgoMediaErrorHandler = function (event) {
    this.imgoRetryProtectedImage(event && event.target);
  }.bind(this);
  document.addEventListener('error', this._imgoMediaErrorHandler, true);
};

component.beforeDestroy = function () {
  if (typeof document !== 'undefined' && this._imgoMediaErrorHandler) {
    document.removeEventListener('error', this._imgoMediaErrorHandler, true);
  }
  imgoMediaCallHook(imgoMediaOriginalBeforeDestroy, this);
};
/* IMGO_MEDIA_AUTH_RECOVERY_END */
