const test = require('node:test')
const assert = require('node:assert/strict')
const { readFileSync } = require('node:fs')
const { resolve } = require('node:path')
const vm = require('node:vm')

function createSocketHarness() {
  const tasks = []
  const timers = new Map()
  const events = []
  let nextTimerId = 1
  let socketErrorHandler

  const uni = {
    connectSocket() {
      const task = {
        readyState: 0,
        onOpen(handler) { this.openHandler = handler },
        onClose(handler) { this.closeHandler = handler },
        onMessage(handler) { this.messageHandler = handler },
        send() {},
        close() {
          this.readyState = 3
          this.closeHandler?.()
        },
        open() {
          this.readyState = 1
          this.openHandler?.()
        },
        fail() {
          this.readyState = 3
          socketErrorHandler?.({ errMsg: 'connection failed' })
          this.closeHandler?.()
        }
      }
      tasks.push(task)
      return task
    },
    onSocketError(handler) { socketErrorHandler = handler },
    onNetworkStatusChange() {},
    offNetworkStatusChange() {},
    $emit(event) { events.push(event) },
    showToast() {}
  }
  const schedule = kind => (callback) => {
    const id = nextTimerId++
    timers.set(id, { kind, callback })
    return id
  }
  const clear = id => timers.delete(id)
  const source = readFileSync(resolve(__dirname, '../common/socket.js'), 'utf8')
    .replace(/^import .*\n/m, '')
    .replace(/export default socketIO\s*$/, 'globalThis.SocketIO = socketIO')
  const context = { uni, api: { wssUrl: 'ws://127.0.0.1:8088/wss' }, console: { info() {}, error() {} },
    setTimeout: schedule('timeout'), clearTimeout: clear,
    setInterval: schedule('interval'), clearInterval: clear }
  vm.runInNewContext(source, context, { filename: 'common/socket.js' })

  return {
    socket: new context.SocketIO({ type: 'ping' }),
    tasks,
    events,
    pendingRetries() {
      return [...timers.values()].filter(timer => timer.kind === 'timeout').length
    },
    runRetry() {
      const [id, timer] = [...timers].find(([, value]) => value.kind === 'timeout') || []
      assert.ok(timer, 'expected a pending retry')
      timers.delete(id)
      timer.callback()
    }
  }
}

test('connecting twice creates one WebSocket task', () => {
  const { socket, tasks } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  socket.connectSocketInit({ type: 'ping' })
  assert.equal(tasks.length, 1)
})

test('a stale closed task can be replaced immediately', () => {
  const { socket, tasks, pendingRetries } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  tasks[0].readyState = 3
  socket.connectSocketInit({ type: 'ping' })
  assert.equal(tasks.length, 2)
  assert.equal(pendingRetries(), 0)
})

test('error and close schedule only one one-shot retry', () => {
  const { socket, tasks, pendingRetries, runRetry } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  tasks[0].fail()
  socket.reconnect()
  assert.equal(pendingRetries(), 1)
  runRetry()
  assert.equal(tasks.length, 2)
  assert.equal(pendingRetries(), 0)
})

test('manual close cancels retry and does not reconnect', () => {
  const { socket, tasks, pendingRetries } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  tasks[0].fail()
  assert.equal(pendingRetries(), 1)
  socket.Close()
  assert.equal(pendingRetries(), 0)
  tasks[0].closeHandler?.()
  assert.equal(tasks.length, 1)
})

test('a successful reconnect clears retry state', () => {
  const { socket, tasks, runRetry, pendingRetries, events } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  tasks[0].fail()
  runRetry()
  tasks[1].open()
  assert.equal(pendingRetries(), 0)
  assert.equal(socket.connectNum, 1)
  assert.equal(socket.is_open_socket, true)
  assert.equal(events.filter(event => event === 'socketStatus').length, 1)
})

test('a late close from an old task cannot stop the replacement connection', () => {
  const { socket, tasks, runRetry } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  tasks[0].fail()
  runRetry()
  tasks[1].open()
  tasks[0].closeHandler?.()
  assert.equal(socket.socketTask, tasks[1])
  assert.equal(socket.is_open_socket, true)
})

test('repeated failures stop retrying after the configured limit', () => {
  const { socket, tasks, pendingRetries, runRetry, events } = createSocketHarness()
  socket.connectSocketInit({ type: 'ping' })
  for (let attempt = 0; attempt < 5; attempt += 1) {
    tasks[attempt].fail()
    if (attempt < 4) runRetry()
  }
  assert.equal(tasks.length, 5)
  assert.equal(pendingRetries(), 0)
  assert.equal(events.filter(event => event === 'connectError').length, 1)
})
