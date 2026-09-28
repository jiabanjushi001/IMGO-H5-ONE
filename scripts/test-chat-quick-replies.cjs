const assert = require('assert/strict');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'frontend/chat-quick-replies.js'), 'utf8');
const component = {components: {}, methods: {}};
vm.runInNewContext(source, {component, Boolean, Number, String, Array});

const calls = [];
const composer = {
  submitDisabled: true,
  clear() { calls.push('clear'); },
  chatArea: {
    insertText(value) { calls.push(['insertText', value]); }
  },
  _checkSubmitDisabled() {
    calls.push('check');
    this.submitDisabled = false;
  }
};
component.methods.imgoUseQuickReply.call({$refs: {IMUI: {$refs: {editor: composer}}}}, '  常用回复  ');
assert.deepEqual(calls, ['clear', ['insertText', '常用回复'], 'check']);
assert.equal(composer.submitDisabled, false, '选择快捷用语后发送按钮应启用');

let fallback = '';
component.methods.imgoUseQuickReply.call({$refs: {IMUI: {setEditorValue(value) { fallback = value; }}}}, '备用回复');
assert.equal(fallback, '备用回复', '旧编辑器应保留兼容写入路径');

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], {cwd: root, stdio: 'inherit'});
const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8');
assert.ok(app.includes("composer.chatArea.insertText(text)"), '修复逻辑必须打入聊天主包');
assert.ok(app.includes("composer._checkSubmitDisabled()"), '发送状态同步逻辑必须打入聊天主包');

console.log('Chat quick replies tests passed');
