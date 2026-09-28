const assert = require('assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');

const submitCalls = [];
const chatComponent = {
  data() {
    return {
      noSpeakBox: true,
      groupMenu: [
        { text: '设置禁言', visible: () => true },
        { text: '查看资料', visible: () => true }
      ]
    };
  },
  methods: { setNoSpeak() { submitCalls.push(this.currentChat.role); } }
};
vm.runInNewContext(read('frontend/group-mute-chat.js'), { component: chatComponent, Object, Array, Number });

function chatContext(role) {
  const errors = [];
  return {
    currentChat: { role },
    noSpeakBox: true,
    $message: { error: message => errors.push(message) },
    errors
  };
}

const manager = chatContext(2);
const managerState = chatComponent.data.call(manager);
assert.equal(managerState.groupMenu.find(item => item.text === '设置禁言').visible({contact: {role: 3}}), true, '群管理员应看到普通成员禁言菜单');
assert.equal(managerState.groupMenu.find(item => item.text === '查看资料').visible({}), true, '其他菜单不应受影响');
chatComponent.methods.setNoSpeak.call(manager);
assert.equal(manager.noSpeakBox, true, '群管理员的禁言弹窗应保留');
assert.deepEqual(manager.errors, []);
assert.deepEqual(submitCalls, [2], '群管理员应可提交禁言请求');

const member = chatContext(3);
const memberState = chatComponent.data.call(member);
assert.equal(memberState.groupMenu.find(item => item.text === '设置禁言').visible({contact: {role: 3}}), false, '普通成员不应看到成员禁言菜单');
chatComponent.methods.setNoSpeak.call(member);
assert.equal(member.noSpeakBox, false, '普通成员的禁言弹窗应关闭');
assert.deepEqual(member.errors, ['只有群主或群管理员可以设置禁言']);
assert.deepEqual(submitCalls, [2], '普通成员不得提交禁言请求');

const owner = chatContext(1);
const ownerState = chatComponent.data.call(owner);
assert.equal(ownerState.groupMenu.find(item => item.text === '设置禁言').visible({contact: {role: 3}}), true, '群主应保留成员禁言菜单');
chatComponent.methods.setNoSpeak.call(owner);
assert.deepEqual(submitCalls, [2, 1], '群主应可提交禁言请求');

function vnodeText(text) { return { children: [{ text }] }; }
const settingsComponent = {};
const settingsScope = {
  component: settingsComponent,
  J() { return { children: [vnodeText('头像'), vnodeText('群管理'), vnodeText('群成员邀请'), vnodeText('群成员隐私'), vnodeText('群历史消息'), vnodeText('群禁言'), vnodeText('其他操作')] }; },
  Array,
  Number,
  String
};
vm.runInNewContext(read('frontend/group-mute-settings.js'), settingsScope);
const managerRows = settingsScope.J.call({groupInfo: {isJoin: 2}}).children.map(child => child.children[0].text);
const memberRows = settingsScope.J.call({groupInfo: {isJoin: 3}}).children.map(child => child.children[0].text);
const ownerRows = settingsScope.J.call({groupInfo: {isJoin: 1}}).children.map(child => child.children[0].text);
assert.equal(managerRows.includes('群禁言'), true, '群管理员的群设置应显示群禁言');
assert.equal(memberRows.includes('群禁言'), false, '普通成员的群设置不应显示群禁言');
assert.equal(ownerRows.includes('群禁言'), true, '群主的群设置应保留群禁言');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
assert.ok(app.includes('IMGO_GROUP_MUTE_CHAT_BEGIN'), '成员禁言权限适配器必须打包');
assert.ok(app.includes('IMGO_GROUP_MUTE_SETTINGS_BEGIN'), '群禁言设置权限适配器必须打包');
assert.ok(app.includes('只有群主或群管理员可以设置禁言'), '前端必须保留普通成员拦截');

console.log('Group mute permissions tests passed');
