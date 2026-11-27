const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { execFileSync } = require('node:child_process');

const root = path.resolve(__dirname, '..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const source = read('frontend/chat-prepared-video.js');

const helperMatch = source.match(/function imgoCanUsePreparedVideo\(userInfo\) \{[\s\S]*?\n\}/);
assert.ok(helperMatch, 'prepared-video role helper must exist');
const context = {};
vm.runInNewContext(helperMatch[0] + '\nthis.check = imgoCanUsePreparedVideo;', context);
assert.equal(context.check({ user_id: 1, agent_mode: 0 }), true);
assert.equal(context.check({ user_id: 7, agent_mode: 1 }), true);
assert.equal(context.check({ user_id: 8, agent_mode: 0 }), false);

execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const firstBuild = read('public/assets/js/app.85372e4e.js');
execFileSync(process.execPath, ['scripts/build-maintenance.cjs'], { cwd: root, stdio: 'inherit' });
const app = read('public/assets/js/app.85372e4e.js');
assert.equal(app, firstBuild, 'prepared-video integration must survive repeated builds unchanged');
assert.match(app, /IMGO_PREPARED_RTC_BEGIN/);
assert.match(app, /getCallVideosAPI/);
assert.match(app, /imgo-prepared-video-dialog/);
assert.match(app, /video_mode:this\.preparedVideoMode/);
assert.match(app, /video-file-id|video_file_id:this\.preparedVideoFileId/);
assert.match(app, /captureStream/);
assert.match(app, /video\.src = source/);
assert.doesNotMatch(app, /response\.blob\(\)/);
assert.match(app, /pendingIceCandidates/);
assert.match(app, /await this\.pc\.setRemoteDescription/);
assert.match(app, /await this\.pc\.setLocalDescription/);
assert.match(app, /facingMode: \{ ideal: 'user' \}/);
assert.match(app, /this\.videoMode === 'recorded' && this\.isReceived/);
assert.match(app, /preparedVideoStarted/);
assert.match(app, /video\.pause\(\);\s*video\.currentTime = 0/);
assert.match(app, /connectionState === 'connected'/);
assert.match(app, /\['connected', 'completed'\]\.includes\(this\.pc\.iceConnectionState\)/);
assert.match(app, /e&&e\.data&&e\.data\.extends&&"907"==e\.data\.extends\.code/);

const playbackEvents = [];
const preparedVideo = {
  muted: false,
  loop: false,
  playsInline: false,
  preload: '',
  style: {},
  src: '',
  currentTime: 5,
  paused: true,
  async play() {
    this.paused = false;
    playbackEvents.push('play');
  },
  pause() {
    this.paused = true;
    playbackEvents.push('pause');
  },
  captureStream() {
    playbackEvents.push('capture');
    return {
      getVideoTracks: () => [{ id: 'video-track' }],
      getAudioTracks: () => [{ id: 'audio-track' }]
    };
  },
  removeAttribute() {},
  remove() {}
};
const rtcComponent = {
  props: {},
  data() { return {}; },
  methods: {
    initPeer() {
      this.pc = {
        connectionState: 'new',
        iceConnectionState: 'new',
        close() {}
      };
    },
    onicecandidate() {},
    closeLocalMedia() {}
  }
};
const rtcContext = {
  component: rtcComponent,
  document: {
    createElement: () => preparedVideo,
    body: { appendChild() {} }
  },
  navigator: { mediaDevices: { getUserMedia: async () => ({ getAudioTracks: () => [] }) } },
  MediaStream: class {},
  RTCSessionDescription: class {},
  RTCIceCandidate: class {}
};
vm.runInNewContext('var component = this.component;\n' + read('frontend/webrtc-prepared-video.js') + '\nthis.component = component;', rtcContext);

(async () => {
  const state = {
    videoUrl: '/storage/test.mp4',
    videoMode: 'recorded',
    isReceived: false,
    preparedVideoElement: null,
    preparedVideoStarted: false,
    pendingIceCandidates: [],
    remoteStream: null,
    $message: { error(message) { throw new Error(message); } },
    hangup() { throw new Error('connected playback must not hang up'); }
  };
  Object.assign(state, rtcContext.component.methods);
  await state.imgoCreatePreparedStream();
  assert.deepEqual(playbackEvents, ['play', 'capture', 'pause'], 'recorded video must be primed and paused before ringing starts');
  assert.equal(preparedVideo.paused, true, 'recorded video must remain paused while waiting for acceptance');
  assert.equal(preparedVideo.currentTime, 0, 'recorded video must rewind before waiting for acceptance');

  state.initPeer({});
  assert.equal(playbackEvents.filter(event => event === 'play').length, 1, 'peer setup must not start playback');
  state.pc.connectionState = 'connected';
  await state.pc.onconnectionstatechange();
  assert.equal(preparedVideo.paused, false, 'recorded video must start only after the peer connection is established');
  assert.equal(preparedVideo.currentTime, 0, 'connected playback must begin at the first frame');
  assert.equal(playbackEvents.filter(event => event === 'play').length, 2);

  state.pc.iceConnectionState = 'completed';
  await state.pc.oniceconnectionstatechange();
  assert.equal(playbackEvents.filter(event => event === 'play').length, 2, 'duplicate connection events must not restart the video');
  console.log('chat prepared-video tests passed');
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
