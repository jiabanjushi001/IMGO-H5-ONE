const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')
const { execFileSync } = require('node:child_process')

const root = path.resolve(__dirname, '..')
const componentSource = fs.readFileSync(path.join(root, 'frontend/system-alert-settings.js'), 'utf8')
const sandbox = {}
vm.runInNewContext(`${componentSource}\nthis.component = ImgoSystemAlertPanel`, sandbox)
const component = sandbox.component

async function exerciseSettingsBehavior() {
  const calls = []
  const messages = []
  const context = Object.assign(component.data(), component.methods, {
    $api: { configApi: {
      getSystemAlert: async () => ({ code: 0, data: {
        enabled: true,
        bot_token: '123456:secret',
        bot_token_set: true,
        chat_id: '-1007',
        events: [],
        options: [
          { value: 'admin_login', label: '后台：登录', group: '后台操作' },
          { value: 'user_login', label: '用户端：登录', group: '用户端操作' }
        ]
      } }),
      setSystemAlert: async payload => { calls.push(['save', payload]); return { code: 0, data: { enabled: true, bot_token: '123456:secret', bot_token_set: true, chat_id: '-1007', events: ['admin_login'], options: [] } } },
      testSystemAlert: async () => { calls.push(['test']); return { code: 0 } }
    } },
    $confirm: async () => true,
    $message: {
      success: value => messages.push(['success', value]),
      error: value => messages.push(['error', value]),
      warning: value => messages.push(['warning', value])
    }
  })

  await context.load()
  assert.equal(context.form.bot_token, '123456:secret', 'super-admin settings must display the saved bot token')
  const groups = component.computed.groupedOptions.call(context)
  assert.deepEqual(Array.from(groups, group => group.label), ['后台操作', '用户端操作'])
  assert.deepEqual(Array.from(context.form.events), [], 'all alarm choices must remain unchecked by default')
  context.form.events = ['admin_login']
  await context.save()
  assert.equal(calls[0][0], 'save')
  assert.equal(calls[0][1].bot_token, '123456:secret', 'visible token must be saved without being erased')
  assert.deepEqual(Array.from(calls[0][1].events), ['admin_login'])
  await context.testConnection()
  assert.equal(calls[1][0], 'test', 'confirmed test action must call the test endpoint')
  assert.ok(messages.some(item => item[0] === 'success'), 'successful actions must be visible to the operator')
}

exerciseSettingsBehavior().then(() => {
  execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'pipe' })
  const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8')
  for (const endpoint of ['getSystemAlert', 'setSystemAlert', 'testSystemAlert']) {
    assert.ok(app.includes(`${endpoint}:t=>Ti(`), `missing ${endpoint} API adapter`)
  }
  const mappedChunk = app.match(/789:"([^"]+)"/)
  assert.ok(mappedChunk, 'system settings chunk mapping is missing')
  const chunk = fs.readFileSync(path.join(root, `public/assets/js/789.${mappedChunk[1]}.js`), 'utf8')
  for (const label of ['系统报警', 'Telegram Bot Token', '报警内容', '后台操作', '用户端操作', '@机器人用户名 id', '操作人', '操作对象', '操作类型', '金额']) {
    assert.ok(chunk.includes(label), `built settings UI is missing ${label}`)
  }
  console.log('System alert settings behavior and built UI verified')
}).catch(error => {
  console.error(error)
  process.exitCode = 1
})
