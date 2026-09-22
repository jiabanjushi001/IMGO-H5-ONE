const test = require('node:test')
const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { resolve } = require('node:path')
const vm = require('node:vm')

function harness() {
  const toasts = [], saved = []
  let options, hidden = 0
  const source = readFileSync(resolve(__dirname, '../pages/mine/profile.vue'), 'utf8')
    .match(/<script>([\s\S]*?)<\/script>/)[1]
    .replace(/^\s*import .*$/gm, '')
    .replace(/const loginStore = useloginStore\(pinia\)/, '')
    .replace('export default', 'component =')
  const context = {
    avatar: {}, loginStore: { login(data) { saved.push(data) } },
    uni: { showLoading() {}, hideLoading() { hidden++ },
      showToast(toast) { toasts.push(toast.title) },
      getStorageSync() { return 'test-token' }, uploadFile(value) { options = value } }
  }
  vm.runInNewContext(source, context)
  const userInfo = { avatar: '/old.png', account: 'test' }
  context.component.methods.uploadAvatar.call({ userInfo, $api: { msgApi: { uploadAvatar: '/upload' } } }, { path: '/test.png' })
  return { userInfo, saved, toasts, options, hidden: () => hidden }
}

for (const msg of ['', '   ', undefined]) {
  test(`success uses a nonempty fallback for msg=${JSON.stringify(msg)}`, () => {
    const h = harness()
    h.options.success({ statusCode: 200, data: JSON.stringify({ code: 0, data: '/new.png', msg }) })
    assert.deepEqual(h.toasts, ['头像更换成功'])
    assert.equal(h.userInfo.avatar, '/new.png')
    assert.equal(h.saved[0].avatar, '/new.png')
    assert.equal(h.hidden(), 1)
  })
}
test('preserves a server success message and string success code', () => {
  const h = harness()
  h.options.success({ statusCode: 200, data: { code: '0', data: '/new.png', msg: ' 上传成功 ' } })
  assert.deepEqual(h.toasts, ['上传成功'])
})
for (const response of [
  { statusCode: 500, data: '{}' },
  { statusCode: 200, data: '<html>error</html>' },
  { statusCode: 200, data: 'null' },
  { statusCode: 200, data: '{"code":400,"msg":""}' },
  { statusCode: 200, data: '{"code":0,"data":[]}' },
  { statusCode: 200, data: '{"code":0,"data":" "}' },
]) {
  test(`invalid upload does not replace avatar: ${JSON.stringify(response)}`, () => {
    const h = harness()
    h.options.success(response)
    assert.equal(h.toasts.length, 1)
    assert.ok(h.toasts[0].length > 0)
    assert.equal(h.userInfo.avatar, '/old.png')
    assert.equal(h.saved.length, 0)
    assert.equal(h.hidden(), 1)
  })
}
test('business errors retain their message', () => {
  const h = harness()
  h.options.success({ statusCode: 200, data: '{"code":400,"msg":"图片过大"}' })
  assert.deepEqual(h.toasts, ['图片过大'])
})
test('network failure closes loading and shows feedback', () => {
  const h = harness()
  h.options.fail({ errMsg: 'network error' })
  assert.deepEqual(h.toasts, ['头像上传失败，请检查网络后重试'])
  assert.equal(h.hidden(), 1)
  assert.equal(h.saved.length, 0)
})
