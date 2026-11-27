component.props = Object.assign({}, component.props, {
  videoMode: { type: String, default: '' },
  videoUrl: { type: String, default: '' }
});

var imgoPreparedRtcBaseData = component.data;
component.data = function () {
  return Object.assign(imgoPreparedRtcBaseData.call(this), {
    preparedVideoElement: null,
    preparedVideoStarted: false,
    pendingIceCandidates: [],
    seenIceCandidates: {},
    answering: false,
    acceptSent: false,
    callTimerStarted: false,
    offerCreating: false,
    offerCreated: false,
    answerCreating: false,
    answerCreated: false
  });
};

var imgoPreparedRtcBaseInitPeer = component.methods.initPeer;
component.methods.initPeer = function (stream) {
  imgoPreparedRtcBaseInitPeer.call(this, stream);
  this.pendingIceCandidates = [];
  this.seenIceCandidates = {};
  this.callTimerStarted = false;
  this.offerCreating = false;
  this.offerCreated = false;
  this.answerCreating = false;
  this.answerCreated = false;
  this.pc.ontrack = function (event) {
    var remote = event.streams && event.streams[0] ? event.streams[0] : new MediaStream([event.track]);
    if (this.remoteStream && (!event.streams || !event.streams[0])) {
      var exists = this.remoteStream.getTracks().some(function (track) { return track.id === event.track.id; });
      if (!exists) this.remoteStream.addTrack(event.track);
    } else {
      this.remoteStream = remote;
    }
    if (this.remoteVideo) {
      this.remoteVideo.srcObject = this.remoteStream;
      var playTask = this.remoteVideo.play();
      if (playTask && typeof playTask.catch === 'function') playTask.catch(function () {});
    }
  }.bind(this);
  this.pc.onconnectionstatechange = function () {
    if (this.pc && this.pc.connectionState === 'connected') {
      this.startTime();
      this.imgoStartPreparedPlayback();
    }
  }.bind(this);
  this.pc.oniceconnectionstatechange = function () {
    if (this.pc && ['connected', 'completed'].includes(this.pc.iceConnectionState)) {
      this.startTime();
      this.imgoStartPreparedPlayback();
    }
  }.bind(this);
  this.onicecandidate();
};

var imgoPreparedRtcBaseStartTime = component.methods.startTime;
component.methods.startTime = function () {
  if (this.callTimerStarted) return;
  this.callTimerStarted = true;
  if (typeof imgoPreparedRtcBaseStartTime === 'function') {
    imgoPreparedRtcBaseStartTime.call(this);
  }
};

component.methods.imgoStartPreparedPlayback = async function () {
  var video = this.preparedVideoElement;
  if (!video || this.preparedVideoStarted || this.videoMode !== 'recorded' || this.isReceived) return;
  this.preparedVideoStarted = true;
  try {
    video.currentTime = 0;
    await video.play();
  } catch (error) {
    this.preparedVideoStarted = false;
    this.$message.error('预设视频播放失败，请重新发起通话');
    this.hangup(true);
  }
};

component.methods.imgoCreatePreparedStream = async function () {
  var source = String(this.videoUrl || '');
  if (!source) throw new Error('预设视频地址无效');
  var video = document.createElement('video');
  video.muted = true;
  video.loop = true;
  video.playsInline = true;
  video.preload = 'auto';
  video.style.position = 'fixed';
  video.style.width = '1px';
  video.style.height = '1px';
  video.style.opacity = '0';
  video.style.pointerEvents = 'none';
  video.src = source;
  document.body.appendChild(video);
  this.preparedVideoElement = video;
  this.preparedVideoStarted = false;
  try {
    await video.play();
  } catch (error) {
    throw new Error('预设视频加载失败，请检查文件是否存在或格式是否受支持');
  }
  var capture = video.captureStream || video.mozCaptureStream;
  if (typeof capture !== 'function') throw new Error('当前浏览器不支持预设视频通话');
  var stream = capture.call(video);
  video.pause();
  video.currentTime = 0;
  if (!stream || !stream.getVideoTracks().length) throw new Error('预设视频没有可用画面');
  if (!stream.getAudioTracks().length) {
    var microphone = await navigator.mediaDevices.getUserMedia({ video: false, audio: { echoCancellation: true } });
    microphone.getAudioTracks().forEach(function (track) { stream.addTrack(track); });
  }
  return stream;
};

