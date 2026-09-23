// Extend the existing Vue 2 group settings; the server owns permission decisions.
const avatarData = component.data;
component.data = function () {
  return Object.assign({}, avatarData.call(this), {imgoAvatarBusy: false});
};
component.methods.imgoChangeGroupAvatar = async function (event) {
  const input = event.target, file = input.files && input.files[0];
  input.value = '';
  if (!file || this.imgoAvatarBusy || !this.groupInfo.canEditAvatar) return;
  if (!/^image\/(png|jpeg|gif|webp)$/.test(file.type) || file.size > 5 * 1024 * 1024) {
    this.$message.warning('请选择不超过 5 MB 的 JPG、PNG、GIF 或 WebP 图片');
    return;
  }
  this.imgoAvatarBusy = true;
  const groupId = this.contact.id;
  try {
    const form = new FormData();
    form.append('file', file);
    const upload = await this.$api.imApi.sendFileAPI(form);
    if (Number(upload.code) !== 0 || !upload.data || !upload.data.file_id) throw new Error(upload.msg || '图片上传失败');
    const result = await this.$api.imApi.editGroupAvatarAPI({group_id: groupId, file_id: upload.data.file_id});
    if (Number(result.code) !== 0 || !result.data || !result.data.avatar) throw new Error(result.msg || '群头像保存失败');
    if (this._isDestroyed) return;
    this.$set(this.groupInfo, 'avatar', result.data.avatar);
    this.$emit('avatarChanged', {id: groupId, avatar: result.data.avatar});
    this.$message.success('群头像已更新');
  } catch (error) {
    if (!this._isDestroyed) this.$message.error(error.message || '群头像更新失败，请重试');
  } finally {
    if (!this._isDestroyed) this.imgoAvatarBusy = false;
  }
};
const legacyGroupRender = J;
J = function () {
  const root = legacyGroupRender.call(this), h = this.$createElement;
  const info = this.groupInfo || {}, allowed = info.canEditAvatar === true;
  root.children[0] = h('div', {class: 'imgo-group-avatar-row', style: {display: 'flex', alignItems: 'center', gap: '16px', padding: '0 0 20px'}}, [
    h('el-avatar', {props: {shape: 'square', size: 64, src: info.avatar || this.contact.avatar}}),
    h('div', {style: {flex: '1', minWidth: '0'}}, [
      h('div', {style: {fontWeight: '600', marginBottom: '6px'}}, [this.contact.displayName]),
      h('div', {style: {fontSize: '12px', color: '#909399'}}, ['群主：' + (info.ownerName || '—')]),
      allowed ? h('div', {style: {fontSize: '12px', color: '#909399', marginTop: '6px'}}, ['图片不超过 5 MB']) : null
    ]),
    allowed ? h('input', {ref: 'imgoGroupAvatarFile', style: {display: 'none'}, attrs: {type: 'file', accept: 'image/png,image/jpeg,image/gif,image/webp'}, on: {change: this.imgoChangeGroupAvatar}}) : null,
    allowed ? h('el-button', {props: {size: 'small', type: 'primary', plain: true, loading: this.imgoAvatarBusy}, on: {click: () => this.$refs.imgoGroupAvatarFile.click()}}, ['更换群头像']) : null
  ]);
  return root;
};
