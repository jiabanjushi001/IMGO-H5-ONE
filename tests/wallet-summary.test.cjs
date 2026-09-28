const test = require('node:test')
const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { resolve } = require('node:path')

const source = readFileSync(resolve(__dirname, '../components/wallet/WalletSummaryCard.vue'), 'utf8')
const mineSource = readFileSync(resolve(__dirname, '../pages/mine/index.vue'), 'utf8')

test('wallet balance adds only system-freeze amount and shows pending separately', () => {
	assert.match(source, /<text class="wallet-title">钱包余额<\/text>/)
	assert.match(source, /props\.availableCents \+ props\.frozenCents/)
	assert.match(source, /<text class="wallet-caption">余额<\/text>/)
	assert.match(source, /待处理金额：\{\{ pendingText \}\}/)
	assert.match(source, /wallet-inline-frozen">（已冻结 \{\{ adminFrozenText \}\}）/)
	assert.match(source, /wallet-status-item is-pending/)
	assert.match(source, /\.wallet-status-item\.is-pending \.wallet-status-dot \{ background:#f2b134/)
	assert.match(source, /\.wallet-inline-frozen \{[^}]*color:#e0525a/)
	assert.doesNotMatch(source, /wallet-status-item is-frozen/)
	assert.doesNotMatch(source, /冻结金额：\{\{ adminFrozenText \}\}/)
	assert.doesNotMatch(source, /availableCents \+ props\.pendingCents/)
})

test('mine page maps admin frozen amount independently from pending withdrawals', () => {
	assert.match(mineSource, /:frozen-cents="walletFrozenCents"/)
	assert.match(mineSource, /:pending-cents="walletPendingCents"/)
	assert.match(mineSource, /walletFrozenCents = Number\(res\.data\.frozen_cents\)/)
	assert.match(mineSource, /walletPendingCents = Number\(res\.data\.pending_cents\)/)
})

test('wallet card only shows the compact frozen amount without extra explanations', () => {
	assert.doesNotMatch(source, /提现处理中/)
	assert.doesNotMatch(source, /wallet-balance-tip/)
	assert.doesNotMatch(source, /冻结金额暂不可再次提现/)
})
