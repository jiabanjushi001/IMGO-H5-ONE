const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

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
assert.ok(rolePanel.includes('角色权限'), 'role panel title missing')
assert.ok(memberRole.includes('admin_role_id'), 'member role selector must use admin_role_id')
assert.ok(build.includes('path:"/manage/role"'), 'role route injection missing')
assert.ok(build.includes('ImgoRoleApi'), 'role API injection missing')
assert.ok(build.includes('ImgoMemberRoleSelect'), 'member role selector injection missing')
assert.ok(build.includes('ordinaryManageGuard'), 'ordinary management routes must redirect to chat')
assert.ok(build.includes('legacyRoleForm'), 'legacy member role editor must be removed')

console.log('RBAC frontend source assertions passed')
