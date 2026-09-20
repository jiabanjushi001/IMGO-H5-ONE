const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const os = require('node:os')
const path = require('node:path')
const { spawnSync } = require('node:child_process')

const root = path.resolve(__dirname, '..')
const moduleFromFile = async file => {
  const source = fs.readFileSync(path.join(root, file), 'utf8')
  return import(`data:text/javascript,${encodeURIComponent(source)}`)
}

test('Yimen bundle preparation gives local resources relative paths and a real config', () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'imgo-yimen-test-'))
  try {
    const input = path.join(temp, 'input')
    const output = path.join(temp, 'output')
    fs.mkdirSync(path.join(input, 'assets'), { recursive: true })
    fs.mkdirSync(path.join(input, 'static'), { recursive: true })
    fs.writeFileSync(path.join(input, 'index.html'), `<!doctype html><html><head>
      <link rel="stylesheet" href="/assets/main.css">
      <!-- IMGO_RUNTIME_CONFIG_BEGIN -->
      <script>window.imgoStartApp = function(entry) {}</script>
      <script src="config.js"></script>
      </head><body><script type="module" src="/assets/index-test.js"></script></body></html>`)
    fs.writeFileSync(path.join(input, 'assets/main.css'),
      'a{background:url(/assets/logo.png)} b{background:url("/static/pic.png")}')
    fs.writeFileSync(path.join(input, 'assets/index-test.js'), 'console.log("/static/image/tabbar/demo.png")')
    fs.writeFileSync(path.join(input, 'static/pic.png'), 'image')
    fs.symlinkSync(path.join(root, 'config.yimen.js'), path.join(input, 'config.js'))
    const result = spawnSync(process.execPath,
      [path.join(root, 'scripts/prepare-yimen-h5.mjs'), input, output], { encoding: 'utf8' })
    assert.equal(result.status, 0, result.stderr)
    const html = fs.readFileSync(path.join(output, 'index.html'), 'utf8')
    assert.match(html, /src="\.\/jsbridge-mini\.js"/)
    assert.match(html, /href="\.\/assets\/main\.css"/)
    assert.match(html, /window\.imgoStartApp\("\.\/assets\/index-test\.js"\)/)
    assert.doesNotMatch(html, /type="module"/)
    assert.equal(fs.lstatSync(path.join(output, 'config.js')).isSymbolicLink(), false)
    assert.equal(fs.readFileSync(path.join(output, 'config.js'), 'utf8'),
      fs.readFileSync(path.join(root, 'config.yimen.js'), 'utf8'))
    const css = fs.readFileSync(path.join(output, 'assets/main.css'), 'utf8')
    assert.match(css, /url\(\.\/logo\.png\)/)
    assert.match(css, /url\("\.\.\/static\/pic\.png"\)/)
    assert.match(fs.readFileSync(path.join(output, 'assets/index-test.js'), 'utf8'),
      /"\.\/static\/image\/tabbar\/demo\.png"/)
    assert.equal(fs.existsSync(path.join(output, 'jsbridge-mini.js')), true)
    const second = spawnSync(process.execPath,
      [path.join(root, 'scripts/prepare-yimen-h5.mjs'), input, output], { encoding: 'utf8' })
    assert.notEqual(second.status, 0)
    assert.match(second.stderr, /输出目录已存在/)
  } finally {
    fs.rmSync(temp, { recursive: true, force: true })
  }
})

test('invite links in the packaged App never expose a local scheme or loopback address', async () => {
  const { inviteShareUrl } = await moduleFromFile('utils/inviteShareUrl.js')
  const code = '123456'
  const location = { protocol: 'fs:', origin: 'fs://www', pathname: '/index.html' }
  assert.equal(inviteShareUrl(code, '', location, true), '')
  assert.equal(inviteShareUrl(code, 'http://127.0.0.1/#/register?inviteCode=123456', location, true), '')
  assert.equal(inviteShareUrl(code, 'https://h5.example.com/#/register?inviteCode=111111', location, true), '')
  assert.equal(inviteShareUrl(code, 'https://h5.example.com/#/register?inviteCode=123456', location, true),
    'https://h5.example.com/#/register?inviteCode=123456')
  assert.equal(inviteShareUrl(code, '', { protocol: 'https:', origin: 'https://h5.example.com', pathname: '/' }, false),
    'https://h5.example.com/#/pages/login/register?inviteCode=123456')
})

test('native scan requests a result rather than auto-opening the QR URL', async () => {
  const previous = global.window
  try {
    let ready = null
    let options = null
    let resultCallback = null
    global.window = {
      imgoPackagedApp: true,
      jsBridge: {
        inApp: true,
        ready(callback) { ready = callback },
        scan(value, callback) { options = value; resultCallback = callback }
      }
    }
    const { startYimenScan } = await moduleFromFile('common/yimenBridge.js')
    const results = []
    assert.equal(startYimenScan(value => results.push(value)), true)
    ready()
    assert.equal(options.needResult, true)
    resultCallback(' https://example.com/scan/g/token ')
    assert.deepEqual(results, ['https://example.com/scan/g/token'])
    global.window.imgoPackagedApp = false
    assert.equal(startYimenScan(value => results.push(value)), false)
  } finally {
    global.window = previous
  }
})
