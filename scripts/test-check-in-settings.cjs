const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')
const { execFileSync } = require('node:child_process')

const root = path.resolve(__dirname, '..')
const source = fs.readFileSync(path.join(root, 'frontend/check-in-settings.js'), 'utf8')
const sandbox = {}
vm.runInNewContext(`${source}\nthis.component = ImgoCheckInSettingsPanel`, sandbox)
const component = sandbox.component

async function exerciseSettingsBehavior() {
  const calls = []
  const messages = []
  const context = Object.assign(component.data(), component.methods, {
    $api: { configApi: {
      getCheckInSettings: async () => ({ code: 0, data: {
        rewardEnabled: 1, cycleDays: 10, highDays: 3,
        highMinYuan: 7, highMaxYuan: 10, normalMinYuan: 3, normalMaxYuan: 5,
        setting_scope: 'mentor', inherited: true
      } }),
      setCheckInSettings: async payload => { calls.push(payload); return { code: 0, data: { ...payload, setting_scope: 'mentor', inherited: false } } }
    } },
    $message: {
      success: value => messages.push(['success', value]),
      error: value => messages.push(['error', value]),
      warning: value => messages.push(['warning', value])
    }
  })

  await context.load()
  assert.equal(context.loaded, true)
  assert.equal(context.form.cycleDays, 10)
  assert.equal(context.form.highMinYuan, 7)
  assert.equal(context.settingScope, 'mentor')
  assert.equal(context.inherited, true)
  context.form.normalMaxYuan = 6
  await context.save()
  assert.equal(calls.length, 1)
  assert.equal(calls[0].normalMaxYuan, 6)
  assert.equal(context.inherited, false)
  assert.ok(messages.some(item => item[0] === 'success'))

  context.form.highMinYuan = 11
  context.form.highMaxYuan = 10
  await context.save()
  assert.equal(calls.length, 1, 'invalid reward ranges must not be saved')
  assert.ok(messages.some(item => item[0] === 'warning'))
}

exerciseSettingsBehavior().then(() => {
  execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'pipe' })
  const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8')
  assert.ok(app.includes('/manage/Config/getCheckInSettings') && app.includes('/manage/Config/setCheckInSettings'), 'scoped check-in settings APIs are missing')
  const mappedChunk = app.match(/789:"([^"]+)"/)
  assert.ok(mappedChunk, 'system settings chunk mapping is missing')
  const chunk = fs.readFileSync(path.join(root, `public/assets/js/789.${mappedChunk[1]}.js`), 'utf8')
  for (const label of ['签到设置', '启用签到奖励', '奖励周期', '高奖励天数', '保存签到设置', '独立签到规则']) {
    assert.ok(chunk.includes(label), `built settings UI is missing ${label}`)
  }
  const css = fs.readFileSync(path.join(root, 'public/assets/css/imgo-maintenance.css'), 'utf8')
  assert.ok(css.includes('.imgo-checkin-settings'), 'check-in settings styles are missing')
  const memberChunkMap = app.match(/687:"([^"]+)"/)
  assert.ok(memberChunkMap, 'member page chunk mapping is missing')
  const memberChunk = fs.readFileSync(path.join(root, `public/assets/js/687.${memberChunkMap[1]}.js`), 'utf8')
  for (const label of ['每日签到奖励详情', '周期天数', '到账奖励', '余额', '今日签到奖励', '累计签到奖励', '每日奖励详情']) {
    assert.ok(memberChunk.includes(label), `built check-in history is missing ${label}`)
  }
  console.log('Check-in settings behavior and built UI verified')
}).catch(error => {
  console.error(error)
  process.exitCode = 1
})
