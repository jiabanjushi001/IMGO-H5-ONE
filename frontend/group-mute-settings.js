/* IMGO_GROUP_MUTE_SETTINGS_BEGIN */
var imgoGroupMuteSettingsRender = J;

function imgoGroupMuteVNodeText(node) {
  if (node == null) return '';
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  var text = node.text == null ? '' : String(node.text);
  if (Array.isArray(node.children)) {
    text += node.children.map(imgoGroupMuteVNodeText).join('');
  }
  return text;
}

J = function () {
  var root = imgoGroupMuteSettingsRender.call(this);
  var role = Number(this.groupInfo && this.groupInfo.isJoin);
  var canMute = role === 1 || role === 2;
  if (!canMute && root && Array.isArray(root.children)) {
    root.children = root.children.filter(function (child) {
      return imgoGroupMuteVNodeText(child).indexOf('群禁言') === -1;
    });
  }
  return root;
};
/* IMGO_GROUP_MUTE_SETTINGS_END */
