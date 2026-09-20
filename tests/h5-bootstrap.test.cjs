const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const vm = require('node:vm')
const path = require('node:path')

const projectRoot = path.resolve(__dirname, '..')
const sourceHTML = fs.readFileSync(path.join(projectRoot, 'index.html'), 'utf8')
const buildHTML = fs.readFileSync(path.join(projectRoot, 'unpackage/dist/build/h5/index.html'), 'utf8')
const bootstrap = [...sourceHTML.matchAll(/<script>([\s\S]*?)<\/script>/g)]
  .map(match => match[1])
  .find(code => code.includes('window.imgoStartApp ='))

function createHarness(fetchImpl, options = {}) {
  const app = { innerHTML: '', textContent: '' }
  const retry = { onclick: null }
  const scripts = []
  const requests = []
  const intervals = []
  const entries = {
    'imgo-active-server': options.savedUrl || '',
    'imgo-server-list': options.savedSignature || ''
  }
  const storage = { get value() { return entries['imgo-active-server'] } }
  let reloads = 0
  const context = {
    window: { httpUrl: options.apiUrl, apiServers: options.servers },
    location: { hostname: '127.0.0.1', port: '8765', origin: 'http://127.0.0.1:8765', reload() { reloads += 1 } },
    sessionStorage: {
      getItem(key) { return entries[key] || null },
      setItem(key, value) { entries[key] = value }
    },
    document: {
      getElementById(id) {
        return { app, 'imgo-retry': retry }[id]
      },
      createElement() { return {} },
      body: { appendChild(script) { scripts.push(script) } }
    },
    AbortController,
    URL,
    fetch: async (url) => {
      requests.push(url)
      return fetchImpl(url)
    },
    setTimeout,
    clearTimeout,
    setInterval(callback) { intervals.push(callback) },
    console: { warn() {} }
  }
  vm.runInNewContext(bootstrap, context)
  return { context, app, retry, scripts, requests, storage, intervals,
    get reloads() { return reloads } }
}

const healthy = { ok: true, json: async () => ({ code: 0 }) }
const unreachable = () => Promise.reject(new Error('connection refused'))
const settle = () => new Promise(resolve => setImmediate(resolve))

test('missing config.js stops before any API request or app bundle load', async () => {
  const harness = createHarness(() => healthy)
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.match(harness.app.innerHTML, /配置文件缺失或无效/)
  assert.equal(harness.requests.length, 0)
  assert.equal(harness.scripts.length, 0)
  assert.equal(typeof harness.retry.onclick, 'function')
})

test('empty or invalid server list never falls back to a built-in API', async () => {
  for (const configuredServers of [[], [{ httpUrl: 'not-a-url' }]]) {
    const harness = createHarness(() => healthy, { servers: configuredServers })
    harness.context.window.imgoStartApp('/assets/index-test.js')
    await settle()
    assert.match(harness.app.innerHTML, /配置文件缺失或无效/)
    assert.equal(harness.requests.length, 0)
    assert.equal(harness.scripts.length, 0)
  }
})

for (const port of ['8089', '8889']) {
  test(`unreachable API on ${port} shows a responsive offline page without loading the app bundle`, async () => {
    const harness = createHarness(unreachable, { apiUrl: `http://127.0.0.1:${port}` })
    harness.context.window.imgoStartApp('/assets/index-test.js')
    await settle()
    assert.match(harness.app.innerHTML, /暂时无法连接服务器/)
    assert.doesNotMatch(harness.app.innerHTML, /127\.0\.0\.1:\d+/)
    assert.doesNotMatch(harness.app.innerHTML, /imgo-offline-url/)
    assert.equal(typeof harness.retry.onclick, 'function')
    assert.equal(harness.scripts.length, 0)
  })
}

