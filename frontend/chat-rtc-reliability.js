// The accept signal can arrive before the HTTP response for the original
// "calling" message. In that case the legacy socket watcher has no wsData yet
// and drops the signal because it cannot compare call ids. Associate that
// first response with the pending outgoing call before handing it to the
// original watcher.
var imgoRtcBaseSocketActionWatch = component.watch.socketAction;

function imgoRtcUserId(user) {
  if (!user) return 0;
  return Number(user.id != null ? user.id : user.user_id) || 0;
}

component.watch.socketAction = function (packet) {
  var message = packet && packet.type === 'webrtc' ? packet.data : null;
  var signal = message && message.extends ? message.extends : null;
  var rtc = this.$refs && this.$refs.webrtc;

  // The server echoes acceptance to all of the callee's sessions. This
  // window already accepted locally; other windows still need the event.
  if (signal && signal.event === 'otherOpt' && Number(signal.code) === 904 &&
      rtc && rtc.acceptSent && this.wsData && this.wsData.id === message.id &&
      imgoRtcUserId(this.user) > 0 && imgoRtcUserId(this.user) === imgoRtcUserId(message.fromUser)) {
    return;
  }

  if (signal && signal.event === 'acceptRtc' && !this.wsData && rtc && Number(rtc.status) === 1) {
    var expectedUserId = imgoRtcUserId(this.caller || rtc.caller);
    var remoteUserId = imgoRtcUserId(message.fromUser);
    if (message.id && expectedUserId > 0 && expectedUserId === remoteUserId) {
      this.wsData = Object.assign({}, message, { toContactId: remoteUserId });
    }
  }

  return imgoRtcBaseSocketActionWatch.call(this, packet);
};
