const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')
const { execFileSync } = require('node:child_process')

const root = path.resolve(__dirname, '..')
const source = fs.readFileSync(path.join(root, 'frontend/withdrawal-settings.js'), 'utf8')
const sandbox = {}
vm.runInNewContext(`${source}\nthis.component = ImgoWithdrawalSettingsPanel`, sandbox)
const component = sandbox.component

async function exercise() {
  const calls = []
  const messages = []
  const context = Object.assign(component.data(), component.methods, {
    $api: { configApi: {
      getWithdrawalSettings: async () => ({ code: 0, data: { enabled: 1, minYuan: 100, passwordRequired: 1 } }),
      setWithdrawalSettings: async payload => { calls.push(payload); return { code: 0, data: payload } }
    } },
    $message: {
      success: value => messages.push(['success', value]),
      error: value => messages.push(['error', value]),
      warning: value => messages.push(['warning', value])
    }
  })
  await context.load()
  assert.equal(context.loaded, true)
  assert.equal(context.form.minYuan, 100)
  assert.equal(context.form.passwordRequired, 1)
  context.form.minYuan = 188
  await context.save()
  assert.equal(calls[0].enabled, 1)
  assert.equal(calls[0].minYuan, 188)
  assert.equal(calls[0].passwordRequired, 1)
  assert.ok(messages.some(item => item[0] === 'success'))
  context.form.minYuan = 0
  await context.save()
  assert.equal(calls.length, 1)
}

exercise().then(() => {
  execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'pipe' })
  const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8')
  assert.ok(app.includes('getWithdrawalSettings'))
  assert.ok(app.includes('setWithdrawalSettings'))
  const mapped = app.match(/789:"([^"]+)"/)
  assert.ok(mapped)
  const chunk = fs.readFileSync(path.join(root, `public/assets/js/789.${mapped[1]}.js`), 'utf8')
  for (const label of ['提现设置', '最低提现金额', '提现密码', '必须验证', '保存提现设置']) assert.ok(chunk.includes(label), `missing ${label}`)
  const css = fs.readFileSync(path.join(root, 'public/assets/css/imgo-maintenance.css'), 'utf8')
  assert.ok(css.includes('.imgo-withdrawal-settings'))
  console.log('Withdrawal settings behavior and built UI verified')
}).catch(error => {
  console.error(error)
  process.exitCode = 1
})