test('healthy API loads the app bundle after the probe', async () => {
  const harness = createHarness(() => healthy, { apiUrl: 'http://127.0.0.1:8088' })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.equal(harness.scripts.length, 1)
  assert.equal(harness.scripts[0].type, 'module')
  assert.equal(harness.scripts[0].src, '/assets/index-test.js')
})

const servers = [
  { httpUrl: 'http://127.0.0.1:8088', wsUrl: 'ws://127.0.0.1:8088/wss' },
  { httpUrl: 'https://backup.example.com', wsUrl: 'wss://socket.example.com/wss' }
]
const signature = list => list.map(server => `${server.httpUrl} ${server.wsUrl}`).join('\n')

test('startup selects the first reachable API and its matching WebSocket', async () => {
  const harness = createHarness(url => url.startsWith(servers[0].httpUrl) ? unreachable() : healthy, { servers })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.deepEqual(harness.requests, [
    `${servers[0].httpUrl}/common/pub/getSystemInfo`,
    `${servers[1].httpUrl}/common/pub/getSystemInfo`
  ])
  assert.equal(harness.context.window.httpUrl, servers[1].httpUrl)
  assert.equal(harness.context.window.wsUrl, servers[1].wsUrl)
  assert.equal(harness.storage.value, servers[1].httpUrl)
  assert.equal(harness.scripts.length, 1)
})

test('saved healthy server is preferred on refresh', async () => {
  const harness = createHarness(() => healthy, {
    servers, savedUrl: servers[1].httpUrl, savedSignature: signature(servers)
  })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.deepEqual(harness.requests, [`${servers[1].httpUrl}/common/pub/getSystemInfo`])
  assert.equal(harness.context.window.httpUrl, servers[1].httpUrl)
})

test('editing server priority starts from the new first server', async () => {
  const reordered = [servers[1], servers[0]]
  const harness = createHarness(() => healthy, {
    servers: reordered, savedUrl: servers[0].httpUrl, savedSignature: signature(servers)
  })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.deepEqual(harness.requests, [`${servers[1].httpUrl}/common/pub/getSystemInfo`])
})

test('a failed runtime health check switches to a reachable backup once', async () => {
  let primaryHealthy = true
  const harness = createHarness(url => url.startsWith(servers[0].httpUrl)
    ? (primaryHealthy ? healthy : unreachable()) : healthy, { servers })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.equal(harness.intervals.length, 1)
  primaryHealthy = false
  harness.intervals[0]()
  await settle()
  assert.equal(harness.storage.value, servers[1].httpUrl)
  assert.equal(harness.reloads, 1)
})

test('runtime keeps the current page when every backup is unreachable', async () => {
  let primaryHealthy = true
  const harness = createHarness(url => url.startsWith(servers[0].httpUrl) && primaryHealthy
    ? healthy : unreachable(), { servers })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  primaryHealthy = false
  harness.intervals[0]()
  await settle()
  assert.equal(harness.reloads, 0)
  assert.equal(harness.storage.value, servers[0].httpUrl)
})

test('missing WebSocket URL is derived from its API URL', async () => {
  const harness = createHarness(() => healthy, { servers: [{ httpUrl: 'https://api.example.com/' }] })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.equal(harness.context.window.httpUrl, 'https://api.example.com')
  assert.equal(harness.context.window.wsUrl, 'wss://api.example.com/wss')
})

test('all configured APIs unreachable show the offline page without loading the bundle', async () => {
  const harness = createHarness(unreachable, { servers })
  harness.context.window.imgoStartApp('/assets/index-test.js')
  await settle()
  assert.equal(harness.requests.length, 2)
  assert.equal(harness.scripts.length, 0)
  assert.match(harness.app.innerHTML, /暂时无法连接服务器/)
})

test('preview build has the offline gate and no eager app module', () => {
  assert.match(buildHTML, /window\.imgoStartApp\("\/assets\/index-[^"/]+\.js"\)/)
  assert.doesNotMatch(buildHTML, /<script type="module"[^>]*src="\/assets\/index-/)
})