component.methods.imgoCreateDeviceStream = async function (isVideo) {
  if (!navigator.mediaDevices || typeof navigator.mediaDevices.getUserMedia !== 'function') {
    throw new Error('当前浏览器不支持摄像头或麦克风');
  }
  var audio = { echoCancellation: true, noiseSuppression: true, autoGainControl: true };
  if (Number(isVideo) !== 1) return navigator.mediaDevices.getUserMedia({ video: false, audio: audio });
  if (this.videoMode === 'recorded' && this.isReceived) {
    return navigator.mediaDevices.getUserMedia({ video: false, audio: audio });
  }
  try {
    return await navigator.mediaDevices.getUserMedia({
      video: { facingMode: { ideal: 'user' }, width: { ideal: 1280 }, height: { ideal: 720 } },
      audio: audio
    });
  } catch (error) {
    if (!['OverconstrainedError', 'ConstraintNotSatisfiedError'].includes(error && error.name)) throw error;
    return navigator.mediaDevices.getUserMedia({ video: true, audio: audio });
  }
};

component.methods.initLocalStream = async function (outgoing, isVideo) {
  this.offerParams = Number(isVideo) === 1
    ? { offerToReceiveAudio: true, offerToReceiveVideo: true }
    : { offerToReceiveAudio: true, offerToReceiveVideo: false };
  try {
    var stream = outgoing && Number(isVideo) === 1 && this.videoMode === 'recorded'
      ? await this.imgoCreatePreparedStream()
      : await this.imgoCreateDeviceStream(isVideo);
    this.initPeer(stream);
    if (outgoing) {
      this.$emit('message', { event: 'calling' });
      this.status = 1;
      this.timer = setInterval(function () {
        this.cutdown--;
        if (this.cutdown === 0) this.hangup(true);
      }.bind(this), 1000);
    } else {
      this.acceptSent = true;
      this.$emit('message', { event: 'acceptRtc', code: 904 });
    }
    return true;
  } catch (error) {
    var name = error && error.name || '';
    var message = error && error.message || '无法打开摄像头或麦克风';
    if (name === 'NotAllowedError' || name === 'PermissionDeniedError') message = '请允许访问摄像头和麦克风';
    if (name === 'NotFoundError' || name === 'DevicesNotFoundError') message = '未检测到摄像头或麦克风';
    this.$message.error(message);
    this.caller = null;
    this.status = 0;
    return false;
  }
};

component.methods.answer = async function () {
  if (this.answering) return;
  this.answering = true;
  this.playMusicCall('close');
  var ready;
  try {
    ready = await this.initLocalStream(false, this.is_video);
  } finally {
    this.answering = false;
  }
  if (!ready) return;
  this.status = 2;
};

component.methods.imgoSetRemoteDescription = async function (description) {
  await this.pc.setRemoteDescription(new RTCSessionDescription(description));
  var candidates = this.pendingIceCandidates.splice(0);
  for (var index = 0; index < candidates.length; index += 1) {
    await this.pc.addIceCandidate(new RTCIceCandidate(candidates[index]));
  }
};

component.methods.imgoAddIceCandidate = async function (candidate) {
  if (!candidate) return;
  var parsed = typeof candidate === 'object' ? candidate : JSON.parse(candidate);
  var key = JSON.stringify(parsed);
  if (this.seenIceCandidates[key]) return;
  this.seenIceCandidates[key] = true;
  if (!this.pc.remoteDescription || !this.pc.remoteDescription.type) {
    this.pendingIceCandidates.push(parsed);
    return;
  }
  await this.pc.addIceCandidate(new RTCIceCandidate(parsed));
};

