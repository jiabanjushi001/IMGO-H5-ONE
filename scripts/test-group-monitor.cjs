const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const { execFileSync } = require('node:child_process')

const root = path.resolve(__dirname, '..')

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], {
  cwd: root,
  stdio: 'pipe'
})

const app = fs.readFileSync(path.join(root, 'public/assets/js/app.85372e4e.js'), 'utf8')
const mappedChunk = app.match(/173:"([^"]+)"/)
assert.ok(mappedChunk, 'management group chunk mapping is missing')

const groupChunk = fs.readFileSync(
  path.join(root, `public/assets/js/173.${mappedChunk[1]}.js`),
  'utf8'
)

assert.ok(
  groupChunk.includes('e("ChatRecord",{key:t.componentKey,attrs:{contact:t.currentChat,manage:!0}})'),
  'group monitor must load chat history through the management message API'
)

console.log('Group monitor uses management message API')
