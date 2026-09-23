const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')

const root = path.resolve(__dirname, '..')
const read = file => fs.readFileSync(path.join(root, file), 'utf8')

const rolePanel = read('frontend/role-panel.js')
const menu = read('frontend/rbac-menu.js')
const memberRole = read('frontend/member-role-select.js')
const memberFilter = read('frontend/member-referral-filter.js')
const memberActions = read('frontend/member-actions.js')
assert.ok(fs.existsSync(path.join(root, 'frontend/member-agent-setting.js')), 'mentor settings dialog component missing')
const agentDialog = read('frontend/member-agent-setting.js')
const build = read('scripts/build-maintenance.cjs')
const builtMembers = read('public/assets/js/687.70d7eca3.js')
const builtApp = read('public/assets/js/app.85372e4e.js')

for (const key of [
  'manage.overview', 'manage.settings', 'manage.users', 'manage.messages',
  'manage.groups', 'manage.files', 'manage.bank', 'manage.finance'
]) assert.ok(menu.includes(key), `missing menu permission ${key}`)

assert.ok(menu.includes("'/chat'"), 'ordinary users must fall back to chat')
assert.ok(rolePanel.includes('ImgoRolePanel'), 'role panel component missing')
assert.ok(rolePanel.includes("h('h1', '角色')"), 'role panel title must be 角色')
assert.ok(rolePanel.includes('Boolean(role.builtin)'), 'built-in roles must be read only')
assert.ok(menu.includes('normal.push(roleRoute)'), 'role menu must be appended last')
assert.ok(memberRole.includes('admin_role_id'), 'member role selector must use admin_role_id')
assert.ok(memberRole.includes('!role.builtin'), 'member selector must exclude built-in role rows')
assert.ok(build.includes('path:"/manage/role"'), 'role route injection missing')
assert.ok(build.includes('ImgoRoleApi'), 'role API injection missing')
assert.ok(build.includes('ImgoMemberRoleSelect'), 'member role selector injection missing')
assert.ok(build.includes('ordinaryManageGuard'), 'ordinary management routes must redirect to chat')
assert.ok(build.includes('legacyRoleForm'), 'legacy member role editor must be removed')
assert.ok(builtMembers.includes("this.$set(this.row, 'admin_role_agent_mode'"), 'built member selector must update agent mode')
assert.equal(builtMembers.split('/* IMGO_MEMBER_ROLE_BEGIN */').length - 1, 1, 'member selector must be injected once')
assert.equal(builtMembers.split('/* IMGO_MEMBER_AGENT_SETTING_BEGIN */').length - 1, 1, 'mentor dialog must be injected once')
assert.ok(builtMembers.includes('t("imgo-member-agent-setting-dialog",{ref:"memberAgentSetting",on:{saved:e.handleChange}})'), 'mentor dialog save must refresh current member list')
assert.equal(builtApp.split('agentSettingApi:ImgoAgentSettingApi').length - 1, 1, 'mentor API registry must be injected once')
assert.ok(builtApp.includes('/manage/agentSetting/detail') && builtApp.includes('/manage/agentSetting/save'), 'mentor API endpoints must be wired')

const roleContext = { window: {} }
vm.runInNewContext(rolePanel + '\nthis.component = ImgoRolePanel', roleContext)
const roleComponent = roleContext.component
const roleState = roleComponent.data()
roleComponent.methods.create.call(roleState)
assert.equal(roleState.form.agent_mode, 1, 'new roles must default to agent mode')
roleComponent.methods.select.call(roleState, {
  role_id: 5, name: '导师专员', remark: '', status: 1, agent_mode: 1,
  role_code: 'mentor', permissions: ['manage.users'], builtin: false
})
assert.equal(roleState.form.agent_mode, 1, 'role editor must retain agent mode')
assert.equal(roleState.form.role_code, 'mentor', 'role editor must retain preset code')
roleState.roles = [{ ...roleState.form, user_count: 0 }]
roleState.permissions = []
const h = (tag, data, children) => ({ tag, data, children })
const renderTree = roleComponent.render.call(roleState, h)
const flatten = node => Array.isArray(node) ? node.flatMap(flatten) : node && typeof node === 'object' ? [node, ...flatten(node.children)] : []
const nodes = flatten(renderTree)
const modeField = nodes.find(node => node.tag === 'el-form-item' && node.data?.props?.label === '代理模式')
assert.ok(modeField, 'role editor must show agent mode switch')
assert.ok(flatten(modeField).some(node => node.tag === 'el-switch' && node.data?.props?.value === 1), 'agent mode switch must reflect saved value')
assert.ok(!nodes.some(node => node.tag === 'el-button' && node.children === '删除角色'), 'mentor preset must not show delete action')
let deleteConfirmed = false
roleComponent.methods.remove.call({ form: roleState.form, $confirm: () => { deleteConfirmed = true } })
assert.equal(deleteConfirmed, false, 'mentor preset deletion must be guarded in component method')

