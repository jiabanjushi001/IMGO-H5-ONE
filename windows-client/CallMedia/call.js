'use strict';

const bridge = window.chrome.webview;
const localView = document.querySelector('#local');
const remoteView = document.querySelector('#remote');
const preparedView = document.querySelector('#prepared');
let peer;
let localStream;
let cameraSender;
let cameraId = '';
let closed = false;
let preparedStarted = false;
let pendingIce = [];
const seenIce = new Set();
let work = Promise.resolve();

function send(kind, fields = {}) {
    if (!closed) bridge.postMessage({kind, ...fields});
}

async function openMicrophone(camera) {
    return navigator.mediaDevices.getUserMedia({
        audio: {echoCancellation: true, noiseSuppression: true},
        video: camera ? {width: {ideal: 960}, height: {ideal: 540}} : false
    });
}

async function openPreparedSource(path) {
    try {
        if (!/^prepared\.(mp4|mov|webm|m4v)$/.test(path) || !preparedView.captureStream) throw new Error();
        preparedView.src = path;
        await preparedView.play();
        if (closed) return null;
        const captured = preparedView.captureStream();
        localStream = captured;
        preparedView.pause();
        preparedView.currentTime = 0;
        if (!captured.getVideoTracks().length) throw new Error();
        return captured;
    } catch {
        throw new DOMException('预录视频无法播放', 'PreparedVideoError');
    }
}

async function start(options) {
    let stream;
    let audioSource = 'microphone';
    if (options.prepared) {
        stream = await openPreparedSource(options.prepared);
        if (!stream || closed) return;
        if (stream.getAudioTracks().length) audioSource = 'recorded';
        else {
            const mic = await openMicrophone(false);
            for (const track of mic.getTracks()) {
                if (closed) track.stop();
                else stream.addTrack(track);
            }
        }
    } else {
        stream = await openMicrophone(options.camera);
    }
    if (closed) {
        stream.getTracks().forEach(track => track.stop());
        return;
    }

    localStream = stream;
    if (options.prepared) preparedView.classList.add('preview');
    else {
        localView.classList.toggle('camera', options.camera);
        localView.srcObject = stream;
    }
    peer = new RTCPeerConnection({iceServers: options.iceServers});
    for (const track of stream.getTracks()) {
        const sender = peer.addTrack(track, stream);
        if (options.camera && track.kind === 'video') {
            cameraSender = sender;
            cameraId = track.getSettings().deviceId || '';
        }
    }
    const remoteStream = new MediaStream();
    remoteView.srcObject = remoteStream;
    peer.ontrack = event => {
        remoteStream.addTrack(event.track);
        remoteView.play().catch(() => send('playbackBlocked'));
    };
    peer.onicecandidate = event => {
        if (event.candidate) send('iceCandidate', {candidate: JSON.stringify(event.candidate.toJSON())});
    };
    peer.onconnectionstatechange = () => {
        const state = peer.connectionState;
        send('state', {state});
        if (state === 'connected' && options.prepared && !preparedStarted && !closed) {
            preparedStarted = true;
            preparedView.play().catch(() => send('error', {name: 'PreparedVideoError', stage: 'play'}));
        }
    };
    send('mediaReady', {audioSource});
    if (cameraSender) {
        navigator.mediaDevices.enumerateDevices().then(devices => {
            send('cameras', {deviceId: cameraId, devices: devices.filter(device => device.kind === 'videoinput')
                .map((device,index) => ({id:device.deviceId,label:device.label || '摄像头 ' + (index + 1)}))});
        }).catch(() => {});
    }
}

async function describe(type) {
    const description = type === 'offer' ? await peer.createOffer() : await peer.createAnswer();
    await peer.setLocalDescription(description);
    send(type, {sdp: peer.localDescription.sdp});
}

async function selectCamera(deviceId, enabled) {
    if (!cameraSender) return;
    const previous = cameraSender.track;
    if (enabled && previous && cameraId === deviceId && previous.readyState === 'live') {
        send('cameraChanged', {deviceId:cameraId, enabled:1});
        return;
    }
    let next;
    try {
        if (enabled) {
            const stream = await navigator.mediaDevices.getUserMedia({audio:false,video:{
                width:{ideal:960}, height:{ideal:540}, ...(deviceId ? {deviceId:{exact:deviceId}} : {})
            }});
            next = stream.getVideoTracks()[0];
            if (closed) { next?.stop(); return; }
            if (!next) throw new Error('No camera track');
        }
        await cameraSender.replaceTrack(next || null);
        if (closed) { next?.stop(); return; }
        if (previous) {
            localStream.removeTrack(previous);
            previous.stop();
        }
        if (next) {
            localStream.addTrack(next);
            cameraId = next.getSettings().deviceId || deviceId;
        }
        localView.classList.toggle('camera', !!next);
        send('cameraChanged', {deviceId:cameraId, enabled:next ? 1 : 0});
    } catch {
        next?.stop();
        send('cameraError');
    }
}

async function signal(event) {
    if (!peer || closed) return;
    switch (event.kind) {
        case 'makeOffer':
            if (!peer.localDescription) await describe('offer');
            break;
        case 'offer':
        case 'answer':
            if (peer.remoteDescription) break;
            await peer.setRemoteDescription({type: event.kind, sdp: event.sdp});
            for (const candidate of pendingIce) await peer.addIceCandidate(candidate);
            pendingIce = [];
            if (event.kind === 'offer') await describe('answer');
            break;
        case 'iceCandidate': {
            const candidate = typeof event.candidate === 'string' ? JSON.parse(event.candidate) : event.candidate;
            if (!candidate || !candidate.candidate) break;
            const key = JSON.stringify([candidate.candidate, candidate.sdpMid, candidate.sdpMLineIndex]);
            if (seenIce.has(key)) break;
            seenIce.add(key);
            if (peer.remoteDescription) await peer.addIceCandidate(candidate);
            else pendingIce.push(candidate);
            break;
        }
        case 'selectCamera':
            await selectCamera(event.deviceId, event.enabled);
            break;
        case 'toggleTrack':
            for (const track of localStream.getTracks()) {
                if (track.kind === event.track) track.enabled = event.enabled;
            }
            if (event.track === 'video') localView.classList.toggle('camera', event.enabled);
            break;
        case 'play':
            await remoteView.play().then(() => send('playing'), () => send('playbackBlocked'));
            break;
    }
}

function stop() {
    closed = true;
    if (peer) peer.close();
    if (localStream) localStream.getTracks().forEach(track => track.stop());
    localView.srcObject = remoteView.srcObject = null;
    if (preparedView.hasAttribute('src')) {
        preparedView.pause();
        preparedView.removeAttribute('src');
        preparedView.load();
    }
}

bridge.addEventListener('message', ({data}) => {
    if (data.kind === 'stop') {
        stop();
        return;
    }
    work = work.then(async () => {
        if (closed) return;
        if (data.kind === 'start') await start(data);
        else await signal(data);
    }).catch(error => send('error', {name: error.name || 'Error', stage: data.kind}));
});
window.addEventListener('pagehide', stop);
send('ready');
