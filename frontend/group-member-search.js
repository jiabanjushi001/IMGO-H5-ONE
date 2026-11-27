/* IMGO_GROUP_MEMBER_SEARCH_BEGIN */
var imgoGroupMemberSearchOriginalData = component.data;
var imgoGroupMemberSearchOriginalComputed = component.computed || {};
var imgoGroupMemberSearchOriginalMethods = component.methods || {};
var imgoGroupMemberSearchOriginalLoad = imgoGroupMemberSearchOriginalMethods.getGroupUserList;

function imgoGroupMemberSearchText(value) {
  return String(value == null ? "" : value).trim().toLowerCase();
}

function imgoGroupMemberMatches(member, keyword) {
  var user = member && member.userInfo ? member.userInfo : {};
  return [user.displayName, user.account, user.realname, user.nickname].some(function (value) {
    return imgoGroupMemberSearchText(value).indexOf(keyword) !== -1;
  });
}

component.data = function () {
  var state = typeof imgoGroupMemberSearchOriginalData === "function"
    ? imgoGroupMemberSearchOriginalData.apply(this, arguments)
    : {};
  state = state || {};
  state.imgoGroupMemberKeyword = "";
  state.imgoGroupMemberSearchGroupId = "";
  return state;
};

component.computed = Object.assign({}, imgoGroupMemberSearchOriginalComputed, {
  imgoFilteredGroupUsers: function () {
    var members = Array.isArray(this.groupUser) ? this.groupUser : [];
    var keyword = imgoGroupMemberSearchText(this.imgoGroupMemberKeyword);
    if (!keyword) return members;
    return members.filter(function (member) {
      return imgoGroupMemberMatches(member, keyword);
    });
  }
});

component.methods = Object.assign({}, imgoGroupMemberSearchOriginalMethods, {
  getGroupUserList: function (groupId) {
    var nextGroupId = String(groupId == null ? "" : groupId);
    if (this.imgoGroupMemberSearchGroupId !== nextGroupId) {
      this.imgoGroupMemberKeyword = "";
    }
    this.imgoGroupMemberSearchGroupId = nextGroupId;
    if (typeof imgoGroupMemberSearchOriginalLoad === "function") {
      return imgoGroupMemberSearchOriginalLoad.apply(this, arguments);
    }
  }
});
/* IMGO_GROUP_MEMBER_SEARCH_END */
