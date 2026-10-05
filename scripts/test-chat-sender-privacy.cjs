const assert = require('assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');
const context = { window: {} };
vm.createContext(context);
vm.runInContext(read('frontend/config-state.js'), context);
assert.equal(context.window.imgoMessageSenderLabel, context.imgoMessageSenderLabel, '群消息子组件必须能够访问全局格式化函数');

const sender = { account: 'wuhu888', displayName: '小吴' };
assert.equal(context.imgoMessageSenderLabel(sender, { user_id: 1, admin_role_id: 0 }), 'wuhu888(小吴)', '超级管理员应看到账号和昵称');
assert.equal(context.imgoMessageSenderLabel(sender, { user_id: 7, admin_role_id: 3 }), 'wuhu888(小吴)', '管理员应看到账号和昵称');
assert.equal(context.imgoMessageSenderLabel(sender, { user_id: 8, admin_role_id: 0 }), '小吴', '普通用户只能看到昵称');
assert.equal(context.imgoMessageSenderLabel({ account: 'secret', displayName: '' }, { user_id: 8, admin_role_id: 0 }), '', '普通用户不能通过空昵称看到账号');
assert.equal(context.imgoMessageSenderLabel({ account: 'same', displayName: 'same' }, { user_id: 1 }), 'same', '账号与昵称相同时不应重复');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
assert.ok(app.includes('window.imgoMessageSenderLabel(s,t.$store.state.userInfo)'), '身份化群消息名称逻辑必须打入聊天主包');
assert.ok(!app.includes('s.account+"("+s.displayName+")"'), '聊天主包不能再对普通用户直接显示账号');

console.log('Chat sender privacy tests passed');
