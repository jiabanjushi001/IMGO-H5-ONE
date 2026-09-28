const test = require('node:test')
const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { resolve } = require('node:path')
const vm = require('node:vm')

const source = readFileSync(resolve(__dirname, '../components/wallet/WithdrawalRecordList.vue'), 'utf8')
const script = source.match(/<script setup>([\s\S]*?)<\/script>/)[1]
  .replace(/const props = defineProps\([^\n]+\)/, 'const props = { records: [] }')
  + '\nglobalThis.walletHistory = { statusName }'
const context = {}
vm.runInNewContext(script, context)

test('frozen withdrawal only adds the frozen status wording', () => {
	assert.equal(context.walletHistory.statusName(3), '已冻结')
	assert.doesNotMatch(source, /金额仍处于冻结中/)
	assert.doesNotMatch(source, /冻结时间/)
	assert.doesNotMatch(source, /冻结原因/)
	assert.doesNotMatch(source, /record-status\.is-3/)
})

test('existing withdrawal states keep their original wording', () => {
	assert.equal(context.walletHistory.statusName(0), '待处理')
	assert.equal(context.walletHistory.statusName(1), '已打款')
	assert.equal(context.walletHistory.statusName(2), '已拒绝')
	assert.equal(context.walletHistory.statusName(99), '未知')
	assert.match(source, /<text>处理时间<\/text>/)
	assert.match(source, /备注：/)
})
