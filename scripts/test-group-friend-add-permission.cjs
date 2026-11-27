const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

const root = path.resolve(__dirname, '..')
const read = file => fs.readFileSync(path.join(root, file), 'utf8')

const memberActions = read('frontend/member-actions.js')
const build = read('scripts/build-maintenance.cjs')
const app = read('public/assets/js/app.85372e4e.js')
const members = read('public/assets/js/687.70d7eca3.js')

assert.ok(memberActions.includes('command==="groupFriendAdd"'), 'member More menu must handle group friend-add permission')
assert.ok(memberActions.includes('开启群添加'), 'disabled member permission must offer enable action')
assert.ok(memberActions.includes('关闭群添加'), 'enabled member permission must offer disable action')
assert.ok(build.includes('/manage/User/setGroupFriendAdd'), 'member permission API must be wired')
assert.ok(app.includes('group_friend_add'), 'login permission must guard group profile privacy')
assert.ok(app.includes('groupFriendAddPermission'), 'permission changes must update an online account immediately')
assert.ok(members.includes('开启群添加') && members.includes('关闭群添加'), 'built member menu must expose both states')
assert.ok(!memberActions.includes('addGroupUser'), 'group friend-add permission must not grant group invitations')

console.log('group friend-add permission tests passed')
