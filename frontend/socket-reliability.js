// Vue 2 socket adapter. Keep one live connection, reconnect without a retry cap,
// and revalidate stale sockets when a mobile browser resumes from background.
const originalSocketData = component.data;
component.data = function () {
  return Object.assign({}, originalSocketData.call(this), {
    is_open_socket: false,
    websocket: null,
    heartbeatInterval: null,
    reconnectTimeOut: null,
    imgoSocketGeneration: 0,
    imgoReconnectAttempt: 0,
    imgoLastSocketActivity: 0,
    imgoManualSocketClose: false,
    imgoVisibilityHandler: null,
    imgoOnlineHandler: null
  });
};

component.methods.imgoClearSocketTimers = function () {
  clearInterval(this.heartbeatInterval);
  clearTimeout(this.reconnectTimeOut);
  this.heartbeatInterval = null;
  this.reconnectTimeOut = null;
};

component.methods.imgoCurrentToken = function () {
  try {
    if (window.ImgoAccountVault && window.ImgoAccountVault.currentToken) {
      return window.ImgoAccountVault.currentToken() || '';
    }
    const item = document.cookie.split(';').map(value => value.trim()).find(value => value.indexOf('authToken=') === 0);
    return item ? decodeURIComponent(item.slice(10)) : '';
  } catch (_) {
    return '';
  }
};

component.methods.checkStatus = function () {
  return Boolean(this.websocket && this.websocket.readyState === WebSocket.OPEN && this.is_open_socket);
};

component.methods.websocketSend = function (payload) {
  if (!this.checkStatus()) {
    this.imgoScheduleReconnect(0);
    return false;
  }
  try {
    this.websocket.send(JSON.stringify(payload));
    return true;
  } catch (_) {
    try { this.websocket.close(); } catch (_) {}
    this.imgoScheduleReconnect(0);
    return false;
  }
};

component.methods.start = function () {
  clearInterval(this.heartbeatInterval);
  this.heartbeatInterval = setInterval(() => {
    if (document.hidden) return;
    if (!this.checkStatus() || Date.now() - this.imgoLastSocketActivity > 75000) {
      this.imgoForceReconnect();
      return;
    }
    this.websocketSend({type: 'ping'});
  }, 25000);
};

component.methods.imgoScheduleReconnect = function (delay) {
  if (this.imgoManualSocketClose || this.reconnectTimeOut) return;
  this.$store.state.wsStatus = false;
  const attempt = this.imgoReconnectAttempt++;
  const wait = typeof delay === 'number' ? delay : Math.min(30000, 1000 * Math.pow(2, Math.min(attempt, 5)));
  this.reconnectTimeOut = setTimeout(() => {
    this.reconnectTimeOut = null;
    this.initWebSocket();
  }, wait);
};

component.methods.imgoForceReconnect = function () {
  if (this.imgoManualSocketClose) return;
  const current = this.websocket;
  this.websocket = null;
  this.is_open_socket = false;
  this.$store.state.wsStatus = false;
  this.imgoClearSocketTimers();
  if (current && current.readyState < WebSocket.CLOSING) {
    try { current.close(); } catch (_) {}
  }
  this.imgoScheduleReconnect(0);
};

component.methods.initWebSocket = function () {
  if (this.imgoManualSocketClose) return;
  if (this.websocket && (this.websocket.readyState === WebSocket.OPEN || this.websocket.readyState === WebSocket.CONNECTING)) return;
  clearTimeout(this.reconnectTimeOut);
  this.reconnectTimeOut = null;
  const generation = ++this.imgoSocketGeneration;
  let socket;
  try {
    socket = new WebSocket(this.getWsUrl());
  } catch (_) {
    this.imgoScheduleReconnect();
    return;
  }
  this.websocket = socket;
  this.is_open_socket = false;
  this.$store.state.wsStatus = false;
  socket.onopen = () => {
    if (generation !== this.imgoSocketGeneration || socket !== this.websocket) return;
    this.imgoLastSocketActivity = Date.now();
  };
  socket.onmessage = event => this.websocketOnMessage(event, socket, generation);
  socket.onerror = () => {
    if (socket === this.websocket) {
      try { socket.close(); } catch (_) { this.imgoScheduleReconnect(); }
    }
  };
  socket.onclose = event => this.websocketClose(event, socket, generation);
  this.$root.$websocket = socket;
};

