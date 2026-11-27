const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')
const { execFileSync } = require('node:child_process')

const root = path.resolve(__dirname, '..')
const read = file => fs.readFileSync(path.join(root, file), 'utf8')
const source = read('frontend/auto-group-existing.js')
const sandbox = {}
vm.runInNewContext(`${source}\nthis.component = ImgoAutoGroupPicker`, sandbox)
const component = sandbox.component

async function exercisePicker() {
  const emitted = []
  const state = Object.assign(component.data(), component.methods, {
    value: { status: 1, mode: 'existing', group_id: 77 },
    $emit(name, value) { emitted.push([name, value]) },
    $api: { groupApi: { getGroupList: async () => ({
      code: 0,
      count: 2,
      data: [
        { group_id: 77, name: '已有群A', owner_id: 7, user_count: 3 },
        { group_id: 88, name: '已有群B', owner_id: 7, user_count: 5 }
      ]
    }) } }
  })
  Object.defineProperty(state, 'mode', { get() { return component.computed.mode.call(state) } })
  Object.defineProperty(state, 'groupID', { get() { return component.computed.groupID.call(state) } })

  await state.loadGroups()
  assert.equal(state.groups.length, 2)
  state.setGroup(88)
  assert.equal(emitted.at(-1)[1].group_id, 88)
  state.setMode('create')
  assert.equal(emitted.at(-1)[1].mode, 'create')
}

exercisePicker().then(() => {
  const mentor = read('frontend/member-agent-setting.js')
  for (const marker of ["mode: 'create'", 'loadGroupOptions', 'this.isSuperOperator ? groups : groups.filter', '加入已有群', '请选择要自动加入的已有群聊', 'group_id: Number(this.autoAddGroup.group_id)']) {
    assert.ok(mentor.includes(marker), `mentor settings missing ${marker}`)
  }

  execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'pipe' })
  const app = read('public/assets/js/app.85372e4e.js')
  const settingsMap = app.match(/789:"([^"]+)"/)
  const memberMap = app.match(/687:"([^"]+)"/)
  assert.ok(settingsMap && memberMap, 'built chunk mapping is missing')
  const settings = read(`public/assets/js/789.${settingsMap[1]}.js`)
  const members = read(`public/assets/js/687.${memberMap[1]}.js`)
  for (const label of ['自动新建群', '加入已有群', '请选择已经创建好的群聊', '达到上限后停止自动加入']) {
    assert.ok(settings.includes(label), `global settings missing ${label}`)
  }
  for (const label of ['自动新建群', '加入已有群', '已有群聊', '选择已经创建好的群聊']) {
    assert.ok(members.includes(label), `mentor settings missing ${label}`)
  }
  console.log('existing auto-group picker and built settings verified')
}).catch(error => {
  console.error(error)
  process.exitCode = 1
})
