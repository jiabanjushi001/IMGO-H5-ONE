const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const values = new Map();
const localStorage = {
  getItem: key => values.has(key) ? values.get(key) : null,
  setItem: (key, value) => values.set(key, String(value)),
  removeItem: key => values.delete(key)
};
const source = read('frontend/member-batch-create.js');
const context = { window: { localStorage }, setTimeout, clearTimeout };
vm.runInNewContext(source + '\nthis.component = ImgoMemberBatchCreateDialog', context);
const component = context.component;

const taskID = '0123456789abcdef0123456789abcdef';
const item = { user_id: 20, account: 'vip_001', password: '123456', realname: '陈子安', sex_label: '男', parent_invite_code: '123456', customer_account: 'service1' };
const messages = { success: [], error: [], info: [] };
const payloads = [];
const emitted = [];
let status = { found: false };

function createState() {
  const state = component.data();
  Object.assign(state, component.methods, {
    $store: { state: { userInfo: { user_id: 7, account: 'mentor', agent_mode: 1 } } },
    $api: { userApi: {
      getUserList: async () => ({ code: 0, data: [
        { user_id: 7, account: 'mentor', realname: '导师', invite_code: '123456', status: 1 },
        { user_id: 8, account: 'member8', realname: '成员八', invite_code: '654321', status: 1 }
      ] }),
      batchAdd: async payload => {
        payloads.push(payload);
        return { code: 0, data: { found: true, task_id: taskID, status: 'queued', total_count: 2, completed_count: 0, success_count: 0, failed_count: 0, percent: 0, items: [] } };
      },
      batchStatus: async () => ({ code: 0, data: status })
    } },
    $message: {
      success: value => messages.success.push(value),
      error: value => messages.error.push(value),
      info: value => messages.info.push(value)
    },
    $emit: value => emitted.push(value)
  });
  Object.defineProperties(state, {
    mentorMode: { get: () => true },
    activeCandidates: { get: () => state.candidates.filter(candidate => Number(candidate.status) === 1) },
    taskActive: { get: () => !!state.task && ['queued', 'running'].includes(state.task.status) },
    taskFinished: { get: () => !!state.task && ['completed', 'failed'].includes(state.task.status) },
    progressPercent: { get: () => state.task ? Number(state.task.percent || 0) : 0 },
    taskStatusText: { get: () => state.task ? ({ queued: '等待开始', running: '正在创建', completed: '创建完成', failed: '创建中断' }[state.task.status]) : '' }
  });
  state.startPolling = () => {};
  state.schedulePoll = () => {};
  return state;
}

(async () => {
  const state = createState();
  await component.methods.open.call(state);
  assert.equal(state.visible, true, 'dialog must open');
  assert.equal(state.form.parent_invite_code, '123456', 'mentor must default to own invite code');
  assert.equal(state.inviterOptions.length, 2, 'mentor may choose an inviter from their team');

  state.form.count = 2;
  state.form.account_prefix = 'vip_';
  state.form.password = '123456';
  assert.equal(component.computed.accountPrefixHint.call(state), '将按 vip_001、vip_002… 的顺序生成；如已有同前缀账号，会自动接续下一编号');
  await component.methods.submit.call(state);
  assert.equal(payloads.length, 1, 'batch API must be called once');
  assert.equal(state.task.status, 'queued', 'submission must switch to progress state');
  assert.equal(localStorage.getItem(state.storageKey()), taskID, 'task id must survive a page refresh');

  status = { found: true, task_id: taskID, status: 'completed', total_count: 2, completed_count: 2, success_count: 2, failed_count: 0, percent: 100, items: [item, Object.assign({}, item, { user_id: 21, account: 'vip_002' })] };
  await component.methods.refreshTask.call(state, true);
  assert.equal(state.task.status, 'completed', 'polling must reach completed state');
  assert.equal(state.results.length, 2, 'completed accounts must be retained for copy');
  assert.ok(emitted.includes('saved'), 'member list must refresh after completion');

  const reloaded = createState();
  assert.equal(component.mounted, undefined, 'member page must not automatically reopen the batch dialog');
  assert.equal(reloaded.visible, false, 'returning to the member page must keep the batch dialog closed');
  assert.equal(reloaded.task, null, 'task details are loaded only after the batch button is clicked');
  await component.methods.open.call(reloaded);
  assert.equal(reloaded.visible, true, 'clicking the batch button must open the dialog');
  assert.equal(reloaded.task.task_id, taskID, 'opening the dialog must recover the saved task');
  assert.equal(reloaded.results.length, 2, 'opening the dialog must recover generated account details');
  assert.equal(reloaded.progressPercent, 100, 'restored progress must be complete');

  const copyText = component.methods.resultText.call(reloaded);
  assert.ok(copyText.startsWith('账号\t密码\t姓名\t性别\t上级邀请码\t客服'));
  assert.ok(copyText.includes('vip_001\t123456\t陈子安\t男\t123456\tservice1'));

  execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
  const app = read('public/assets/js/app.85372e4e.js');
  const members = read('public/assets/js/687.70d7eca3.js');
  const css = read('public/assets/css/imgo-maintenance.css');
  assert.ok(app.includes('batchAdd:t=>Ti({url:"/manage/User/batchAdd"'), 'batch creation API must be registered');
  assert.ok(app.includes('batchStatus:t=>Ti({url:"/manage/User/batchStatus"'), 'batch progress API must be registered');
  assert.ok(members.includes('ImgoMemberBatchCreateDialog'), 'batch dialog must be bundled');
  assert.ok(members.includes('自动接续下一编号'), 'batch dialog must explain sequential prefix accounts');
  assert.ok(members.includes('任务已保存到服务器'), 'batch dialog must explain refresh recovery');
  assert.ok(css.includes('.imgo-member-batch-progress'), 'batch progress styles must be bundled');
  console.log('Member batch on-demand progress recovery and bundle verified');
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
