const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')

const root = path.resolve(__dirname, '..')
const read = file => fs.readFileSync(path.join(root, file), 'utf8')

test('视频选择后生成并上传真实封面', () => {
	const input = read('components/message/im-input.vue')
	const utility = read('utils/video-cover.js')
	assert.match(input, /createAndUploadVideoPoster\(res\)/)
	assert.match(input, /poster\s*\n?\s*}/)
	assert.match(utility, /drawImage\(video/)
	assert.match(utility, /formData\.append\('file', blob/)
})

test('聊天和记录页不再直接显示固定视频占位图', () => {
	for (const file of ['pages/message/chat.vue', 'pages/message/record.vue']) {
		const source = read(file)
		assert.match(source, /<VideoMessageCover/)
		assert.doesNotMatch(source, /mediaSrc\(item\.extends\.poster\)/)
	}
})

test('固定占位封面回退到视频真实画面', () => {
	const component = read('components/message/VideoMessageCover.vue')
	const utility = read('utils/video-cover.js')
	assert.match(utility, /video-placeholder\\\.svg/)
	assert.match(component, /v-else-if="displayVideo"/)
	assert.match(component, /resolveMediaDisplayUrl\(props\.src\)/)
})