component.methods.websocketOnMessage = async function (event, socket, generation) {
  if (socket !== this.websocket || generation !== this.imgoSocketGeneration) return;
  this.imgoLastSocketActivity = Date.now();
  let packet;
  try { packet = JSON.parse(event.data); } catch (_) { return; }
  if (packet.type === 'pong') return;
  if (packet.type === 'error' && Number(packet.code) === 401) {
    this.imgoManualSocketClose = true;
    this.$store.state.wsStatus = false;
    try { socket.close(); } catch (_) {}
    this.$store.dispatch('LogOut').finally(() => this.$router.push({path: '/login'}));
    return;
  }
  if (packet.type !== 'init') {
    this.$store.commit('catchSocketAction', packet);
    return;
  }
  const user = this.$store.state.userInfo;
  const token = this.imgoCurrentToken();
  if (!user || !user.user_id || !token) {
    this.imgoManualSocketClose = true;
    try { socket.close(); } catch (_) {}
    return;
  }
  document.cookie = 'client_id=' + encodeURIComponent(packet.client_id) + '; path=/; SameSite=Lax';
  try {
    const result = await this.$api.commonApi.bindClientIdAPI({client_id: packet.client_id, user_id: user.user_id});
    if (socket !== this.websocket || generation !== this.imgoSocketGeneration || Number(result.code) !== 0) throw new Error('socket bind failed');
    socket.send(JSON.stringify({type: 'bindUid', user_id: user.user_id, token}));
    this.is_open_socket = true;
    this.imgoReconnectAttempt = 0;
    this.imgoLastSocketActivity = Date.now();
    this.$store.state.wsStatus = true;
    this.start();
    this.$store.commit('catchSocketAction', {type: 'friendApplyChanged'});
    this.$nextTick(() => this.$store.commit('catchSocketAction', {type: 'socketRecovered', data: {client_id: packet.client_id}}));
  } catch (_) {
    try { socket.close(); } catch (_) { this.imgoScheduleReconnect(); }
  }
};

component.methods.websocketClose = function (_event, socket, generation) {
  if (socket !== this.websocket || generation !== this.imgoSocketGeneration) return;
  this.websocket = null;
  this.is_open_socket = false;
  this.$store.state.wsStatus = false;
  this.imgoClearSocketTimers();
  if (!this.imgoManualSocketClose) this.imgoScheduleReconnect();
};

component.methods.reconnect = function () {
  this.imgoManualSocketClose = false;
  this.imgoReconnectAttempt = 0;
  this.imgoForceReconnect();
};

component.methods.close = function () {
  this.imgoManualSocketClose = true;
  this.imgoClearSocketTimers();
  const socket = this.websocket;
  this.websocket = null;
  this.is_open_socket = false;
  this.$store.state.wsStatus = false;
  if (socket) try { socket.close(); } catch (_) {}
};

component.created = function () {
  this.imgoManualSocketClose = false;
  this.imgoVisibilityHandler = () => {
    if (document.hidden) return;
    if (!this.checkStatus() || Date.now() - this.imgoLastSocketActivity > 45000) this.imgoForceReconnect();
    else {
      this.websocketSend({type: 'ping'});
      this.$store.commit('catchSocketAction', {type: 'socketRecovered', data: {reason: 'visible'}});
    }
  };
  this.imgoOnlineHandler = () => this.imgoForceReconnect();
  document.addEventListener('visibilitychange', this.imgoVisibilityHandler);
  window.addEventListener('online', this.imgoOnlineHandler);
  this.initWebSocket();
};

component.beforeDestroy = function () {
  document.removeEventListener('visibilitychange', this.imgoVisibilityHandler);
  window.removeEventListener('online', this.imgoOnlineHandler);
  this.close();
};
