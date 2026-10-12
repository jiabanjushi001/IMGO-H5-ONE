const test = require('node:test')
const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { resolve } = require('node:path')

const withdrawPage = readFileSync(resolve(__dirname, '../pages/mine/wallet-withdraw.vue'), 'utf8')
const paymentPage = readFileSync(resolve(__dirname, '../pages/mine/payment-methods.vue'), 'utf8')
const walletApi = readFileSync(resolve(__dirname, '../api/wallet.js'), 'utf8')
const walletCard = readFileSync(resolve(__dirname, '../components/wallet/WalletSummaryCard.vue'), 'utf8')

test('withdrawal page enforces server settings and sends selected payment type', () => {
	assert.match(withdrawPage, /minimum_withdrawal_cents/)
	assert.match(withdrawPage, /withdrawal_enabled/)
	assert.match(withdrawPage, /amountCents\.value >= minimumCents\.value/)
	assert.match(withdrawPage, /payment_type: selectedType\.value/)
	assert.match(withdrawPage, /title: '提现成功'/)
	assert.match(walletApi, /payment_type = 'bank'/)
})

test('payment binding page supports bank, Alipay and WeChat QR', () => {
	assert.match(paymentPage, />银行卡</)
	assert.match(paymentPage, />支付宝</)
	assert.match(paymentPage, />微信收款码</)
	assert.match(paymentPage, /payment_type: 'alipay'/)
	assert.match(paymentPage, /payment_type: 'wechat'/)
	assert.match(paymentPage, /qr_file_id: wechat\.value\.qrFileId/)
	assert.match(paymentPage, /canEdit\(method\).*status.*=== 2/)
	assert.match(paymentPage, /已提交，等待审核/)
	assert.match(withdrawPage, /alipay\.status\) === 1/)
	assert.match(withdrawPage, /wechat\.status\) === 1/)
})

test('wallet entry reflects disabled withdrawal state', () => {
	assert.match(walletCard, /!withdrawalEnabled/)
	assert.match(walletCard, /暂停提现/)
})