component.methods.createOffer = async function () {
  if (this.offerCreating || this.offerCreated) return;
  this.offerCreating = true;
  try {
    var offer = await this.pc.createOffer(this.offerParams);
    await this.pc.setLocalDescription(offer);
    this.offerCreated = true;
    this.$emit('message', { event: 'offer', sdp: this.pc.localDescription.sdp });
  } finally {
    this.offerCreating = false;
  }
};

component.methods.createAnswer = async function () {
  if (this.answerCreating || this.answerCreated) return;
  this.answerCreating = true;
  try {
    var answer = await this.pc.createAnswer(this.offerParams);
    await this.pc.setLocalDescription(answer);
    this.answerCreated = true;
    this.$emit('message', { event: 'answer', sdp: this.pc.localDescription.sdp });
  } finally {
    this.answerCreating = false;
  }
};

component.methods.webrtcAction = async function (message) {
  var event = message.extends || {};
  try {
    switch (event.event) {
      case 'calling':
        this.caller = message.fromUser;
        this.is_video = parseInt(event.type, 10);
	    this.videoMode = event.video_mode || '';
        this.status = 3;
        this.isReceived = true;
        this.playMusicCall('state');
        break;
      case 'hangup':
        this.hangup(false);
        break;
      case 'busy':
        this.status = 4;
        this.hangup(false);
        break;
      case 'acceptRtc':
        this.status = 2;
        clearInterval(this.timer);
        this.timer = null;
        this.playMusicCall('close');
        await this.createOffer();
        break;
      case 'answer':
        if (!this.pc.remoteDescription || this.pc.remoteDescription.type !== 'answer') {
          await this.imgoSetRemoteDescription({ type: 'answer', sdp: event.sdp });
        }
        this.startTime();
        break;
      case 'iceCandidate':
        await this.imgoAddIceCandidate(event.iceCandidate);
        break;
      case 'offer':
        if (!this.pc.remoteDescription || this.pc.remoteDescription.type !== 'offer') {
          await this.imgoSetRemoteDescription({ type: 'offer', sdp: event.sdp });
        }
        await this.createAnswer();
        this.startTime();
        break;
    }
  } catch (error) {
    console.warn('WebRTC signaling failed', error);
    if (this.$message && typeof this.$message.error === 'function') {
      this.$message.error('通话连接失败，请重新拨打');
    }
    this.hangup(true);
  }
};

var imgoPreparedRtcBaseCloseLocalMedia = component.methods.closeLocalMedia;
component.methods.closeLocalMedia = function () {
  imgoPreparedRtcBaseCloseLocalMedia.call(this);
  if (this.remoteStream && this.remoteStream.getTracks) {
    this.remoteStream.getTracks().forEach(function (track) { track.stop(); });
  }
  this.remoteStream = null;
  if (this.remoteVideo) this.remoteVideo.srcObject = null;
  if (this.localVideo) this.localVideo.srcObject = null;
  this.acceptSent = false;
  this.pendingIceCandidates = [];
  this.seenIceCandidates = {};
  this.callTimerStarted = false;
  this.offerCreating = false;
  this.offerCreated = false;
  this.answerCreating = false;
  this.answerCreated = false;
  this.timer = null;
  this.timerIntervalId = null;
  if (this.pc && this.pc.connectionState !== 'closed') this.pc.close();
  this.pc = null;
  if (this.preparedVideoElement) {
    this.preparedVideoElement.pause();
    this.preparedVideoElement.removeAttribute('src');
    this.preparedVideoElement.remove();
    this.preparedVideoElement = null;
  }
  this.preparedVideoStarted = false;
};
