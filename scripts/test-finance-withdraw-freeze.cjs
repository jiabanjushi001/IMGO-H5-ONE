const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { execFileSync } = require('child_process');

const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'frontend/finance-orders.js'), 'utf8');
const sandbox = {};
vm.runInNewContext(`${source}\nglobalThis.__withdrawal = { ImgoWithdrawalOrders, imgoWithdrawalStatusName };`, sandbox);

async function testFreeze() {
  const calls = [];
  const notices = [];
  const view = {
    detail: { withdrawal_id: 51, remark: '资料待复核' },
    processing: false,
    $confirm: async (...args) => { calls.push(['confirm', ...args]) },
    $api: { walletApi: { freeze: async payload => { calls.push(['freeze', payload]); return { code: 0 } } } },
    $message: { success: value => notices.push(value), error: value => notices.push(value), warning: value => notices.push(value) },
    close() { calls.push(['close']) },
    async refresh() { calls.push(['refresh']) }
  };
  await sandbox.__withdrawal.ImgoWithdrawalOrders.methods.freeze.call(view, true);
  assert.strictEqual(JSON.stringify(calls.find(item => item[0] === 'freeze')[1]), JSON.stringify({ withdrawal_id: 51, frozen: 1, remark: '资料待复核' }));
  assert(notices.includes('提现订单已冻结'));
}

async function testUnfreezeClearsReason() {
  let payload;
  const view = {
    detail: { withdrawal_id: 51, remark: '原冻结原因' }, processing: false,
    $confirm: async () => {},
    $api: { walletApi: { freeze: async value => { payload = value; return { code: 0 } } } },
    $message: { success() {}, error() {}, warning() {} }, close() {}, async refresh() {}
  };
  await sandbox.__withdrawal.ImgoWithdrawalOrders.methods.freeze.call(view, false);
  assert.strictEqual(JSON.stringify(payload), JSON.stringify({ withdrawal_id: 51, frozen: 0, remark: '' }));
  assert.strictEqual(sandbox.__withdrawal.imgoWithdrawalStatusName(3), '已冻结');
}

async function main() {
  await testFreeze();
  await testUnfreezeClearsReason();
  execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'pipe' });
  const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8');
  const finance = fs.readFileSync(path.join(root, 'public/assets/js/585.dea7864f.js'), 'utf8');
  assert(app.includes('/manage/wallet/freeze'));
  assert(finance.includes('冻结订单'));
  assert(finance.includes('解除冻结'));
  console.log('finance withdrawal freeze checks passed');
}

main().catch(error => { console.error(error); process.exitCode = 1; });
