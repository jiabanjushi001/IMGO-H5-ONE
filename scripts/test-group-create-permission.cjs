const assert = require('assert/strict');
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = relative => fs.readFileSync(path.join(root, relative), 'utf8');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
const match = app.match(/789:"(imgo[a-f0-9]+)"/);
assert.ok(match, '主包必须引用设置分包');
const settings = read('public/assets/js/789.' + match[1] + '.js');
for (const label of ['建群权限', '仅超管和导师', '所有用户', '普通用户尝试创建群聊时会提示权限不足']) {
  assert.ok(settings.includes(label), '设置分包缺少：' + label);
}
assert.ok(settings.includes('attrs:{label:"建群权限",prop:"groupCreateRole"}'), '建群权限必须始终渲染');
assert.ok(!settings.includes('expression:"chatInfo.groupChat==1"}],attrs:{label:"建群权限"'), '关闭建群时不能隐藏权限设置');
assert.ok(app.includes('return this.$message.error("权限不足")'), '普通用户点击创建群聊时必须提示权限不足');

console.log('Group create permission settings tests passed');
