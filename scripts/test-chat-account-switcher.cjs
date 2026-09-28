const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');

const storage = new Map();
const cookies = new Map([
  ['authToken', 'token-one'],
  ['sessionId', 'session-one'],
  ['UserInfo', { user_id: 1, account: 'alpha', realname: '甲账号', avatar: '' }]
]);
const component = {
  components: {},
  mounted() {
    this.legacyMounted = true;
  }
};
const context = {
  component,
  Date,
  JSON,
  o: () => ({
    get: key => cookies.get(key),
    set: (key, value) => cookies.set(key, value),
    rm: key => cookies.delete(key)
  }),
  window: {
    localStorage: {
      getItem: key => storage.get(key) || null,
      setItem: (key, value) => storage.set(key, value)
    },
    location: { origin: 'http://127.0.0.1:8088', pathname: '/', reload() {}, replace() {} },
    setTimeout(fn) { fn(); }
  }
};

vm.runInNewContext(read('frontend/chat-account-switcher.js'), context);
const vault = context.window.ImgoAccountVault;
assert.ok(vault, 'account vault must be exposed to login and logout actions');

const instance = { $store: { state: { userInfo: cookies.get('UserInfo') } } };
component.mounted.call(instance);
assert.strictEqual(instance.legacyMounted, true, 'existing chat mounted hook must be retained');
assert.strictEqual(vault.read().length, 1, 'current account must be captured on chat mount');

vault.save({
  authToken: 'token-two',
  sessionId: 'session-two',
  password: 'must-not-be-persisted',
  userInfo: { user_id: 2, account: 'beta', realname: '乙账号', password: 'also-secret', custom_permission: 'retained' }
});
assert.strictEqual(vault.read().length, 2, 'multiple accounts must be retained');
assert.ok(!storage.get('imgo:chat-accounts:v1').includes('must-not-be-persisted'), 'passwords must never be persisted');
assert.ok(!storage.get('imgo:chat-accounts:v1').includes('also-secret'), 'password fields inside user info must never be persisted');

vault.switchTo(vault.read().find(account => account.user_id === 2));
assert.strictEqual(cookies.get('authToken'), 'token-two', 'switching must replace the active token');
assert.strictEqual(cookies.get('sessionId'), 'session-two', 'switching must replace the active session');
assert.strictEqual(cookies.get('UserInfo').user_id, 2, 'switching must replace cached user info');
assert.strictEqual(cookies.get('UserInfo').custom_permission, 'retained', 'switching must retain non-secret user state');

vault.remove('id:1');
assert.deepStrictEqual(Array.from(vault.read(), item => item.user_id), [2], 'a saved non-current account can be removed');
for (let userId = 3; userId <= 11; userId += 1) {
  vault.save({
    authToken: 'token-' + userId,
    sessionId: 'session-' + userId,
    userInfo: { user_id: userId, account: 'account' + userId, realname: '账号' + userId }
  });
}
assert.strictEqual(vault.read().length, 10, 'ten logged-in accounts must remain available for switching');

const css = read('frontend/chat-account-switcher.css');
assert.ok(css.includes('@media (max-width: 700px)'), 'account switcher needs a mobile layout');
assert.ok(css.includes('align-items: flex-end'), 'mobile dialog must be a bottom sheet');
assert.ok(css.includes('env(safe-area-inset-bottom)'), 'mobile dialog must respect safe-area insets');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
const bundleCss = read('public/assets/css/imgo-maintenance.css');
assert.ok(app.includes('IMGO_CHAT_ACCOUNT_SWITCHER_BEGIN'), 'account switcher module must be bundled');
assert.ok(app.includes('title:"切换账号"'), 'chat menu must contain the switch-account entry');
assert.ok(app.includes('e("imgo-account-switcher",{ref:"ImgoAccountSwitcher"})'), 'chat root must render the account dialog');
assert.ok(app.includes('window.ImgoAccountVault.save({authToken:i.authToken,sessionId:i.sessionId,userInfo:i.userInfo})'), 'successful login must save the new account');
assert.ok(app.includes('window.ImgoAccountVault.removeCurrent()'), 'explicit logout must remove the active saved token');
assert.ok(bundleCss.includes('IMGO_CHAT_ACCOUNT_SWITCHER_CSS_BEGIN'), 'account switcher CSS must be bundled');

console.log('Chat account switcher tests passed');
