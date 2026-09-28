/* IMGO_GROUP_MUTE_CHAT_BEGIN */
var imgoGroupMuteOriginalData = component.data;
var imgoGroupMuteOriginalMethods = component.methods || {};
var imgoGroupMuteOriginalSubmit = imgoGroupMuteOriginalMethods.setNoSpeak;

function imgoCurrentUserCanMute(instance) {
  var role = Number(instance && instance.currentChat && instance.currentChat.role);
  return role === 1 || role === 2;
}

component.data = function () {
  var state = typeof imgoGroupMuteOriginalData === 'function'
    ? imgoGroupMuteOriginalData.apply(this, arguments)
    : {};
  state = state || {};
  var muteAction = Array.isArray(state.groupMenu)
    ? state.groupMenu.find(function (item) { return item && item.text === '设置禁言'; })
    : null;
  if (muteAction) {
    var originalVisible = muteAction.visible;
    muteAction.visible = function (menuContext) {
      return imgoCurrentUserCanMute(this) &&
        (typeof originalVisible !== 'function' || originalVisible(menuContext));
    }.bind(this);
  }
  return state;
};

component.methods = Object.assign({}, imgoGroupMuteOriginalMethods, {
  setNoSpeak: function () {
    if (!imgoCurrentUserCanMute(this)) {
      this.noSpeakBox = false;
      if (this.$message && typeof this.$message.error === 'function') {
        this.$message.error('只有群主或群管理员可以设置禁言');
      }
      return;
    }
    if (typeof imgoGroupMuteOriginalSubmit === 'function') {
      return imgoGroupMuteOriginalSubmit.apply(this, arguments);
    }
  }
});
/* IMGO_GROUP_MUTE_CHAT_END */
