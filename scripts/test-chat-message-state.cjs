const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const component = {
  data() { return {params: {page: 1, limit: 20}, unread: 5}; },
  methods: {
    handleChangeContact(contact, ui) {
      ui.updateContact({id: contact.id, unread: 0});
      if (Number(contact.is_notice) === 1) this.unread -= Number(contact.unread) || 0;
      this.currentChat = contact;
    },
    recieveMsg(message) {
      this.$refs.IMUI.appendMessage(message);
    }
  }
};
const context = {
  component,
  Promise,
  Number,
  Array,
  Object,
  String,
  window: {setTimeout(callback) { callback(); }}
};
vm.runInNewContext(fs.readFileSync('frontend/chat-message-state.js', 'utf8'), context);

function makeUI(contact) {
  const updates = [];
  const removed = [];
  const cleared = [];
  let reset = 0;
  let messages = [];
  return {
    updates,
    removed,
    cleared,
    get messages() { return messages; },
    set messages(value) { messages = value; },
    get reset() { return reset; },
    _changeContactLock: true,
    $refs: {messages: {resetLoadState() { reset += 1; }}},
    CacheMessageLoaded: {remove(id) { removed.push(String(id)); }},
    CacheDraft: {
      values: new Map(),
      get(id) { return this.values.get(String(id)); },
      set(id, value) { this.values.set(String(id), value); },
      remove(id) { this.values.delete(String(id)); }
    },
    getCurrentContact() { return contact; },
    getMessages() { return messages; },
    clearMessages(id) {
      cleared.push(String(id));
      messages = [];
      this.CacheMessageLoaded.remove(id);
      this.CacheDraft.remove(id);
    },
    findContact() { return contact; },
    updateContact(value) { updates.push(Object.assign({}, value)); }
  };
}

function bindInstance(overrides) {
  const instance = Object.assign(component.data(), {
    initMenus() {},
    $message: {errors: [], error(message) { this.errors.push(message); }}
  }, overrides || {});
  for (const [name, method] of Object.entries(component.methods)) instance[name] = method.bind(instance);
  return instance;
}

