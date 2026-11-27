/* IMGO_CHAT_MESSAGE_STATE_BEGIN */
var imgoMessageStateOriginalData = component.data;
var imgoMessageStateOriginalMethods = component.methods || {};
var imgoMessageStateOriginalChangeContact = imgoMessageStateOriginalMethods.handleChangeContact;
var imgoMessageStateOriginalReceive = imgoMessageStateOriginalMethods.recieveMsg;

function imgoMessageContactId(contact) {
  return contact && contact.id !== undefined && contact.id !== null ? String(contact.id) : '';
}

function imgoMessageUnread(contact) {
  var unread = Number(contact && contact.unread);
  return Number.isFinite(unread) && unread > 0 ? unread : 0;
}

function imgoMessageCurrentMatches(ui, contactId) {
  if (!ui || typeof ui.getCurrentContact !== 'function') return true;
  var current = ui.getCurrentContact();
  return Boolean(current && String(current.id) === String(contactId));
}

function imgoMessageLatestTime(messages) {
  var latest = 0;
  for (var index = 0; index < messages.length; index += 1) {
    latest = Math.max(latest, Number(messages[index] && messages[index].sendTime) || 0);
  }
  return latest;
}

component.data = function () {
  var state = typeof imgoMessageStateOriginalData === 'function'
    ? imgoMessageStateOriginalData.apply(this, arguments)
    : {};
  state = state || {};
  state.imgoMessagePullSequence = 0;
  state.imgoPendingUnread = {};
  return state;
};

