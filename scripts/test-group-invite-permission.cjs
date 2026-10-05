const assert = require('assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');

const calls = [];
const component = {
  methods: {
    openAddGroupUser() { calls.push(['open']); },
    manageGroup(userIds, mode) { calls.push(['manage', userIds, mode]); }
  }
};
vm.runInNewContext(read('frontend/group-invite-permission.js'), { component, Object, Number });

function context(role, managerInvite, legacyInvite = 0) {
  const errors = [];
  return {
    currentChat: { role, setting: { manager_invite: managerInvite, invite: legacyInvite } },
    createChatBox: true,
    $message: { error: message => errors.push(message) },
    errors
  };
}

const owner = context(1, 0);
component.methods.openAddGroupUser.call(owner);
assert.deepEqual(calls, [['open']], '群主应始终可以邀请');

const enabledManager = context(2, 1);
component.methods.openAddGroupUser.call(enabledManager);
component.methods.manageGroup.call(enabledManager, [9], 0);
assert.deepEqual(calls, [['open'], ['open'], ['manage', [9], 0]], '启用后群管理员应可以邀请');

const disabledManager = context(2, 0);
component.methods.openAddGroupUser.call(disabledManager);
component.methods.manageGroup.call(disabledManager, [9], 0);
assert.deepEqual(disabledManager.errors, ['只有群主或群管理员可以邀请成员', '只有群主或群管理员可以邀请成员']);
assert.equal(disabledManager.createChatBox, false, '被拒绝后应关闭添加成员弹窗');

const legacyManager = context(2, undefined, 1);
component.methods.openAddGroupUser.call(legacyManager);
assert.deepEqual(legacyManager.errors, ['只有群主或群管理员可以邀请成员'], '旧 invite=1 不能自动开启管理员邀请');

const member = context(3, 1);
component.methods.openAddGroupUser.call(member);
component.methods.manageGroup.call(member, [9], 0);
assert.deepEqual(member.errors, ['只有群主或群管理员可以邀请成员', '只有群主或群管理员可以邀请成员']);
assert.deepEqual(calls, [['open'], ['open'], ['manage', [9], 0]], '普通成员不得触发邀请请求');

component.methods.manageGroup.call(member, [9], 1, '新群');
assert.deepEqual(calls.at(-1), ['manage', [9], 1], '创建群聊流程不应受现有群邀请权限影响');

function vnodeText(text) { return { children: [{ text }] }; }
const settingsComponent = {};
const settingsScope = {
  J() {
    return {
      children: [
        vnodeText('头像'),
        vnodeText('群管理'),
        vnodeText('管理员邀请'),
        vnodeText('群成员隐私'),
        vnodeText('群历史消息'),
        vnodeText('群禁言'),
        vnodeText('其他操作')
      ]
    };
  },
  Array,
  Number,
  String
};
vm.runInNewContext(read('frontend/group-invite-settings.js'), settingsScope);
const ownerRows = settingsScope.J.call({ groupInfo: { isJoin: 1 } }).children.map(child => child.children[0].text);
const managerRows = settingsScope.J.call({ groupInfo: { isJoin: 2 } }).children.map(child => child.children[0].text);
const memberRows = settingsScope.J.call({ groupInfo: { isJoin: 3 } }).children.map(child => child.children[0].text);
assert.equal(ownerRows.includes('管理员邀请'), true, '群主应看到管理员邀请开关');
assert.equal(managerRows.includes('管理员邀请'), false, '群管理员不应看到管理员邀请开关');
assert.equal(memberRows.includes('管理员邀请'), false, '普通成员不应看到管理员邀请开关');
assert.equal(managerRows.includes('群成员隐私'), true, '隐藏邀请开关不能影响其他设置项');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
assert.ok(app.includes('IMGO_GROUP_INVITE_PERMISSION_BEGIN'), '邀请权限适配器必须打包');
assert.ok(app.includes('IMGO_GROUP_INVITE_SETTINGS_BEGIN'), '仅群主可见的邀请设置适配器必须打包');
assert.ok(app.includes('1==i.role||2==i.role&&1==i.setting.manager_invite'), '添加成员按钮只能对群主或获准的群管理员显示');
assert.ok(!app.includes('i.role<3||1==i.setting.invite'), '旧的普通成员邀请条件必须移除');
assert.ok(app.includes('value:t.setting.manager_invite'), '管理员邀请开关必须绑定独立的新字段');
assert.ok(app.includes('管理员邀请：'), '设置项标题应显示管理员邀请');
assert.ok(app.includes('允许管理员邀请'), '设置说明应显示允许管理员邀请');
assert.ok(app.includes('群主始终可以邀请'), '设置说明应明确群主权限');

console.log('Group invite permission tests passed');
