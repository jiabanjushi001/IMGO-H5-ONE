// Refresh conversation summaries and the open conversation after a socket gap.
const socketRecoveryWatch = component.watch.socketAction;
component.watch.socketAction = function (event) {
  if (event && event.type === 'socketRecovered') {
    this.imgoRecoverSocketGap();
    return;
  }
  return socketRecoveryWatch.call(this, event);
};

component.methods.imgoRecoverSocketGap = async function () {
  if (this._imgoSocketRecoveryBusy) {
    this._imgoSocketRecoveryAgain = true;
    return;
  }
  const ui = this.$refs.IMUI;
  if (!ui) return;
  this._imgoSocketRecoveryBusy = true;
  try {
    const result = await this.$api.imApi.getContactsAPI();
    if (Number(result.code) !== 0 || !Array.isArray(result.data)) return;
    const serverContacts = result.data;
    const existing = new Set(ui.getContacts().map(item => String(item.id)));
    for (const contact of serverContacts) {
      const next = Object.assign({}, contact);
      if (next.type) next.lastContent = ui.lastContentRender({type: next.type, content: next.lastContent});
      if (existing.has(String(next.id))) ui.updateContact(next);
      else ui.appendContact(next);
    }
    this.contacts = serverContacts;
    this.$store.commit('initContacts', serverContacts);
    this.unread = serverContacts.reduce((sum, item) => sum + (Number(item.is_notice) === 1 ? Number(item.unread) || 0 : 0), 0);
    this.atUnread = serverContacts.reduce((sum, item) => sum + (Number(item.is_at) || 0), 0);
    const current = ui.getCurrentContact();
    if (current && current.id && current.id !== 'system') {
      const messages = await this.$api.imApi.getMessageListAPI({toContactId: current.id, is_group: current.is_group, page: 1, limit: 50});
      if (Number(messages.code) === 0 && Array.isArray(messages.data)) {
        const known = new Set(ui.getMessages(current.id).map(item => String(item.id)));
        for (const original of messages.data) {
          if (known.has(String(original.id))) continue;
          const message = Object.assign({}, original, {toContactId: current.id});
          ui.appendMessage(message, false);
          known.add(String(message.id));
        }
        const latest = messages.data[messages.data.length - 1];
        if (latest) ui.updateContact({id: current.id, unread: 0, lastContent: ui.lastContentRender(latest), lastSendTime: latest.sendTime});
      }
    }
    this.lastMessages = ui.lastMessages;
    this.initMenus(ui);
  } catch (_) {
    // A later heartbeat, visibility change, or reconnect retries the sync.
  } finally {
    this._imgoSocketRecoveryBusy = false;
    if (this._imgoSocketRecoveryAgain) {
      this._imgoSocketRecoveryAgain = false;
      this.$nextTick(() => this.imgoRecoverSocketGap());
    }
  }
};
