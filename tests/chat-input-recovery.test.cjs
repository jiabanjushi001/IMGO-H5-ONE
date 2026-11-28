const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

const root = path.resolve(__dirname, '..')
const input = fs.readFileSync(path.join(root, 'components/message/im-input.vue'), 'utf8')

test('聊天输入框从语音模式返回时重新挂载并恢复焦点', () => {
	assert.match(input, /v-if="recShow==false"/)
	assert.match(input, /shouldFocusEditor=true/)
	assert.match(input, /setTimeout\(\(\)=>this\.focusEditorDom\(\),30\)/)
})

test('表情插入失败或回调丢失时解除只读状态', () => {
	assert.match(input, /fail:\s*releaseReadOnly/)
	assert.match(input, /editorRestoreTimer=setTimeout\(releaseReadOnly,1200\)/)
	assert.match(input, /this\.readOnly=false/)
})

test('玩家点击输入框时主动恢复可编辑状态', () => {
	assert.match(input, /@tap="activateEditor"/)
	assert.match(input, /setAttribute\('contenteditable','true'\)/)
})
