const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');

const mobileChatPath = path.join(root, 'frontend/mobile-chat.js');
const mobileCssPath = path.join(root, 'frontend/mobile-compat.css');

assert.ok(fs.existsSync(mobileChatPath), 'frontend/mobile-chat.js is required');
assert.ok(fs.existsSync(mobileCssPath), 'frontend/mobile-compat.css is required');

let changedContact = null;
let mountedCalls = 0;
let destroyedCalls = 0;
let mediaListener = null;
const component = {
  data() {
    return { legacyState: true };
  },
  methods: {
    openChat(contactId, imui) {
      return imui.changeContact(contactId);
    },
    handleChangeContact(contact) {
      changedContact = contact;
      if (contact.shouldThrow) throw new Error('legacy contact failure');
      return 'legacy-result';
    }
  },
  mounted() {
    mountedCalls += 1;
  },
  beforeDestroy() {
    destroyedCalls += 1;
  }
};
const mediaQuery = {
  matches: true,
  addEventListener(type, listener) {
    assert.strictEqual(type, 'change');
    mediaListener = listener;
  },
  removeEventListener(type, listener) {
    assert.strictEqual(type, 'change');
    assert.strictEqual(listener, mediaListener);
  }
};

vm.runInNewContext(read('frontend/mobile-chat.js'), {
  component,
  window: { matchMedia: () => mediaQuery }
});

const state = component.data();
assert.strictEqual(state.legacyState, true, 'legacy data must be retained');
assert.strictEqual(state.imgoMobileConversation, false, 'mobile chat starts on the contact list');

const instance = Object.assign({}, state);
const changeCalls = [];
component.methods.openChat.call(instance, 27, {
  changeContact(contactId, menu) {
    changeCalls.push([contactId, menu]);
    return 'change-result';
  }
});
assert.deepStrictEqual(changeCalls, [[27, 'messages']], 'opening a contact from the address book must switch to the messages menu');

changedContact = null;
const addressBookCalls = [];
component.methods.handleChangeContact.call(instance, { id: 42 }, {
  activeSidebar: 'contacts',
  currentContactIdSidebarContact: 27,
  changeContact(contactId, menu) {
    addressBookCalls.push([contactId, menu]);
    return 'address-book-result';
  }
});
assert.deepStrictEqual(addressBookCalls, [[27, 'messages']], 'address-book selection must open the selected contact in messages');
assert.strictEqual(changedContact, null, 'the stale message contact must not be handled during address-book redirection');
assert.strictEqual(instance.imgoMobileConversation, true, 'address-book selection must enter the mobile conversation');
component.methods.imgoMobileBack.call(instance);

const result = component.methods.handleChangeContact.call(instance, { id: 42 });
assert.strictEqual(result, 'legacy-result', 'legacy contact selection result must be retained');
assert.deepStrictEqual(changedContact, { id: 42 }, 'legacy contact selection must still run');
assert.strictEqual(instance.imgoMobileConversation, true, 'selecting a contact opens the conversation on mobile');
component.methods.imgoMobileBack.call(instance);
assert.strictEqual(instance.imgoMobileConversation, false, 'back returns to the contact list');
assert.throws(
  () => component.methods.handleChangeContact.call(instance, { id: 27, shouldThrow: true }),
  /legacy contact failure/
);
assert.strictEqual(instance.imgoMobileConversation, true, 'mobile conversation opens even when legacy contact setup fails');
component.methods.imgoMobileBack.call(instance);

component.mounted.call(instance);
assert.strictEqual(mountedCalls, 1, 'legacy mounted hook must be retained');
assert.strictEqual(typeof mediaListener, 'function', 'viewport changes must be observed');
mediaQuery.matches = false;
instance.imgoMobileConversation = true;
mediaListener(mediaQuery);
assert.strictEqual(instance.imgoMobileConversation, false, 'desktop mode clears mobile-only navigation state');
component.beforeDestroy.call(instance);
assert.strictEqual(destroyedCalls, 1, 'legacy beforeDestroy hook must be retained');

const css = read('frontend/mobile-compat.css');
for (const selector of [
  '@media (max-width: 760px)',
  '.imgo-admin .main-aside',
  '.el-submenu.imgo-mobile-submenu-open > .el-menu--vertical',
  '.el-submenu.imgo-mobile-submenu-open > .el-menu--vertical > .el-menu--popup',
  '.el-submenu.imgo-mobile-submenu-open > .el-menu--inline',
  '.el-menu--inline::before',
  'display: grid !important',
  '.imgo-admin .el-table__fixed',
  '.imgo-admin .group-user-box',
  '.imgo-mobile-conversation .lemon-container',
  '.imgo-mobile-chat-back',
  '.lemon-container:has(.setting-switch)',
  '.lemon-container:has(.setting-switch) .user-center',
  '.lemon-container:has(.setting-switch) .el-form-item__content',
  '.lemon-wrapper:has(> .lemon-container:has(.setting-switch):not([style*="display: none"]))',
  '.lemon-container:has(.file-header)',
  '.lemon-wrapper:has(> .lemon-container:has(.file-header):not([style*="display: none"]))',
  '.lemon-container:has(.file-header) .file-header-search',
  '.lemon-container:has(.file-header) .file-header + .el-container',
  '.lemon-menu__item[title="全屏/窗口"]',
  '.lemon-menu__item[title="客户端下载"]',
  'max-height: none !important',
  '.sideMenu-message .lemon-wrapper'
]) {
  assert.ok(css.includes(selector), `mobile CSS is missing ${selector}`);
}

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], {
  cwd: root,
  stdio: 'inherit'
});

const app = read('public/assets/js/app.85372e4e.js');
const bundleCss = read('public/assets/css/imgo-maintenance.css');
assert.ok(app.includes('imgo-mobile-conversation'), 'chat root must expose mobile conversation state');
assert.ok(app.includes('imgo-mobile-chat-back'), 'chat title must include a mobile back button');
assert.ok(app.includes('IMGO_MOBILE_CHAT_BEGIN'), 'mobile chat adapter must be bundled');
assert.ok(app.includes('mobileSubmenuPath'), 'management shell must track the open mobile submenu');
assert.ok(app.includes('imgo-mobile-submenu-open'), 'management submenu must expose its mobile open state');
assert.ok(app.includes('handleMobileSubmenu'), 'management submenu must handle touch activation');
assert.ok(app.includes('removeEventListener("click",this.closeMobileSubmenu)'), 'management shell must clean up its outside-click listener');
assert.ok(
  app.includes('this.activeSidebar==ke&&this.changeMenu(we),this.currentContactId=e,this.currentContactId'),
  'address-book contacts must switch to the messages channel before emitting the selected contact'
);
assert.ok(
  app.includes('t.chatArea.checkboxElm?t.chatArea.revisePCCheckDialogLabel('),
  'desktop mention-dialog labels must only be changed when the desktop dialog exists'
);
assert.ok(
  app.includes(':t.chatArea.dialogH5Elm&&t.chatArea.reviseH5DialogLabel('),
  'mobile mention-dialog labels must use the H5 dialog API'
);
assert.ok(bundleCss.includes('IMGO_MOBILE_COMPAT_BEGIN'), 'mobile compatibility CSS must be bundled');
assert.ok(bundleCss.includes('.imgo-mobile-conversation .lemon-container'), 'mobile conversation layout must reach the built CSS');

console.log('Mobile admin and chat compatibility tests passed');
