// Refresh all visible copies after saving or receiving the group event.
component.methods.imgoGroupAvatarChanged = function (data) {
  const id = data.id || data.group_id;
  const ui = this.$refs.IMUI;
  if (!id || !data.avatar || !ui) return;
  if (ui.getContacts().some(item => String(item.id) === String(id))) ui.updateContact({id, avatar: data.avatar});
  for (const item of [this.currentChat, this.contactSetting].concat(this.contacts || [])) {
    if (item && String(item.id) === String(id)) this.$set(item, 'avatar', data.avatar);
  }
};
const avatarSocketAction = component.watch.socketAction;
component.watch.socketAction = function (event) {
  if (event && event.type === 'editGroupAvatar') {
    this.imgoGroupAvatarChanged(event.data || {});
    return;
  }
  return avatarSocketAction.call(this, event);
};
