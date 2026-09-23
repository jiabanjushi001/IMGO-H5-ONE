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

console.log('RBAC frontend source assertions passed')
