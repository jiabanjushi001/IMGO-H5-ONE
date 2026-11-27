const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const component = {
  watch: {socketAction() {}},
  methods: {}
};
vm.runInNewContext(fs.readFileSync('frontend/socket-recovery.js', 'utf8'), {component});

function makeUI(messages) {
  const appended = [];
  const updatedContacts = [];
  const invalidatedContacts = [];
  return {
    appended,
    updatedContacts,
    invalidatedContacts,
    lastMessages: [],
    CacheMessageLoaded: {remove(id) { invalidatedContacts.push(String(id)); }},
    getMessages() { return messages; },
    appendMessage(message) { appended.push(message); },
    getContacts() { return [{id: 'chat-1'}]; },
    getCurrentContact() { return {id: 'chat-1', is_group: 0}; },
    updateContact(contact) { updatedContacts.push(contact); },
    appendContact() {},
    lastContentRender(message) { return message.content || ''; }
  };
}

function bindInstance(target) {
  for (const [name, method] of Object.entries(component.methods)) {
    target[name] = method.bind(target);
  }
  return target;
}

async function main() {
  const visible = Array.from({length: 10}, (_, index) => ({
    id: String(91 + index),
    msg_id: 91 + index,
    sendTime: 9100 + index
  }));
  const serverWindow = Array.from({length: 52}, (_, index) => ({
    id: String(51 + index),
    msg_id: 51 + index,
    sendTime: 5100 + index,
    content: 'message-' + String(51 + index)
  }));
  // Deliberately put the two missed live messages out of order. The merge must
  // append only those records and restore chronological order.
  serverWindow.push(serverWindow.shift());
  const ui = makeUI(visible);
  const target = bindInstance({
    $refs: {IMUI: ui},
    $store: {commit() {}},
    $nextTick(callback) { callback(); },
    $api: {imApi: {
      getContactsAPI() { return Promise.resolve({code: 0, data: [{id: 'chat-1'}]}); },
      getMessageListAPI() { return Promise.resolve({code: 0, data: serverWindow}); }
    }},
    initMenus() {},
    imgoCommitContactUnread(contact, recoveryUI) { recoveryUI.updateContact({id: contact.id, unread: 0}); }
  });

  await target.imgoRecoverSocketGap();
  assert.deepEqual(ui.appended.map(message => message.msg_id), [101, 102]);
  assert.ok(ui.appended.every(message => message.toContactId === 'chat-1'));
  assert.equal(ui.appended.some(message => message.msg_id <= 90), false,
    'history records must remain for ordinary pagination');
  assert.deepEqual(ui.updatedContacts.at(-1), {id: 'chat-1', unread: 0},
    'socket recovery must commit read state only after messages load');

  const emptyUI = makeUI([]);
  const recovered = target.imgoAppendRecoveredMessages(emptyUI, {id: 'chat-2'}, [
    {id: '3', msg_id: 3, sendTime: 300},
    {id: '1', msg_id: 1, sendTime: 100},
    {id: '2', msg_id: 2, sendTime: 200}
  ]);
  assert.deepEqual(Array.from(recovered, message => message.msg_id), [1, 2, 3]);
  assert.deepEqual(emptyUI.appended.map(message => message.msg_id), [1, 2, 3]);

  const fallbackUI = makeUI([{id: 'visible', sendTime: 200}]);
  target.imgoAppendRecoveredMessages(fallbackUI, {id: 'chat-3'}, [
    {id: 'older', sendTime: 100},
    {id: 'newer', sendTime: 300}
  ]);
  assert.deepEqual(fallbackUI.appended.map(message => message.id), ['newer']);

  console.log('socket recovery checks passed');
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
