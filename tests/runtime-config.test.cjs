const test = require('node:test')
const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { resolve } = require('node:path')
const vm = require('node:vm')

function loadRuntimeConfig(windowConfig) {
  const source = readFileSync(resolve(__dirname, '../common/config.js'), 'utf8')
    .replace('export default {', 'globalThis.runtimeConfig = {')
  const context = {
    window: { ...windowConfig },
    URL,
    URLSearchParams,
    process: { env: { NODE_ENV: 'test' } }
  }
  vm.runInNewContext(source, context, { filename: 'common/config.js' })
  return { config: context.runtimeConfig, window: context.window }
}

test('same-origin HTTP API corrects an accidental wss URL to ws', () => {
  const { config } = loadRuntimeConfig({
    apiServers: [{
      httpUrl: 'http://127.0.0.1:8088',
      wsUrl: 'wss://127.0.0.1:8088/wss'
    }]
  })

  assert.equal(config.apiUrl, 'http://127.0.0.1:8088')
  assert.equal(config.wssUrl, 'ws://127.0.0.1:8088/wss')
})

test('same-origin HTTPS API always uses a secure WebSocket', () => {
  const { config } = loadRuntimeConfig({
    apiServers: [{
      httpUrl: 'https://api.example.com',
      wsUrl: 'ws://api.example.com/wss'
    }]
  })

  assert.equal(config.wssUrl, 'wss://api.example.com/wss')
})

test('a dedicated WebSocket gateway keeps its explicitly configured protocol', () => {
  const { config } = loadRuntimeConfig({
    apiServers: [{
      httpUrl: 'http://api.example.com',
      wsUrl: 'wss://socket.example.com/wss'
    }]
  })

  assert.equal(config.wssUrl, 'wss://socket.example.com/wss')
})

test('runtime server changes are normalized before reconnecting', () => {
  const { config, window } = loadRuntimeConfig({
    apiServers: [{
      httpUrl: 'https://api.example.com',
      wsUrl: 'wss://api.example.com/wss'
    }]
  })
  window.httpUrl = 'http://127.0.0.1:8088'
  window.wsUrl = 'wss://127.0.0.1:8088/wss'

  assert.equal(config.getWssUrl(), 'ws://127.0.0.1:8088/wss')
})
