const fs = require('fs');
const vm = require('vm');
const assert = require('assert/strict');
const component = {data() { return {groupInfo: {}, setting: {}}; }, methods: {}};
const scope = {component, J() {return {children: [{old: true}]};}, FormData};
vm.createContext(scope);
vm.runInContext(fs.readFileSync('frontend/group-avatar.js', 'utf8'), scope);
const h = (tag, data, children) => ({tag, data, children});
function context(allowed) {
 const events = [], messages = [], calls = [];
 return Object.assign(component.data(), component.methods, {
  groupInfo: {canEditAvatar: allowed, ownerName: 'Owner'}, contact: {id: 'group-5', avatar: 'old', displayName: 'Test'},
  $createElement: h, $set(object, key, value) {object[key] = value;}, $emit(...args) {events.push(args);},
  $message: {warning: text => messages.push(text), error: text => messages.push(text), success: text => messages.push(text)},
  $api: {imApi: {async sendFileAPI(form) { calls.push('upload'); assert(form.get('file')); return {code: 0, data: {file_id: 9}}; },
   async editGroupAvatarAPI(data) {calls.push('save'); assert.equal(data.group_id, 'group-5'); assert.equal(data.file_id, 9); return {code: 0, data: {avatar: 'new?v=1'}};}}},
  events, messages, calls
 });
}
(async () => {
 for (const allowed of [false, true]) {
  const ctx = context(allowed);
  const row = scope.J.call(ctx).children[0];
  assert.equal(row.children.some(node => node && node.tag === 'el-button'), allowed);
  await ctx.imgoChangeGroupAvatar({target: {value: 'file', files: [new File(['png'], 'avatar.png', {type: 'image/png'})]}});
  assert.equal(ctx.calls.length, allowed ? 2 : 0);
  assert.equal(ctx.contact.avatar, 'old', 'must not mutate contact prop');
  if (allowed) {assert.equal(ctx.groupInfo.avatar, 'new?v=1'); assert.equal(ctx.events[0][0], 'avatarChanged');}
  assert.equal(ctx.imgoAvatarBusy, false);
 }
 const failed = context(true);
 failed.$api.imApi.editGroupAvatarAPI = async () => ({code: 403, msg: '无权操作'});
 await failed.imgoChangeGroupAvatar({target: {value: '', files: [new File(['png'], 'avatar.png', {type: 'image/png'})]}});
 assert.equal(failed.events.length, 0);
 assert.equal(failed.imgoAvatarBusy, false);
 assert.equal(failed.messages[0], '无权操作');
 console.log('Group avatar UI: permissions, upload/save, event, error handling passed');
})().catch(error => {console.error(error); process.exitCode = 1;});
