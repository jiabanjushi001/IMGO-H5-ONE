// Preserve the shared preview API while keeping image and close control together.
component.data = function () { return {imageFailed: false}; };
component.computed = {
 isImage() { return /\.(?:jpe?g|png|gif|webp|bmp|svg|avif|ico)(?:[?#]|$)/i.test(this.url); }
};
component.mounted = function () {
 this._previewFocus = document.activeElement;
 this._previewEscape = event => { if (event.key === 'Escape') this.closeDrawer(); };
 document.addEventListener('keydown', this._previewEscape);
 this.$nextTick(() => { if (this.$refs.close) this.$refs.close.focus(); });
};
component.beforeDestroy = function () {
 document.removeEventListener('keydown', this._previewEscape);
 if (this._previewFocus && this._previewFocus.isConnected) this._previewFocus.focus();
};
component.render = function (h) {
 const close = h('button', {ref: 'close', class: 'imgo-preview-close', attrs: {type: 'button', title: '关闭（Esc）', 'aria-label': '关闭预览'}, on: {click: this.closeDrawer}}, [h('i', {class: 'el-icon-close'})]);
 let media;
 if (this.isImage) media = this.imageFailed ? h('div', {class: 'imgo-preview-error'}, '图片加载失败，请关闭后重试') :
  h('img', {class: 'imgo-preview-image', attrs: {src: this.url, alt: '图片预览'}, on: {error: () => {this.imageFailed = true;}}});
 else media = h('iframe', {class: 'imgo-preview-document', attrs: {src: this.url, frameborder: '0', title: '文件预览'}});
 return h('div', {class: 'imgo-preview-overlay', attrs: {role: 'dialog', 'aria-modal': 'true', 'aria-label': '文件预览'}, on: {click: event => {if (event.target === event.currentTarget) this.closeDrawer();}}}, [
  h('div', {class: ['imgo-preview-frame', this.isImage ? 'is-image' : 'is-document']}, [close, media])
 ]);
};
// The webpack normalizer receives its render function separately.
k = component.render;