const memberContext = { window: {} }
vm.runInNewContext(memberRole + '\nthis.component = ImgoMemberRoleSelect', memberContext)
const memberComponent = memberContext.component
const selfRoleTree = memberComponent.render.call({
  isSuperOperator: false,
  row: { user_id: 7, is_self: 1 },
  roleLabel: '导师专员'
}, h)
assert.ok(JSON.stringify(selfRoleTree).includes('我自己'), 'mentor own row must be visibly marked')
const filterContext = {}
vm.runInNewContext(memberFilter + '\nthis.component = ImgoMemberReferralFilter', filterContext)
const filterComponent = filterContext.component
for (const [agentMode, expectedClearable] of [[true, false], [false, true]]) {
  const tree = filterComponent.render.call({ scope: agentMode ? 'all' : '', agentMode, $emit() {} }, h)
  const select = flatten(tree).find(node => node.tag === 'el-select')
  assert.equal(select.data.props.clearable, expectedClearable, 'agent filter must keep a selected scope')
}
const chunks = []
vm.runInNewContext(builtMembers, { self: { webpackChunkRaingad_IM: { push: chunk => chunks.push(chunk) } } })
const memberExports = {}
const memberRequire = id => id === 1001 ? { Z: component => ({ exports: component }) } : id === 3822 ? { rn: () => ({}) } : {}
memberRequire.r = () => {}
memberRequire.d = (target, definitions) => { for (const [key, getter] of Object.entries(definitions)) Object.defineProperty(target, key, { get: getter }) }
chunks[0][1][4368]({}, memberExports, memberRequire)
const page = memberExports.default
assert.equal(page.data.call({ $store: { state: { userInfo: { user_id: 7, agent_mode: 1 } } } }).params.referral_scope, 'all', 'agent member page must start with all descendants')
assert.equal(page.data.call({ $store: { state: { userInfo: { user_id: 1, agent_mode: 0 } } } }).params.referral_scope, '', 'global member page must keep blank scope')
const memberRow = { user_id: 7, admin_role_id: 0, admin_role_name: '普通用户', admin_role_agent_mode: 0 }
const memberState = {
  row: memberRow, roles: [{ role_id: 5, name: '导师专员', agent_mode: 1 }], saving: false,
  roleValue: 0, roleLabel: '普通用户',
  $set: (row, key, value) => { row[key] = value },
  $api: { userApi: { setRole: async () => ({ code: 0 }) } },
  $message: { success() {}, error(message) { throw Error(message) } }
}
async function checkMemberRoleMode() {
  await memberComponent.methods.change.call(memberState, 5)
  assert.equal(memberRow.admin_role_agent_mode, 1, 'member role change must update agent mode')
}

function actionNodes(operatorID, operatorAgentMode, rowUserID, rowAgentMode, isSelf) {
  const e = {
    $store: { state: { userInfo: { user_id: operatorID, agent_mode: operatorAgentMode } } }, $refs: { memberAgentSetting: { open() {} } },
    _u: value => value, _v: value => value, _e: () => null,
    openDialogue() {}, handleClick() {}, editUser() {}, editPass() {}
  }
  const s = { row: { user_id: rowUserID, admin_role_agent_mode: rowAgentMode, is_self: isSelf } }
  const t = (tag, data, children) => ({ tag, data: Array.isArray(data) ? undefined : data, children: Array.isArray(data) ? data : children })
  const table = vm.runInNewContext(memberActions, { t, e, s })
  return flatten(table.data.scopedSlots[0].fn(s))
}
assert.ok(actionNodes(1, 0, 7, 1, 0).some(node => node.tag === 'el-dropdown-item' && node.children?.includes('导师设置')), 'super administrator must see mentor settings for mentor rows')
assert.ok(!actionNodes(1, 0, 7, 0, 0).some(node => node.tag === 'el-dropdown-item' && node.children?.includes('导师设置')), 'ordinary rows must hide mentor settings')
const mentorSelfActions = actionNodes(7, undefined, 7, 1, 1).filter(node => node.tag === 'el-dropdown-item')
assert.ok(mentorSelfActions.some(node => node.children?.includes('导师设置')), 'mentor must see settings on own row')
for (const label of ['会话列表', '查看', '编辑', '改密', '修改邀请码']) {
  assert.ok(!mentorSelfActions.some(node => node.children?.includes(label)), `mentor own row must hide ${label}`)
}
assert.ok(!actionNodes(7, 1, 8, 1, 0).some(node => node.tag === 'el-dropdown-item' && node.children?.includes('导师设置')), 'mentor must not edit another mentor settings')

