const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const scheduled = [];
function WebSocketMock() {}
WebSocketMock.CONNECTING = 0;
WebSocketMock.OPEN = 1;
WebSocketMock.CLOSING = 2;

const component = {data() { return {}; }, methods: {}};
const context = {
  component,
  WebSocket: WebSocketMock,
  window: {addEventListener() {}, removeEventListener() {}},
  document: {cookie: '', hidden: false, addEventListener() {}, removeEventListener() {}},
  clearInterval() {},
  clearTimeout() {},
  setInterval() { return 1; },
  setTimeout(callback, delay) {
    scheduled.push({callback, delay});
    return scheduled.length;
  }
};
vm.runInNewContext(fs.readFileSync('frontend/socket-reliability.js', 'utf8'), context);

function instance() {
  const target = Object.assign(component.data.call({}), {
    $store: {state: {wsStatus: false}, commit() {}, dispatch() { return Promise.resolve(); }},
    $root: {},
    $api: {commonApi: {bindClientIdAPI() { return Promise.resolve({code: 0}); }}},
    $nextTick(callback) { callback(); },
    getWsUrl() { return 'ws://127.0.0.1/wss'; }
  });
  for (const [name, method] of Object.entries(component.methods)) target[name] = method.bind(target);
  return target;
}

const client = instance();
let sends = 0;
client.websocket = {readyState: 3, send() { sends += 1; }};
assert.equal(client.websocketSend({type: 'ping'}), false);
assert.equal(sends, 0, 'closed socket must never receive send()');
assert.equal(scheduled.at(-1).delay, 0, 'closed send schedules immediate recovery');

client.reconnectTimeOut = null;
client.websocket = {readyState: WebSocketMock.OPEN, send(value) { sends += 1; assert.equal(value, '{"type":"ping"}'); }};
client.is_open_socket = true;
assert.equal(client.websocketSend({type: 'ping'}), true);
assert.equal(sends, 1);

const before = scheduled.length;
for (let index = 0; index < 12; index += 1) {
  client.reconnectTimeOut = null;
  client.imgoScheduleReconnect();
}
const retryDelays = scheduled.slice(before).map(item => item.delay);
assert.equal(retryDelays.length, 12, 'reconnect must not stop after three attempts');
assert.ok(retryDelays.every(delay => delay <= 30000), 'retry delay is capped');

context.WebSocket = Object.assign(function () { throw new Error('bad URL'); }, WebSocketMock);
client.websocket = null;
client.reconnectTimeOut = null;
const constructorBefore = scheduled.length;
assert.doesNotThrow(() => client.initWebSocket());
assert.equal(scheduled.length, constructorBefore + 1, 'constructor failure schedules recovery');

console.log('socket reliability checks passed');