component.methods = Object.assign({}, imgoMessageStateOriginalMethods, {
  imgoRememberContactUnread: function (contact, ui) {
    var contactId = imgoMessageContactId(contact);
    var unread = imgoMessageUnread(contact);
    if (!contactId || unread < 1) return 0;
    this.imgoPendingUnread[contactId] = Math.max(Number(this.imgoPendingUnread[contactId]) || 0, unread);
    if (ui && ui.CacheMessageLoaded && typeof ui.CacheMessageLoaded.remove === 'function') {
      ui.CacheMessageLoaded.remove(contactId);
    }
    return unread;
  },
  imgoPrepareUnreadRefresh: function (contact, ui, unread) {
    if (!ui || typeof ui.getMessages !== 'function' || typeof ui.clearMessages !== 'function') {
      return false;
    }
    var cached = ui.getMessages(contact.id) || [];
    var summaryTime = Number(contact && contact.lastSendTime) || 0;
    var cacheIsStale = summaryTime > imgoMessageLatestTime(cached);
    if (unread < 1 && !cacheIsStale) return false;

    // Lemon IMUI does not emit pull-messages when a conversation already has
    // an in-memory message array (including an empty array). Removing only
    // CacheMessageLoaded therefore leaves a stale conversation visible forever.
    // Drop that array before changeContact finishes so IMUI performs a fresh
    // latest-page request. A newer sidebar timestamp is also sufficient because
    // self-messages from another device do not increment unread.
    var draft;
    var hasDraft = false;
    if (ui.CacheDraft && typeof ui.CacheDraft.get === 'function') {
      draft = ui.CacheDraft.get(contact.id);
      hasDraft = draft !== undefined && draft !== null && draft !== false;
    }
    ui.clearMessages(contact.id);
    if (hasDraft && ui.CacheDraft && typeof ui.CacheDraft.set === 'function') {
      ui.CacheDraft.set(contact.id, draft);
    }
    return true;
  },
  imgoEnsureRealtimeMessageStored: function (message, ui) {
    if (!message || !message.id || !ui || typeof ui.findMessage !== 'function') return false;
    if (ui.findMessage(message.id)) return true;
    if (typeof ui._addMessage !== 'function') return false;
    ui._addMessage(message, message.toContactId, 1);
    if (imgoMessageCurrentMatches(ui, message.toContactId) && typeof ui.updateCurrentMessages === 'function') {
      ui.updateCurrentMessages();
    }
    return true;
  },
  recieveMsg: function (message) {
    var result;
    if (typeof imgoMessageStateOriginalReceive === 'function') {
      result = imgoMessageStateOriginalReceive.apply(this, arguments);
    }
    var ui = this.$refs && this.$refs.IMUI;
    this.imgoEnsureRealtimeMessageStored(message, ui);
    return result;
  },
  imgoRestoreContactUnread: function (contact, ui, unread, totalUnread) {
    var contactId = imgoMessageContactId(contact);
    if (!contactId || unread < 1 || !imgoMessageCurrentMatches(ui, contactId)) return;
    if (ui && typeof ui.updateContact === 'function') ui.updateContact({id: contact.id, unread: unread});
    this.unread = totalUnread;
    if (ui && typeof this.initMenus === 'function') this.initMenus(ui);
  },
  imgoCommitContactUnread: function (contact, ui) {
    var contactId = imgoMessageContactId(contact);
    if (!contactId || !imgoMessageCurrentMatches(ui, contactId)) return;
    var pending = Math.max(
      Number(this.imgoPendingUnread[contactId]) || 0,
      imgoMessageUnread(contact),
      ui && typeof ui.findContact === 'function' ? imgoMessageUnread(ui.findContact(contact.id)) : 0
    );
    if (ui && typeof ui.updateContact === 'function') ui.updateContact({id: contact.id, unread: 0});
    contact.unread = 0;
    if (Number(contact && contact.is_notice) === 1 && pending > 0) {
      this.unread = Math.max(0, (Number(this.unread) || 0) - pending);
    }
    delete this.imgoPendingUnread[contactId];
    if (ui && typeof this.initMenus === 'function') this.initMenus(ui);
  },
  imgoResetMessagePull: function (ui, contactId) {
    if (ui && ui.CacheMessageLoaded && typeof ui.CacheMessageLoaded.remove === 'function') {
      ui.CacheMessageLoaded.remove(contactId);
    }
    if (ui && ui.$refs && ui.$refs.messages && typeof ui.$refs.messages.resetLoadState === 'function') {
      ui.$refs.messages.resetLoadState();
    }
    if (ui) ui._changeContactLock = false;
  },
  handleChangeContact: function (contact, ui) {
    var unread = this.imgoRememberContactUnread(contact, ui);
    var totalUnread = Number(this.unread) || 0;
    this.imgoPrepareUnreadRefresh(contact, ui, unread);
    var result;
    try {
      if (typeof imgoMessageStateOriginalChangeContact === 'function') {
        result = imgoMessageStateOriginalChangeContact.apply(this, arguments);
      }
    } finally {
      this.imgoRestoreContactUnread(contact, ui, unread, totalUnread);
    }
    return result;
  },
  handlePullMessages: function (contact, done, ui) {
    var contactId = imgoMessageContactId(contact);
    if (!contactId || !ui || typeof done !== 'function') return false;
    var cached = typeof ui.getMessages === 'function' ? (ui.getMessages(contact.id) || []) : [];
    var request = Object.assign({}, this.params || {});
    request.page = cached.length > 0 ? 1 : (Number(request.page) || 1);
    request.last_id = cached.length > 0 ? (Number(cached[0].msg_id) || 0) : 0;
    request.toContactId = contact.id;
    request.is_group = contact.is_group;
    var limit = Number(request.limit) || 20;
    var sequence = (Number(this.imgoMessagePullSequence) || 0) + 1;
    this.imgoMessagePullSequence = sequence;
    var self = this;
    var attempt = 0;

    function currentRequest() {
      return sequence === self.imgoMessagePullSequence && imgoMessageCurrentMatches(ui, contactId);
    }

    function load() {
      return Promise.resolve().then(function () {
        return self.$api.imApi.getMessageListAPI(Object.assign({}, request));
      }).then(function (result) {
        if (!result || Number(result.code) !== 0 || !Array.isArray(result.data)) {
          throw new Error(result && result.msg ? result.msg : '消息数据无效');
        }
        if (!currentRequest()) return false;
        self.params.page = request.page + 1;
        self.params.last_id = request.last_id;
        self.params.toContactId = request.toContactId;
        self.params.is_group = request.is_group;
        done(result.data, result.data.length < limit);
        self.imgoCommitContactUnread(contact, ui);
        return true;
      }).catch(function (error) {
        if (!currentRequest()) return false;
        if (attempt < 1) {
          attempt += 1;
          return new Promise(function (resolve) {
            window.setTimeout(resolve, 300);
          }).then(load);
        }
        self.imgoResetMessagePull(ui, contactId);
        if (self.$message && typeof self.$message.error === 'function') {
          self.$message.error((error && error.message) || '消息加载失败，请重试');
        }
        return false;
      });
    }

    this._imgoLastMessagePull = load();
    return true;
  }
});
/* IMGO_CHAT_MESSAGE_STATE_END */