const agentContext = {}
vm.runInNewContext(agentDialog + '\nthis.component = ImgoMemberAgentSettingDialog', agentContext)
const agentComponent = agentContext.component
const agentState = Object.assign(agentComponent.data(), {
  $store: { state: { userInfo: { user_id: 1 } } },
  $message: { success() {}, error(message) { throw Error(message) } },
  $set(row, key, value) { row[key] = value },
  $emit() {},
  $api: {
    agentSettingApi: {
      detail: async () => ({ code: 0, data: { inherit_auto_user: true, inherit_auto_group: true, auto_add_user: {}, auto_add_group: {}, global_auto_add_user: { status: 1, user_ids: [999], welcome: '欢迎' }, global_auto_add_group: { status: 1, owner_uid: 999, userMax: 10, name: '全局群' } } }),
      save: async payload => { agentState.savedPayload = payload; return { code: 0 } }
    },
    userApi: { getUserList: async () => ({ code: 0, data: [
      { user_id: 8, account: 'child', status: 1 },
      { user_id: 9, account: 'disabled', status: '0' },
      { user_id: 10, account: 'active-string', status: '1' }
    ], count: 3 }) }
  }
})
Object.assign(agentState, agentComponent.methods)
for (const [name, method] of Object.entries(agentComponent.methods)) agentState[name] = method.bind(agentState)
Object.defineProperty(agentState, 'isSuperOperator', { get: () => agentComponent.computed.isSuperOperator.call(agentState) })
async function checkAgentDialog() {
  const row = { user_id: 7, account: 'mentor', status: 1, admin_role_agent_mode: 1 }
  await agentState.open(row)
  assert.equal(agentState.visible, true, 'mentor dialog must open')
  assert.deepEqual(Array.from(agentState.options, option => Number(option.user_id)), [7, 8, 10], 'only enabled mentor and descendants may be selected')

  agentState.close()
  const listPayloads = []
  const originalGetUserList = agentState.$api.userApi.getUserList
  agentState.$api.userApi.getUserList = async payload => { listPayloads.push(payload); return originalGetUserList(payload) }
  agentState.$store.state.userInfo = { user_id: 7 }
  await agentState.open({ ...row, is_self: 1 })
  assert.equal(agentState.visible, true, 'mentor must open own settings')
  assert.equal(listPayloads[0].keywords, '', 'mentor own options must load self and all descendants without account filtering')
  agentState.close()
  agentState.$store.state.userInfo = { user_id: 1, agent_mode: 0 }
  agentState.$api.userApi.getUserList = originalGetUserList
  await agentState.open(row)
  assert.deepEqual(Array.from(await agentState.loadOptions({ user_id: 7, account: 'mentor', status: '0' }), option => Number(option.user_id)), [8, 10], 'disabled mentor must not be selectable')
  agentState.applyDetail({ inherit_auto_user: false, inherit_auto_group: false, auto_add_user: { status: 1, user_ids: [8, 9] }, auto_add_group: { status: 1, owner_uid: 9, userMax: 5, name: '导师群' } })
  assert.deepEqual(Array.from(agentState.autoAddUser.user_ids), [8], 'disabled saved customer must be removed from editable override')
  assert.equal(agentState.autoAddGroup.owner_uid, 0, 'disabled saved group owner must be removed from editable override')
  assert.ok(agentState.unavailableNotice, 'removed saved accounts must be explained in the dialog')
  assert.ok(flatten(agentComponent.render.call(agentState, h)).some(node => node.tag === 'el-alert' && node.data?.props?.title === agentState.unavailableNotice), 'unavailable override warning must be visible')
  agentState.applyDetail({ inherit_auto_user: true, inherit_auto_group: true, global_auto_add_user: { status: 1, user_ids: [999], welcome: '欢迎' }, global_auto_add_group: { status: 1, owner_uid: 999, userMax: 10, name: '全局群' } })
  const dialogNodes = flatten(agentComponent.render.call(agentState, h))
  assert.equal(dialogNodes.filter(node => node.tag === 'el-switch' && node.data?.attrs?.['aria-label'] === '继承全局设置').length, 2, 'dialog must show two independent inheritance switches')
  assert.ok(dialogNodes.some(node => node.children === '全局群'), 'inherited group summary must be visible')
  agentState.inheritAutoUser = false
  agentState.inheritAutoGroup = false
  agentState.autoAddUser.status = 0
  agentState.autoAddGroup.status = 0
  await agentState.submit()
  assert.equal(agentState.savedPayload.agent_user_id, 7, 'save must target current mentor')
  assert.equal(agentState.savedPayload.inherit_auto_user, false)
  assert.equal(agentState.savedPayload.auto_add_user.status, 0, 'explicit user disable must be saved')
  assert.equal(agentState.savedPayload.auto_add_group.status, 0, 'explicit group disable must be saved')
  assert.equal(agentState.savedPayload.auto_add_user.user_ids.length, 0, 'global customer outside mentor team must not be submitted as override')
  assert.equal(agentState.savedPayload.auto_add_group.owner_uid, 0, 'global group owner outside mentor team must not be submitted as override')
  assert.equal(row.agent_setting_inherit_auto_user, true, 'saved row must refresh from detail response')
  agentState.close()
  agentState.unavailableNotice = '旧导师的账号提示'
  let firstResolve
  const first = new Promise(resolve => { firstResolve = resolve })
  let detailCalls = 0
  agentState.$api.agentSettingApi.detail = () => { detailCalls++; return detailCalls === 1 ? first : Promise.resolve({ code: 0, data: { inherit_auto_user: false, inherit_auto_group: false, auto_add_user: { status: 0 }, auto_add_group: { status: 0 } } }) }
  const oldOpen = agentState.open({ user_id: 7, account: 'mentor', admin_role_agent_mode: 1 })
  assert.equal(agentState.unavailableNotice, '', 'opening another mentor must clear stale account warning')
  agentState.open({ user_id: 7, account: 'mentor', admin_role_agent_mode: 1 })
  assert.equal(detailCalls, 1, 'duplicate open must not duplicate detail request')
  const newOpen = agentState.open({ user_id: 9, account: 'mentor2', admin_role_agent_mode: 1 })
  await newOpen
  firstResolve({ code: 0, data: { inherit_auto_user: true, inherit_auto_group: true } })
  await oldOpen
  assert.equal(agentState.row.user_id, 9, 'old detail response must not overwrite newly opened mentor')
  assert.equal(agentState.inheritAutoUser, false, 'new mentor settings must remain current')
  agentState.close()
  agentState.$api.agentSettingApi.detail = async () => ({ code: 500, msg: '读取失败' })
  await agentState.open(row)
  assert.equal(agentState.error, '读取失败', 'detail failure must keep an actionable error in the dialog')
  assert.equal(flatten(agentComponent.render.call(agentState, h)).find(node => node.tag === 'el-button' && node.children === '保存').data.props.disabled, true, 'failed load must disable save')
  agentState.$api.agentSettingApi.detail = async () => ({ code: 0, data: { inherit_auto_user: true, inherit_auto_group: true, global_auto_add_user: { status: 0 }, global_auto_add_group: { status: 0 } } })
  await agentState.open(row)
  assert.equal(agentState.error, '', 'retry must clear load error')
  const messages = { success: [], warning: [], error: [] }
  agentState.$message = {
    success: message => messages.success.push(message),
    warning: message => messages.warning.push(message),
    error: message => messages.error.push(message)
  }
  let saved = 0
  agentState.$api.agentSettingApi.save = async () => { saved++; return { code: 0 } }
  agentState.$api.agentSettingApi.detail = async () => ({ code: 500, msg: '回读失败' })
  await agentState.submit()
  assert.equal(saved, 1, 'save must reach server once')
  assert.deepEqual(messages.success, ['导师设置已保存'], 'successful save must be acknowledged before refresh')
  assert.ok(messages.warning.some(message => message.includes('已保存') && message.includes('刷新失败')), 'refresh failure must explain that save succeeded')
  assert.deepEqual(messages.error, [], 'refresh failure must not be reported as save failure')
  assert.equal(agentState.visible, false, 'refresh failure must close stale editor to prevent repeat submit')
}

const context = {}
vm.runInNewContext(menu, context)
const routes = [
  { path: '/manage/index' },
  { path: '/manage/role', meta: { title: '角色' } },
  { path: '/manage/setting' }
]
const superMenu = context.imgoBuildAdminMenu({ user_id: 1, menu_permissions: [] }, routes)
assert.equal(superMenu.at(-1).path, '/manage/role', 'role menu must be last for super administrator')
const ordinaryMenu = context.imgoBuildAdminMenu({ user_id: 7, menu_permissions: [] }, routes)
assert.equal(ordinaryMenu.some(item => item.path === '/manage/role'), false, 'ordinary user must not see role menu')

Promise.all([checkMemberRoleMode(), checkAgentDialog()]).then(() => console.log('RBAC frontend source assertions passed')).catch(error => { console.error(error); process.exitCode = 1 })
