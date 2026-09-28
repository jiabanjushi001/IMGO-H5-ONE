const assert = require('assert');
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');

const css = read('frontend/login-mobile.css');
for (const expected of [
  'IMGO_ADD_ACCOUNT_LOGIN_BEGIN',
  '@media (max-width: 700px)',
  'min-height: 100dvh',
  'env(safe-area-inset-top)',
  'font-size: 16px',
  '@media (max-height: 650px) and (max-width: 700px)'
]) assert.ok(css.includes(expected), `login mobile CSS is missing ${expected}`);

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });

const app = read('public/assets/js/app.85372e4e.js');
const bundleCss = read('public/assets/css/imgo-maintenance.css');
assert.ok(app.includes('class:{"imgo-add-account-login":t.isAddAccount}'), 'add-account login needs a route-specific class');
assert.ok(app.includes('登录其他账号'), 'add-account login needs a clear title');
assert.ok(app.includes('登录成功后会保存到本机账号列表'), 'add-account mode needs a persistence explanation');
assert.ok(app.includes('返回账号列表'), 'add-account mode needs a safe return action');
assert.ok(app.includes('e&&!this.isAddAccount?'), 'add-account mode must not prefill the previous remembered account');
assert.ok(app.includes('autocomplete:t.isAddAccount?"new-password":"current-password"'), 'add-account password must opt out of prior account autofill');
assert.ok(app.includes('name:t.isAddAccount?"imgo-add-account":"username"'), 'add-account username needs an isolated autofill identity');
assert.ok(app.includes('window.ImgoAccountVault.switchTo(t)'), 'return action must restore the previous account');
assert.ok(app.includes('window.ImgoAccountVault.switchTo(t),void window.location.reload()'), 'return action must reload before mounting authenticated chat');
assert.ok(!app.includes('window.location.pathname+"#/chat"'), 'return action must not mount chat before the restored store is reloaded');
assert.ok(bundleCss.includes('IMGO_ADD_ACCOUNT_LOGIN_BEGIN'), 'login mobile CSS must be bundled');

console.log('Add-account mobile login tests passed');
