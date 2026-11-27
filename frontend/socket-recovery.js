// Refresh conversation summaries and the open conversation after a socket gap.
const socketRecoveryWatch = component.watch.socketAction;
component.watch.socketAction = function (event) {
  if (event && event.type === 'socketRecovered') {
    this.imgoRecoverSocketGap();
    return;
  }
  return socketRecoveryWatch.call(this, event);
};

function imgoMessageTokens(message) {
  const tokens = [];
  if (message && message.id !== undefined && message.id !== null && String(message.id) !== '') {
    tokens.push('id:' + String(message.id));
  }
  if (message && Number(message.msg_id) > 0) {
    tokens.push('msg:' + String(Number(message.msg_id)));
  }
  return tokens;
}

function imgoMessageOrder(message) {
  const msgId = Number(message && message.msg_id);
  const sendTime = Number(message && message.sendTime);
  return {
    msgId: Number.isFinite(msgId) && msgId > 0 ? msgId : 0,
    sendTime: Number.isFinite(sendTime) && sendTime > 0 ? sendTime : 0
  };
}

function imgoCompareMessages(left, right) {
  const a = imgoMessageOrder(left);
  const b = imgoMessageOrder(right);
  if (a.msgId && b.msgId && a.msgId !== b.msgId) return a.msgId - b.msgId;
  if (a.sendTime !== b.sendTime) return a.sendTime - b.sendTime;
  return 0;
}

// Socket recovery fetches a wider latest-message window than the ordinary
// initial page. Only append records newer than the newest visible message;
// older records belong to history pagination and appending them here puts
// historical messages after the newest message.
component.methods.imgoAppendRecoveredMessages = function (ui, current, records) {
  const visible = ui.getMessages(current.id) || [];
  const known = new Set();
  let newestMsgId = 0;
  let newestSendTime = 0;
  for (const message of visible) {
    for (const token of imgoMessageTokens(message)) known.add(token);
    const order = imgoMessageOrder(message);
    newestMsgId = Math.max(newestMsgId, order.msgId);
    newestSendTime = Math.max(newestSendTime, order.sendTime);
  }

  const recovered = [];
  for (const original of records) {
    const tokens = imgoMessageTokens(original);
    if (tokens.some(token => known.has(token))) continue;
    const order = imgoMessageOrder(original);
    const isNewer = visible.length === 0 ||
      (newestMsgId > 0 && order.msgId > newestMsgId) ||
      (newestMsgId === 0 && newestSendTime > 0 && order.sendTime > newestSendTime);
    if (!isNewer) continue;
    const message = Object.assign({}, original, {toContactId: current.id});
    recovered.push(message);
    for (const token of tokens) known.add(token);
  }

  recovered.sort(imgoCompareMessages);
  for (const message of recovered) ui.appendMessage(message, false);
  return recovered;
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
      if (Number(next.unread) > 0 && ui.CacheMessageLoaded && typeof ui.CacheMessageLoaded.remove === 'function') {
        ui.CacheMessageLoaded.remove(next.id);
      }
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
        this.imgoAppendRecoveredMessages(ui, current, messages.data);
        const latest = messages.data[messages.data.length - 1];
        if (latest) ui.updateContact({id: current.id, lastContent: ui.lastContentRender(latest), lastSendTime: latest.sendTime});
        if (typeof this.imgoCommitContactUnread === 'function') this.imgoCommitContactUnread(current, ui);
        else ui.updateContact({id: current.id, unread: 0});
      }
    }
    this.lastMessages = ui.lastMessages;
    this.initMenus(ui);
  } catch (error) {
    if (typeof console !== 'undefined' && console.warn) console.warn('聊天断线恢复失败', error);
    // A later heartbeat, visibility change, or reconnect retries the sync.
  } finally {
    this._imgoSocketRecoveryBusy = false;
    if (this._imgoSocketRecoveryAgain) {
      this._imgoSocketRecoveryAgain = false;
      this.$nextTick(() => this.imgoRecoverSocketGap());
    }
  }
};