async function main() {
  const contact = {id: 9, unread: 2, is_notice: 1, is_group: 0};
  const ui = makeUI(contact);
  const instance = bindInstance({
    $api: {imApi: {getMessageListAPI() {
      return Promise.resolve({code: 0, data: [{id: 'm1', msg_id: 1}]});
    }}}
  });

  instance.handleChangeContact(contact, ui);
  assert.ok(ui.cleared.includes('9'),
    'an unread conversation must invalidate even an empty in-memory message array');
  assert.equal(instance.unread, 5, 'opening a contact must not clear the global unread count before messages load');
  assert.deepEqual(ui.updates.at(-1), {id: 9, unread: 2}, 'contact unread badge must be restored while loading');
  let callbackResult;
  assert.equal(instance.handlePullMessages(contact, (messages, loaded) => {
    callbackResult = {messages, loaded};
  }, ui), true);
  await instance._imgoLastMessagePull;
  assert.equal(callbackResult.messages.length, 1, 'successful pull must deliver messages');
  assert.equal(instance.unread, 3, 'unread count must clear only after successful pull');
  assert.equal(contact.unread, 0, 'current contact state must be committed as read');
  assert.deepEqual(ui.updates.at(-1), {id: 9, unread: 0});

  const staleContact = {id: 11, unread: 1, is_notice: 1, is_group: 0};
  const staleUI = makeUI(staleContact);
  staleUI.messages = [{id: 'old-10', msg_id: 10, sendTime: 1000}];
  staleUI.CacheDraft.set(11, {editorValue: '尚未发送的草稿'});
  let staleRequest;
  const stale = bindInstance({
    unread: 1,
    $api: {imApi: {getMessageListAPI(request) {
      staleRequest = request;
      return Promise.resolve({code: 0, data: [{id: 'new-11', msg_id: 11, sendTime: 1100}]});
    }}}
  });
  stale.handleChangeContact(staleContact, staleUI);
  assert.equal(staleUI.messages.length, 0,
    'opening a cached conversation with unread messages must invalidate its stale message array');
  assert.deepEqual(staleUI.CacheDraft.get(11), {editorValue: '尚未发送的草稿'},
    'refreshing stale messages must preserve the local draft');
  let staleCallback;
  stale.handlePullMessages(staleContact, (messages, loaded) => {
    staleCallback = {messages, loaded};
  }, staleUI);
  await stale._imgoLastMessagePull;
  assert.equal(staleRequest.last_id, 0,
    'unread refresh must request the latest page instead of messages older than the cache');
  assert.deepEqual(staleCallback.messages.map(message => message.msg_id), [11]);
  assert.equal(stale.unread, 0);

  const summaryContact = {id: 13, unread: 0, is_notice: 1, is_group: 0, lastSendTime: 2000};
  const summaryUI = makeUI(summaryContact);
  summaryUI.messages = [{id: 'summary-old', msg_id: 20, sendTime: 1000}];
  const summary = bindInstance({unread: 0});
  summary.handleChangeContact(summaryContact, summaryUI);
  assert.ok(summaryUI.cleared.includes('13'),
    'a newer sidebar summary must refresh stale messages even without an unread badge');

  const liveContact = {id: 14, unread: 1, is_notice: 1, is_group: 0};
  const liveStored = [];
  const liveUI = {
    appendMessage() {}, // Reproduces Lemon IMUI: an unopened chat updates only its sidebar summary.
    findMessage(id) { return liveStored.find(message => message.id === id); },
    _addMessage(message, contactId) { liveStored.push(Object.assign({}, message, {toContactId: contactId})); },
    getCurrentContact() { return {}; },
    updateCurrentMessages() { throw new Error('inactive contact must not replace the visible message list'); }
  };
  const live = bindInstance({$refs: {IMUI: liveUI}});
  const liveMessage = {id: 'live-21', msg_id: 21, toContactId: 14, sendTime: 2100};
  live.recieveMsg(liveMessage);
  live.recieveMsg(liveMessage);
  assert.deepEqual(liveStored.map(message => message.id), ['live-21'],
    'an unopened conversation must store the realtime body exactly once, not only update its sidebar');

  const historyContact = {id: 12, unread: 0, is_notice: 1, is_group: 0};
  const historyUI = makeUI(historyContact);
  historyUI.messages = [{id: 'history-10', msg_id: 10, sendTime: 1000}];
  let historyRequest;
  const history = bindInstance({
    unread: 0,
    $api: {imApi: {getMessageListAPI(request) {
      historyRequest = request;
      return Promise.resolve({code: 0, data: []});
    }}}
  });
  history.handleChangeContact(historyContact, historyUI);
  assert.equal(historyUI.messages.length, 1,
    'opening a read conversation must retain its cached messages');
  history.handlePullMessages(historyContact, () => {}, historyUI);
  await history._imgoLastMessagePull;
  assert.equal(historyRequest.last_id, 10,
    'ordinary upward pagination must continue from the oldest cached message');

  const failingContact = {id: 10, unread: 1, is_notice: 1, is_group: 0};
  const failingUI = makeUI(failingContact);
  let attempts = 0;
  const failing = bindInstance({
    unread: 1,
    $api: {imApi: {getMessageListAPI() {
      attempts += 1;
      return Promise.reject(new Error('network unavailable'));
    }}}
  });
  failing.handleChangeContact(failingContact, failingUI);
  failing.handlePullMessages(failingContact, () => {
    throw new Error('failed request must not cache an empty result');
  }, failingUI);
  await failing._imgoLastMessagePull;
  assert.equal(attempts, 2, 'a transient message failure must be retried once');
  assert.equal(failing.unread, 1, 'failed pull must preserve unread count');
  assert.equal(failingUI._changeContactLock, false, 'failed pull must release the conversation lock');
  assert.ok(failingUI.removed.includes('10'), 'failed pull must invalidate loaded-message cache');
  assert.equal(failingUI.reset, 1, 'failed pull must reset the scroll loader');

  console.log('chat message state checks passed');
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
