/* IMGO_GROUP_INVITE_PERMISSION_BEGIN */
var imgoGroupInviteOriginalMethods = component.methods || {};
var imgoGroupInviteOriginalOpen = imgoGroupInviteOriginalMethods.openAddGroupUser;
var imgoGroupInviteOriginalManage = imgoGroupInviteOriginalMethods.manageGroup;

function imgoCanInviteGroupMember(instance) {
  var chat = instance && instance.currentChat || {};
  var role = Number(chat.role);
  var managerInvite = Number(chat.setting && chat.setting.manager_invite);
  return role === 1 || role === 2 && managerInvite === 1;
}

function imgoRejectGroupInvite(instance) {
  if (instance && instance.$message && typeof instance.$message.error === 'function') {
    instance.$message.error('只有群主或群管理员可以邀请成员');
  }
}

component.methods = Object.assign({}, imgoGroupInviteOriginalMethods, {
  openAddGroupUser: function () {
    if (!imgoCanInviteGroupMember(this)) {
      imgoRejectGroupInvite(this);
      return;
    }
    if (typeof imgoGroupInviteOriginalOpen === 'function') {
      return imgoGroupInviteOriginalOpen.apply(this, arguments);
    }
  },
  manageGroup: function (userIds, mode) {
    if (Number(mode) === 0 && !imgoCanInviteGroupMember(this)) {
      this.createChatBox = false;
      imgoRejectGroupInvite(this);
      return;
    }
    if (typeof imgoGroupInviteOriginalManage === 'function') {
      return imgoGroupInviteOriginalManage.apply(this, arguments);
    }
  }
});
/* IMGO_GROUP_INVITE_PERMISSION_END */
