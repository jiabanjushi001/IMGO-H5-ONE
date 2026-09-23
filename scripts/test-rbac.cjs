const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const vm = require('node:vm')

const root = path.resolve(__dirname, '..')
const read = file => fs.readFileSync(path.join(root, file), 'utf8')

const rolePanel = read('frontend/role-panel.js')
const menu = read('frontend/rbac-menu.js')
const memberRole = read('frontend/member-role-select.js')
const build = read('scripts/build-maintenance.cjs')
const builtMembers = read('public/assets/js/687.70d7eca3.js')

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

checkMemberRoleMode().then(() => console.log('RBAC frontend source assertions passed')).catch(error => { console.error(error); process.exitCode = 1 })
