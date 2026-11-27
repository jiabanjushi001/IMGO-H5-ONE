const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')

const root = path.resolve(__dirname, '..')
const source = fs.readFileSync(path.join(root, 'frontend/withdrawal-notifier.js'), 'utf8')
const spoken = []
const played = []
const messages = []
class SpeechSynthesisUtterance {
  constructor(text) { this.text = text }
}
const window = {
  SpeechSynthesisUtterance,
  speechSynthesis: { speak: speech => spoken.push(speech) },
  Audio: class { constructor(src) { this.src = src } play() { played.push(this.src); return Promise.resolve() } }
}
const sandbox = { window }
vm.runInNewContext(`${source}\nthis.install = imgoInstallWithdrawalNotifier`, sandbox)

let listener
const store = {
  _vm: { $message: options => messages.push(options) },
  watch(select, callback) {
    assert.equal(select({ socketAction: 'event' }), 'event')
    listener = callback
  }
}
sandbox.install(store)
sandbox.install(store)
assert.equal(typeof listener, 'function')

listener({ type: 'message', data: {} })
assert.equal(spoken.length, 0)
listener({ type: 'withdrawalSubmitted', time: 1, data: { withdrawal_id: 61, user_id: 7, account: 'member7', amount_cents: 1234 } })
assert.equal(spoken.length, 1)
assert.match(spoken[0].text, /member7/)
assert.match(spoken[0].text, /12\.34元/)
assert.equal(spoken[0].lang, 'zh-CN')
listener({ type: 'withdrawalSubmitted', time: 2, data: { withdrawal_id: 61, user_id: 7, account: 'member7', amount_cents: 1234 } })
assert.equal(spoken.length, 1, 'duplicate socket delivery must not speak twice')
listener({ type: 'paymentMethodSubmitted', time: 3, data: { user_id: 7, account: 'member7', payment_type: 'wechat', payment_label: '微信收款码' } })
assert.equal(spoken.length, 2)
assert.match(spoken[1].text, /微信收款码审核申请/)
assert.equal(messages.length, 2)
assert.match(messages[1].message, /member7/)
assert.equal(played.length, 0, 'speech support should not use fallback audio')
console.log('Withdrawal and payment-method mentor notifier verified')
