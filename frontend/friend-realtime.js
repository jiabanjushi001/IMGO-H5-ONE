// Vue 2 chat adapter: update only the invitation contact, preserving the active chat.
const originalSocketAction = component.watch.socketAction;
component.watch.socketAction = function (event) {
  if (event && (event.type === 'friendApply' || event.type === 'friendApplyChanged')) {
    this.imgoRefreshInvitations(event.type === 'friendApply');
    return;
  }
  return originalSocketAction.call(this, event);
};
component.methods.imgoRefreshInvitations = async function (incoming) {
  const request = (this._imgoInviteRequest || 0) + 1;
  this._imgoInviteRequest = request;
  try {
    const result = await this.$api.friendApi.getApplyMsg();
    const ui = this.$refs.IMUI;
    if (this._isDestroyed || request !== this._imgoInviteRequest || !ui || Number(result.code) !== 0) return;
    const count = Math.max(0, Number(result.data) || 0);
    let contact = ui.getContacts().find(item => item.id === 'system');
    if (!contact && Number(this.globalConfig.sysInfo.runMode) !== 2) return;
    if (!contact) {
      contact = {id: 'system', displayName: '新邀请', name_py: 'xinyaoqing', avatar: xe,
        is_group: 2, index: '[1]系统消息', click(done) { done(); },
        renderContainer: () => this.$createElement(Ce), unread: 0, is_notice: 1};
      ui.appendContact(contact);
    }
    const previous = Number(contact.unread) || 0;
    ui.updateContact({id: 'system', unread: count, lastContent: count ? '新的申请' : '',
      lastSendTime: incoming ? Date.now() : contact.lastSendTime});
    this.unread = Math.max(0, (Number(this.unread) || 0) - previous + count);
    this.lastMessages = ui.lastMessages;
    this.initMenus(ui);
  } catch (_) {
    // Keep existing state on network failure; retry on the next event/reconnection.
  }
};
