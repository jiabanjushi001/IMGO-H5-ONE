/* IMGO_GROUP_INVITE_SETTINGS_BEGIN */
var imgoGroupInviteSettingsRender = J;

function imgoGroupInviteVNodeText(node) {
  if (node == null) return '';
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  var text = node.text == null ? '' : String(node.text);
  if (Array.isArray(node.children)) {
    text += node.children.map(imgoGroupInviteVNodeText).join('');
  }
  return text;
}

J = function () {
  var root = imgoGroupInviteSettingsRender.call(this);
  var role = Number(this.groupInfo && this.groupInfo.isJoin);
  if (role !== 1 && root && Array.isArray(root.children)) {
    root.children = root.children.filter(function (child) {
      return imgoGroupInviteVNodeText(child).indexOf('管理员邀请') === -1;
    });
  }
  return root;
};
/* IMGO_GROUP_INVITE_SETTINGS_END */
