const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');

function installAdapter(component, file, globals = {}) {
  const context = Object.assign({ component, console }, globals);
  vm.runInNewContext('var component = this.component;\n' + read(file) + '\nthis.component = component;', context);
  return context.component;
}

// Regression: a fast callee can accept before the caller's HTTP request has
// populated wsData. The adapter must associate that early signal and let the
// existing watcher deliver it to the WebRTC component.
const delivered = [];
const chatComponent = {
  watch: {
    socketAction(packet) {
      if (packet.type === 'webrtc' && this.wsData && this.wsData.id === packet.data.id) {
        this.$refs.webrtc.webrtcAction(packet.data);
      }
    }
  }
};
installAdapter(chatComponent, 'frontend/chat-rtc-reliability.js');

const caller = {
  wsData: null,
  caller: { id: 22 },
  $refs: {
    webrtc: {
      status: 1,
      webrtcAction(message) { delivered.push(message); }
    }
  }
};
const earlyAccept = {
  type: 'webrtc',
  data: {
    id: 'call-101',
    msg_id: 'message-101',
    fromUser: { id: 22, user_id: 22 },
    extends: { event: 'acceptRtc' }
  }
};
chatComponent.watch.socketAction.call(caller, earlyAccept);
assert.equal(caller.wsData.id, 'call-101');
assert.equal(caller.wsData.toContactId, 22);
assert.equal(delivered.length, 1, 'early accept must reach the WebRTC child');

const unrelatedCaller = {
  wsData: null,
  caller: { id: 23 },
  $refs: {
    webrtc: {
      status: 1,
      webrtcAction() { throw new Error('unrelated accept must not be delivered'); }
    }
  }
};
chatComponent.watch.socketAction.call(unrelatedCaller, earlyAccept);
assert.equal(unrelatedCaller.wsData, null, 'accept from another user must not attach to the call');

const acceptEcho = {
  type: 'webrtc',
  data: { id: 'call-101', fromUser: { user_id: 22 }, extends: { event: 'otherOpt', code: 904 } }
};
const callee = {
  user: { id: 22 },
  wsData: earlyAccept.data,
  $refs: { webrtc: { acceptSent: true, webrtcAction(message) { delivered.push(message); } } }
};
chatComponent.watch.socketAction.call(callee, acceptEcho);
assert.equal(delivered.length, 1, 'the accepting window must retain its peer after the server echo');
callee.$refs.webrtc.acceptSent = false;
chatComponent.watch.socketAction.call(callee, acceptEcho);
assert.equal(delivered.length, 2, 'another window must still receive the operation');

let timerStarts = 0;
const emitted = [];
const rtcComponent = {
  props: {},
  data() { return {}; },
  methods: {
    initPeer() {},
    onicecandidate() {},
    closeLocalMedia() {},
    startTime() { timerStarts += 1; }
  }
};
installAdapter(rtcComponent, 'frontend/webrtc-prepared-video.js', {
  document: {},
  navigator: {},
  MediaStream: class {},
  RTCSessionDescription: class {
    constructor(value) { Object.assign(this, value); }
  },
  RTCIceCandidate: class {
    constructor(value) { Object.assign(this, value); }
  },
  clearInterval() {}
});

function createRtcState() {
  const state = Object.assign(rtcComponent.data.call({}), {
    status: 1,
    timer: 7,
    caller: { id: 22 },
    offerParams: {},
    $message: { error(message) { throw new Error(message); } },
    $emit(event, payload) { emitted.push({ event, payload }); },
    playMusicCall() {},
    hangup() { throw new Error('valid signaling must not hang up'); }
  });
  Object.assign(state, rtcComponent.methods);
  return state;
}

(async () => {
  const answerer = createRtcState();
  answerer.initLocalStream = async () => true;
  answerer.is_video = 0;
  await answerer.answer();
  assert.equal(answerer.status, 2);
  assert.equal(timerStarts, 0, 'callee must not show a connected timer before SDP exchange');

  let offers = 0;
  const outgoing = createRtcState();
  outgoing.pc = {
    remoteDescription: null,
    localDescription: null,
    async createOffer() { offers += 1; return { type: 'offer', sdp: 'offer-sdp' }; },
    async setLocalDescription(description) { this.localDescription = description; },
    async setRemoteDescription(description) { this.remoteDescription = description; },
    async addIceCandidate() {}
  };
  await outgoing.webrtcAction(earlyAccept.data);
  await outgoing.webrtcAction(earlyAccept.data);
  assert.equal(outgoing.status, 2);
  assert.equal(offers, 1, 'duplicate accept signals must create only one offer');
  assert.equal(timerStarts, 0, 'caller must wait for the answer before starting its timer');

  const answerSignal = { extends: { event: 'answer', sdp: 'answer-sdp' } };
  await outgoing.webrtcAction(answerSignal);
  await outgoing.webrtcAction(answerSignal);
  assert.equal(outgoing.pc.remoteDescription.type, 'answer');
  assert.equal(timerStarts, 1, 'caller timer must start once after the answer is installed');

  let answers = 0;
  const incoming = createRtcState();
  incoming.pc = {
    remoteDescription: null,
    localDescription: null,
    async setRemoteDescription(description) { this.remoteDescription = description; },
    async createAnswer() { answers += 1; return { type: 'answer', sdp: 'answer-sdp' }; },
    async setLocalDescription(description) { this.localDescription = description; },
    async addIceCandidate() {}
  };
  const offerSignal = { extends: { event: 'offer', sdp: 'offer-sdp' } };
  await incoming.webrtcAction(offerSignal);
  await incoming.webrtcAction(offerSignal);
  assert.equal(answers, 1, 'duplicate offer signals must create only one answer');
  assert.equal(timerStarts, 2, 'callee timer must start once after sending its answer');

  console.log('chat RTC reliability tests passed');
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
