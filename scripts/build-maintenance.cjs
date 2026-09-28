// Reproducible adapter for the supplied Vue 2 webpack distribution.
// Preserve all original routes and notice-management components.
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const root = path.resolve(__dirname, '..');
const read = p => fs.readFileSync(path.join(root, p), 'utf8');
const write = (p, v) => fs.writeFileSync(path.join(root, p), v);
const panel = read('frontend/maintenance-panel.js').replace('export default', 'const ImgoMaintenancePanel =');
const chunk = 'public/assets/js/585.dea7864f.js';
let text = read(chunk);
const begin = '/* IMGO_MAINTENANCE_BEGIN */', end = '/* IMGO_MAINTENANCE_END */';
if (text.includes(begin)) text = text.slice(0, text.indexOf(begin) - 1) + text.slice(text.indexOf(end) + end.length);
text = text.replace(/d=r\.exports;*(?=},5080:)/, 'd=r.exports');
// Notice rows keep their editor click target; deletion is a separate button.
const noticeTime = 'a("div",{staticClass:"c-999"},[t._v(t._s(s.create_time))])';
const noticeActions = 'a("div",{staticClass:"imgo-notice-actions"},[a("span",{staticClass:"c-999"},[t._v(t._s(s.create_time))]),a("el-button",{attrs:{type:"text",size:"small"},staticClass:"imgo-notice-delete",on:{click:function(e){e.stopPropagation();return t.imgoDeleteNotice(s)}}},[t._v("删除")])],1)';
if (!text.includes(noticeActions)) {
 if (!text.includes(noticeTime)) throw Error('Notice row changed; review delete adapter');
 text = text.replace(noticeTime, noticeActions);
}
const anchor = 'd=r.exports';
if (!text.includes(anchor)) throw Error('Management chunk changed; review adapter before rebuilding');
const overview = read('frontend/overview-chart.js') + '\n' + read('frontend/overview-panel.js');
const bankPanel = read('frontend/bank-panel.js');
const financeOrders = read('frontend/finance-orders.js');
const rolePanel = read('frontend/role-panel.js');
const auditPanel = read('frontend/audit-panel.js');

// The management group picker must send the selected group id so existing
// members are excluded while an agent adds people from their own team.
let managementGroups = read('public/assets/js/173.9e08cf23.js');
const managementGroupOwner = 'e("p",{staticClass:"chat-message c-999"},[t._v(" 创建人："+t._s(s.owner_id_info.realname))])';
const managementGroupNumber = 'e("p",{staticClass:"chat-message c-999"},[t._v("群号："+t._s(s.group_id)+" · 创建人："+t._s(s.owner_id_info.realname))])';
if (!managementGroups.includes(managementGroupNumber)) {
 if (!managementGroups.includes(managementGroupOwner)) throw Error('Management group list metadata anchor changed');
 managementGroups = managementGroups.replace(managementGroupOwner, managementGroupNumber);
}
const managementGroupNumberSwitch = 'e("div",{staticClass:"imgo-group-number-join",on:{click:function(t){t.stopPropagation()}}},[e("div",{staticClass:"imgo-group-number-join-copy"},[e("span",[t._v("群号加入")]),e("small",[t._v("输入群号即可加入私有群")])]),e("el-switch",{attrs:{value:1==s.number_join,disabled:!s.can_set_number_join},on:{change:function(e){return t.setNumberJoin(s,e)}}})],1)';
if (!managementGroups.includes(managementGroupNumberSwitch)) {
 if (!managementGroups.includes(managementGroupNumber)) throw Error('Management group number switch anchor changed');
 managementGroups = managementGroups.replace(managementGroupNumber, managementGroupNumber + ',' + managementGroupNumberSwitch);
}
const managementGroupDialog = 'attrs:{visible:t.createChatBox,title:t.dialogTitle,isAdd:t.isAdd,userIds:t.userIds}';
const scopedManagementGroupDialog = 'attrs:{visible:t.createChatBox,title:t.dialogTitle,isAdd:t.isAdd,userIds:t.userIds,groupId:t.active}';
if (!managementGroups.includes(scopedManagementGroupDialog)) {
 if (!managementGroups.includes(managementGroupDialog)) throw Error('Management group picker anchor changed');
 managementGroups = managementGroups.replace(managementGroupDialog, scopedManagementGroupDialog);
}
const addGroupMemberButton = 'e("el-button",{attrs:{plain:"",round:""},on:{click:t.openAddUser}},[t._v("添加成员")])';
const broadcastGroupButton = 'e("el-button",{attrs:{type:"primary",plain:"",round:""},on:{click:t.openBroadcast}},[t._v("群发")])';
if (!managementGroups.includes(broadcastGroupButton)) {
 if (!managementGroups.includes(addGroupMemberButton)) throw Error('Management group member button changed');
 managementGroups = managementGroups.replace(addGroupMemberButton, broadcastGroupButton+','+addGroupMemberButton);
}
// Group monitoring is an administrative view. Without this prop ChatRecord
// calls the ordinary member history endpoint, which rejects administrators who
// are not members of the selected group (including the system super admin).
const ordinaryGroupMonitor = 'e("ChatRecord",{key:t.componentKey,attrs:{contact:t.currentChat}})';
const managedGroupMonitor = 'e("ChatRecord",{key:t.componentKey,attrs:{contact:t.currentChat,manage:!0}})';
if (!managementGroups.includes(managedGroupMonitor)) {
 if (!managementGroups.includes(ordinaryGroupMonitor)) throw Error('Management group monitor anchor changed');
 managementGroups = managementGroups.replace(ordinaryGroupMonitor, managedGroupMonitor);
}
const groupDialogAnchor = 'e("Group",{attrs:{visible:t.createChatBox,title:t.dialogTitle,isAdd:t.isAdd,userIds:t.userIds,groupId:t.active}';
const broadcastDialog = 'e("el-dialog",{attrs:{title:"群发消息",visible:t.broadcastVisible,width:"520px","append-to-body":"","close-on-click-modal":!1},on:{"update:visible":function(e){t.broadcastVisible=e}}},[e("div",{staticClass:"imgo-broadcast-hint"},[t._v("由导师设置的客服向当前群成员发送私聊；超级管理员会按成员所属导师分别发送。")]),e("el-input",{attrs:{type:"textarea",rows:6,maxlength:2048,"show-word-limit":"",placeholder:"请输入群发内容"},model:{value:t.broadcastContent,callback:function(e){t.broadcastContent=e},expression:"broadcastContent"}}),e("span",{attrs:{slot:"footer"},slot:"footer"},[e("el-button",{attrs:{disabled:t.broadcastSending},on:{click:t.closeBroadcast}},[t._v("取消")]),e("el-button",{attrs:{type:"primary",loading:t.broadcastSending},on:{click:t.submitBroadcast}},[t._v("确认群发")])],1)],1),';
if (!managementGroups.includes('staticClass:"imgo-broadcast-hint"')) {
 if (!managementGroups.includes(groupDialogAnchor)) throw Error('Management group dialog anchor changed');
 managementGroups = managementGroups.replace(groupDialogAnchor, broadcastDialog+groupDialogAnchor);
}
const groupBroadcastComponentAnchor = 'p=c,d=s(1001)';
if (!managementGroups.includes('/* IMGO_GROUP_BROADCAST_BEGIN */')) {
 if (!managementGroups.includes(groupBroadcastComponentAnchor)) throw Error('Management group component anchor changed');
 managementGroups = managementGroups.replace(groupBroadcastComponentAnchor, 'p=(function(component){/* IMGO_GROUP_BROADCAST_BEGIN */'+read('frontend/group-broadcast.js')+'/* IMGO_GROUP_BROADCAST_END */return component;})(c),d=s(1001)');
}
const managementGroupsHash = crypto.createHash('sha256').update(managementGroups).digest('hex').slice(0,12);
write(`public/assets/js/173.imgo${managementGroupsHash}.js`, managementGroups);
const extension = `;${begin}\n${read('frontend/notice-actions.js')};\n${panel};\n${overview};\n${bankPanel};\n${financeOrders};\n${rolePanel};\n${auditPanel};\nconst LegacyManagement = d; d = { name: 'ImgoManagement', render(h) { const bank = this.$route.path === '/manage/bank', finance = this.$route.path.startsWith('/manage/finance/'), role = this.$route.path === '/manage/role', audit = this.$route.path === '/manage/audit', superAdmin=Number((this.$store.state.userInfo||{}).user_id)===1; return h('div', {class: 'imgo-management'}, [audit ? h(ImgoAuditPanel) : role ? h(ImgoRolePanel) : finance ? h(ImgoFinanceShell) : bank ? h(ImgoBankPanel) : h(ImgoOverview, superAdmin?[h(LegacyManagement),h(ImgoMaintenancePanel)]:[])]); } };\n${end}`;
text = text.replace(anchor, anchor + extension);
write(chunk, text);
let app = read('public/assets/js/app.85372e4e.js');
const accountLoginSave = ',window.ImgoAccountVault&&window.ImgoAccountVault.save({authToken:i.authToken,sessionId:i.sessionId,userInfo:i.userInfo})';
app = app.replaceAll(accountLoginSave, '');
const accountLoginAnchor = 't("SET_AUTH",i),t("SET_USERINFO",i.userInfo)';
if (!app.includes(accountLoginAnchor)) throw Error('Account vault login anchor changed');
app = app.replace(accountLoginAnchor, accountLoginAnchor + accountLoginSave);
const accountLogoutRemove = 'window.ImgoAccountVault&&window.ImgoAccountVault.removeCurrent(),';
app = app.replaceAll(accountLogoutRemove, '');
const accountLogoutAnchor = 'Mi.logoutAPI().then((()=>{o().rm("authToken")';
if (!app.includes(accountLogoutAnchor)) throw Error('Account vault logout anchor changed');
app = app.replace(accountLogoutAnchor, 'Mi.logoutAPI().then((()=>{' + accountLogoutRemove + 'o().rm("authToken")');
// Lemon IMUI stores address-book selection in a separate id but then reads the
// message-list id. When no recent chat is active it returns before emitting the
// selected group. Opening a contact is a chat action, so switch channels first.
const addressBookContactSwitch = 'this.activeSidebar==ke?this.currentContactIdSidebarContact=e:this.currentContactId=e,this.currentContactId';
const messageContactSwitch = 'this.activeSidebar==ke&&this.changeMenu(we),this.currentContactId=e,this.currentContactId';
if (!app.includes(messageContactSwitch)) {
 if (!app.includes(addressBookContactSwitch)) throw Error('Lemon IMUI contact switch changed');
 app = app.replace(addressBookContactSwitch, messageContactSwitch);
}
// LemonEditor creates exactly one mention picker for the current device. Its
// upstream initializer always customizes the desktop picker, which is absent
// on H5 and crashes while reading checkboxElm.querySelector().
const unsafeMentionDialogLabels = 't.chatArea.revisePCCheckDialogLabel({title:"选择要@的人",searchPlaceholder:"搜素人员名称",searchEmptyLabel:"没有匹配到任何结果",userTagTitle:"成员列表",checkAllLabel:"全选",confirmLabel:"确定",cancelLabel:"取消"})';
const responsiveMentionDialogLabels = 't.chatArea.checkboxElm?t.chatArea.revisePCCheckDialogLabel({title:"选择要@的人",searchPlaceholder:"搜素人员名称",searchEmptyLabel:"没有匹配到任何结果",userTagTitle:"成员列表",checkAllLabel:"全选",confirmLabel:"确定",cancelLabel:"取消"}):t.chatArea.dialogH5Elm&&t.chatArea.reviseH5DialogLabel({title:"选择要@的人",searchPlaceholder:"搜素人员名称",confirmLabel:"确定",cancelLabel:"收起"})';
if (!app.includes(responsiveMentionDialogLabels)) {
 if (!app.includes(unsafeMentionDialogLabels)) throw Error('LemonEditor mention-dialog initializer changed');
 app = app.replace(unsafeMentionDialogLabels, responsiveMentionDialogLabels);
}
// Quick replies belong to the chat composer, not the member broadcast dialog.
app = app.replace('saveQuickReplies:t=>Ti({url:"/manage/User/saveQuickReplies",method:"post",data:t}),quickReplies:t=>Ti({url:"/manage/User/quickReplies",method:"post",data:t}),', '');
if (!app.includes('broadcast:t=>Ti({url:"/manage/User/broadcast"')) {
 const userAPIAnchor = 'const Os={setInviteCode:';
 if (!app.includes(userAPIAnchor)) throw Error('Management user API anchor changed');
 app = app.replace(userAPIAnchor, 'const Os={broadcast:t=>Ti({url:"/manage/User/broadcast",method:"post",data:t}),setInviteCode:');
}
if (!app.includes('broadcastOptions:t=>Ti({url:"/manage/User/broadcastOptions"')) {
 const userAPIAnchor = 'const Os={broadcast:';
 if (!app.includes(userAPIAnchor)) throw Error('Management user broadcast API anchor changed');
 app = app.replace(userAPIAnchor, 'const Os={broadcastOptions:t=>Ti({url:"/manage/User/broadcastOptions",method:"post",data:t}),broadcast:');
}
if (!app.includes('broadcast:t=>Ti({url:"/manage/Group/broadcast"')) {
 const groupAPIAnchor = 'const Ps={getGroupList:';
 if (!app.includes(groupAPIAnchor)) throw Error('Management group API anchor changed');
 app = app.replace(groupAPIAnchor, 'const Ps={broadcast:t=>Ti({url:"/manage/Group/broadcast",method:"post",data:t}),getGroupList:');
}
if (!app.includes('setNumberJoin:t=>Ti({url:"/manage/Group/setNumberJoin"')) {
 const groupAPIAnchor = 'const Ps={broadcast:';
 if (!app.includes(groupAPIAnchor)) throw Error('Management group number-join API anchor changed');
 app = app.replace(groupAPIAnchor, 'const Ps={setNumberJoin:t=>Ti({url:"/manage/Group/setNumberJoin",method:"post",data:t}),broadcast:');
}
if (!/173:"(?:9e08cf23|imgo[a-f0-9]+)"/.test(app)) throw Error('Management group chunk hash map changed');
app = app.replace(/173:"(?:9e08cf23|imgo[a-f0-9]+)"/, `173:"imgo${managementGroupsHash}"`);
const groupPickerTarget = 'this.groupId&&2==this.isAdd&&(t.group_id=this.groupId)';
const scopedGroupPickerTarget = 'this.groupId&&(0==this.isAdd||2==this.isAdd)&&(t.group_id=this.groupId)';
if (!app.includes(scopedGroupPickerTarget)) {
 if (!app.includes(groupPickerTarget)) throw Error('Group picker target anchor changed');
 app = app.replace(groupPickerTarget, scopedGroupPickerTarget);
}
const ordinaryManageGuard = 'p.Message.error("您没有权限访问该页面"),i(!1),Fi().done()';
if (app.includes(ordinaryManageGuard)) {
 app = app.replace(ordinaryManageGuard, 'p.Message.error("您没有权限访问该页面"),i("/chat"),Fi().done()');
}
// Install invitation adapters before Vue normalizes the component options.
for (const [name, variable, source] of [
 ['CHAT', 'Ae', 'frontend/friend-realtime.js'],
 ['LIST', 've', 'frontend/friend-apply-realtime.js']
]) {
 const start = '/* IMGO_FRIEND_' + name + '_BEGIN */';
 const end = '/* IMGO_FRIEND_' + name + '_END */';
 const generated = '(function(component){' + start;
 if (app.includes(generated)) {
  const from = app.indexOf(generated), to = app.indexOf(end, from);
  if (to < 0) throw Error('Incomplete invitation adapter');
  app = app.slice(0, from) + variable + app.slice(to + end.length + ('})(' + variable + ')').length);
 }
 const anchor = name === 'CHAT' ? '_e=Ae,' : 'be=ve,';
 if (!app.includes(anchor)) throw Error('Invitation component anchor changed: ' + name);
 app = app.replace(anchor, anchor.split('=')[0] + '=(function(component){' + start + '\n' + read(source) + (name === 'CHAT' ? '\n' + read('frontend/chat-profile.js') + '\n' + read('frontend/chat-context-remark.js') + '\n' + read('frontend/group-avatar-chat.js') + '\n' + read('frontend/chat-quick-replies.js') + '\n' + read('frontend/chat-join-group.js') + '\n' + read('frontend/chat-account-switcher.js') + '\n' + read('frontend/contact-search.js') + '\n' + read('frontend/group-mute-chat.js') + '\n' + read('frontend/mobile-chat.js') + '\n' + read('frontend/socket-recovery.js') : '') +
  '\nreturn component;' + end + '})(' + variable + '),');
}
const legacyUploadLimit = 'if(i.size>1024*this.globalConfig.fileUpload.size*1024)return s.removeMessage(t.id),this.$message.error("上传的内容不等大于"+this.globalConfig.fileUpload.size+"MB！");';
const mediaUploadLimit = 'if(i.size>1024*imgoUploadLimitMB(t,this.globalConfig.fileUpload)*1024)return s.removeMessage(t.id),this.$message.error("上传的内容不能大于"+imgoUploadLimitMB(t,this.globalConfig.fileUpload)+"MB！");';
if (!app.includes(mediaUploadLimit)) {
 if (!app.includes(legacyUploadLimit)) throw Error('Chat upload-size guard changed');
 app = app.replace(legacyUploadLimit, mediaUploadLimit);
}
const contactSearchInput = 'attrs:{placeholder:"\u641c\u7d22\u8054\u7cfb\u4eba","prefix-icon":"el-icon-search"}';
const improvedContactSearchInput = 'attrs:{placeholder:"\u641c\u7d22\u8d26\u53f7\u3001\u6635\u79f0\u6216\u7fa4聊","prefix-icon":"el-icon-search",autocomplete:"one-time-code",name:"imgo-contact-search"}';
if (!app.includes(improvedContactSearchInput)) {
 if (!app.includes(contactSearchInput)) throw Error('Contact search input anchor changed');
 app = app.replace(contactSearchInput, improvedContactSearchInput);
}
const emptyContactSearch = '0==t.searchList.length?e("div",{staticStyle:{margin:"20px"},attrs:{align:"center"}},[t._v(" \u6682无 ")]):t._e()';
const improvedEmptyContactSearch = 't.keywords&&0==t.searchList.length?e("div",{staticStyle:{margin:"20px"},attrs:{align:"center"}},[t._v(" \u672a找到联系人 ")]):t._e()';
if (!app.includes(improvedEmptyContactSearch)) {
 if (!app.includes(emptyContactSearch)) throw Error('Contact search empty-state anchor changed');
 app = app.replace(emptyContactSearch, improvedEmptyContactSearch);
}
// Install the socket lifecycle adapter before Vue normalizes the component.
const socketBegin = '/* IMGO_SOCKET_RELIABILITY_BEGIN */', socketEnd = '/* IMGO_SOCKET_RELIABILITY_END */';
const socketGenerated = '(function(component){' + socketBegin;
if (app.includes(socketGenerated)) {
 const from = app.indexOf(socketGenerated), to = app.indexOf(socketEnd, from);
 if (to < 0) throw Error('Incomplete socket reliability adapter');
 app = app.slice(0, from) + 'G' + app.slice(to + socketEnd.length + '})(G)'.length);
}
const socketComponentIndex = app.indexOf('G={name:"socket"');
const socketAnchorIndex = app.indexOf('K=G,z=', socketComponentIndex);
if (socketComponentIndex < 0 || socketAnchorIndex < 0) throw Error('Socket component anchor changed');
const socketReplacement = 'K=(function(component){' + socketBegin + '\n' + read('frontend/socket-reliability.js') + '\nreturn component;' + socketEnd + '})(G),z=';
app = app.slice(0, socketAnchorIndex) + socketReplacement + app.slice(socketAnchorIndex + 'K=G,z='.length);
if (!app.includes('Rs.getQuickRepliesAPI=')) {
 const imApiAnchor = 'Rs.forwardMessageAPI=';
 if (!app.includes(imApiAnchor)) throw Error('Chat API anchor changed');
 app = app.replace(imApiAnchor, 'Rs.getQuickRepliesAPI=t=>Ti({url:"enterprise/im/getQuickReplies",method:"post",data:t}),Rs.saveQuickRepliesAPI=t=>Ti({url:"enterprise/im/saveQuickReplies",method:"post",data:t}),' + imApiAnchor);
}
if (!app.includes('Rs.joinGroupAPI=')) {
 const imApiAnchor = 'Rs.getQuickRepliesAPI=';
 if (!app.includes(imApiAnchor)) throw Error('Join group API anchor changed');
 app = app.replace(imApiAnchor, 'Rs.joinGroupAPI=t=>Ti({url:"enterprise/group/joinGroup",method:"post",data:t}),' + imApiAnchor);
}
const joinGroupMenu = 't.globalConfig.chatInfo.groupChat?e("el-dropdown-item",{attrs:{command:"joinGroup"}},[t._v("添加群聊")]):t._e(),';
app = app.replaceAll(joinGroupMenu, '');
const createGroupMenu = 't.globalConfig.chatInfo.groupChat?e("el-dropdown-item",{attrs:{command:"addGroup"}},[t._v("创建群聊")]):t._e()';
if (!app.includes(createGroupMenu)) throw Error('Chat plus menu anchor changed');
app = app.replace(createGroupMenu, joinGroupMenu + createGroupMenu);
const joinGroupDialog = 'e("imgo-join-group",{ref:"ImgoJoinGroup",on:{joined:t.imgoJoinedGroup}}),';
app = app.replaceAll(joinGroupDialog, '');
// Sender names are essential in group conversations. Keep the renderer enabled
// even when an older saved client preference hid message names.
const hiddenMessageNameProp = '"hide-message-name":t.setting.hideMessageName';
const visibleMessageNameProp = '"hide-message-name":!1';
if (!app.includes(visibleMessageNameProp)) {
 if (!app.includes(hiddenMessageNameProp)) throw Error('Chat message-name visibility anchor changed');
 app = app.replace(hiddenMessageNameProp, visibleMessageNameProp);
}
// Incoming group messages identify the sender as account(nickname). The
// current user's reversed messages intentionally omit this redundant label.
const plainMessageSender = '0==this.hideName&&e("span",{on:{click:function(e){t._emitClick(e,"displayName")}}},[s.displayName])';
const qualifiedMessageSender = '0==this.hideName&&!this.reverse&&e("span",{on:{click:function(e){t._emitClick(e,"displayName")}}},[s.account&&s.displayName&&s.account!==s.displayName?s.account+"("+s.displayName+")":s.account||s.displayName])';
if (!app.includes(qualifiedMessageSender)) {
 if (!app.includes(plainMessageSender)) throw Error('Chat message sender-label anchor changed');
 app = app.replace(plainMessageSender, qualifiedMessageSender);
}
const chatGroupDialog = 'e("Group",{attrs:{visible:t.createChatBox,title:t.dialogTitle,isAdd:t.isAdd,userIds:t.userIds,groupId:t.group_id}';
if (!app.includes(chatGroupDialog)) throw Error('Chat group dialog anchor changed');
app = app.replace(chatGroupDialog, joinGroupDialog + chatGroupDialog);
const quickReplyFooter = 'e("imgo-chat-quick-replies",{on:{select:t.imgoUseQuickReply}}),';
app = app.replaceAll(quickReplyFooter, '');
const editorFooterSpacer = 't.quote?e("div",{staticClass:"message-quote cur-handle mr-10 lz-flex lz-space-between lz-align-items-center"},[e("div",{staticClass:"text-overflow"},[t._v(t._s(t.quote.content))]),e("div",{staticClass:"el-icon-close",on:{click:function(e){return t.closeQuote()}}})]):t._e(),e("div"),e("div",[t._v(t._s(1==t.setting.sendKey?"使用 Ctrl + Enter 换行":"使用 Ctrl + Enter 发送消息"))])';
const editorFooterQuick = 't.quote?e("div",{staticClass:"message-quote cur-handle mr-10 lz-flex lz-space-between lz-align-items-center"},[e("div",{staticClass:"text-overflow"},[t._v(t._s(t.quote.content))]),e("div",{staticClass:"el-icon-close",on:{click:function(e){return t.closeQuote()}}})]):t._e(),'+quickReplyFooter+'e("div"),e("div",[t._v(t._s(1==t.setting.sendKey?"使用 Ctrl + Enter 换行":"使用 Ctrl + Enter 发送消息"))])';
if (!app.includes(editorFooterSpacer)) throw Error('Chat editor footer anchor changed');
app = app.replace(editorFooterSpacer, editorFooterQuick);
// Group avatar settings adapter, installed before Vue normalizes the component.
const groupAvatarStart = '/* IMGO_GROUP_AVATAR_BEGIN */', groupAvatarEnd = '/* IMGO_GROUP_AVATAR_END */';
const groupAvatarWrapper = '(function(component){' + groupAvatarStart;
if (app.includes(groupAvatarWrapper)) {
 const from = app.indexOf(groupAvatarWrapper), to = app.indexOf(groupAvatarEnd, from);
 if (to < 0) throw Error('Incomplete group avatar adapter');
 app = app.slice(0, from) + 'W' + app.slice(to + groupAvatarEnd.length + '})(W)'.length);
}
if (!app.includes('Z=W,X=')) throw Error('Group settings component changed');
app = app.replace('Z=W,X=', 'Z=(function(component){' + groupAvatarStart + '\n' + read('frontend/group-avatar.js') + '\n' + read('frontend/group-mute-settings.js') + '\nreturn component;' + groupAvatarEnd + '})(W),X=');
if (!app.includes('Rs.editGroupAvatarAPI=')) app = app.replace('Rs.editGroupNameAPI=', 'Rs.editGroupAvatarAPI=t=>Ti({url:"enterprise/group/editGroupAvatar",method:"post",data:t}),Rs.editGroupNameAPI=');
app = app.replace('i.is_group&&1==t.currentChat.role?', 'i.is_group&&(1==t.currentChat.role||2==t.currentChat.role||Number(t.userInfo.user_id)===1||Number(t.userInfo.role)>0)?');
app = app.replace('on:{changeOwner:t.changeOwner}', 'on:{changeOwner:t.changeOwner,avatarChanged:t.imgoGroupAvatarChanged}');
// HTTP bind success means this socket is authenticated, including after reconnect.
const invitationBind = 'this.websocketSend({type:"bindUid",user_id:i.user_id,token:s}),console.log';
const invitationBound = 'this.websocketSend({type:"bindUid",user_id:i.user_id,token:s}),this.$store.commit("catchSocketAction",{type:"friendApplyChanged"}),console.log';
if (!app.includes(invitationBound)) {
 if (!app.includes(invitationBind)) throw Error('Socket bind callback changed');
 app = app.replace(invitationBind, invitationBound);
}
const privateTitle = '0==t.is_group?e("span",{staticClass:"displayName"}';
const adminProfileTitle = '0==t.is_group&&(Number(t.userInfo.user_id)===1||Number(t.userInfo.role)>0)?e("imgo-chat-profile",{attrs:{contact:i}}):';
const profileTitle = '0==t.is_group?e("imgo-chat-profile",{attrs:{contact:i}}):';
const authorizedProfileTitle = '0==t.is_group&&(Number(t.userInfo.user_id)===1||Number(t.userInfo.admin_role_id)>0)?e("imgo-chat-profile",{attrs:{contact:i}}):';
if (app.includes(profileTitle)) {
 app = app.replace(profileTitle, authorizedProfileTitle);
} else if (app.includes(adminProfileTitle)) {
 app = app.replace(adminProfileTitle, authorizedProfileTitle);
} else if (!app.includes(authorizedProfileTitle)) {
 if (!app.includes(privateTitle)) throw Error('Private chat title anchor changed');
 app = app.replace(privateTitle, authorizedProfileTitle+privateTitle);
}
app = app.replace('staticClass:"chat-box"', 'staticClass:"chat-box imgo-chat-design"');
const mobileChatRoot = 'e("div",{staticClass:"chat-box imgo-chat-design",class:{"imgo-mobile-conversation":t.imgoMobileConversation}},[';
if (!app.includes(mobileChatRoot)) {
 const chatRoot = 'e("div",{staticClass:"chat-box imgo-chat-design"},[';
 if (!app.includes(chatRoot)) throw Error('Chat root renderer changed');
 app = app.replace(chatRoot, mobileChatRoot);
}
const accountSwitcherDialog = 'e("imgo-account-switcher",{ref:"ImgoAccountSwitcher"}),';
app = app.replaceAll(accountSwitcherDialog, '');
if (!app.includes(mobileChatRoot)) throw Error('Account switcher chat root anchor changed');
app = app.replace(mobileChatRoot, mobileChatRoot + accountSwitcherDialog);
const accountSwitcherMenu = '{name:"accountSwitch",title:"切换账号",unread:0,render:t=>e("i",{class:"el-icon-user"}),click:()=>{this.$refs.ImgoAccountSwitcher&&this.$refs.ImgoAccountSwitcher.open()},isBottom:!0},';
app = app.replaceAll(accountSwitcherMenu, '');
const accountSwitcherMenuAnchor = '{name:"setting",title:"设置",unread:0,render:t=>e("i",{class:"el-icon-setting"}),renderContainer:()=>e(Gt),isBottom:!0}';
if (!app.includes(accountSwitcherMenuAnchor)) throw Error('Account switcher menu anchor changed');
app = app.replace(accountSwitcherMenuAnchor, accountSwitcherMenu + accountSwitcherMenuAnchor);
const mobileChatTitle = 'e("div",{staticClass:"message-title-box"},[e("button",{staticClass:"imgo-mobile-chat-back",attrs:{type:"button","aria-label":"\u8fd4\u56de\u4f1a\u8bdd\u5217\u8868"},on:{click:t.imgoMobileBack}},[e("i",{staticClass:"el-icon-arrow-left"})]),e("div",[';
if (!app.includes(mobileChatTitle)) {
 const chatTitle = 'e("div",{staticClass:"message-title-box"},[e("div",[';
 if (!app.includes(chatTitle)) throw Error('Chat title renderer changed');
 app = app.replace(chatTitle, mobileChatTitle);
}
if (!app.includes('setRemark:t=>Ti(')) app=app.replace('const Os={', 'const Os={setRemark:t=>Ti({url:"/manage/User/setRemark",method:"post",data:t}),');
const previewBegin = '/* IMGO_PREVIEW_BEGIN */', previewEnd = '/* IMGO_PREVIEW_END */';
const previewWrapper = '(function(component){' + previewBegin;
if (app.includes(previewWrapper)) {
 const from = app.indexOf(previewWrapper), to = app.indexOf(previewEnd, from);
 if (to < 0) throw Error('Incomplete preview adapter');
 app = app.slice(0,from) + 'A' + app.slice(to + previewEnd.length + '})(A)'.length);
}
if (!app.includes('_=A,E=')) throw Error('Preview component anchor changed');
app = app.replace('_=A,E=', '_=(function(component){'+previewBegin+'\n'+read('frontend/file-preview.js')+'\nreturn component;'+previewEnd+'})(A),E=');
const apiAnchor = 'const Ns={getTaskList:';
if (!app.includes('setTaskConfig:t=>Ti({url:"/manage/Task/setTaskConfig"')) {
 if (!app.includes(apiAnchor)) throw Error('Task API bundle changed');
 app = app.replace(apiAnchor, 'const Ns={setTaskConfig:t=>Ti({url:"/manage/Task/setTaskConfig",method:"post",data:t}),getTaskList:');
}
if (!app.includes('getOverview:t=>Ti(')) {
 app = app.replace('const Ns={', 'const Ns={getOverview:t=>Ti({url:"/manage/index/overview",method:"post",data:t}),');
}
if (!app.includes('path:"/manage/bank",name:"bank"')) {
 const nextRoute = '},{path:"/manage/setting",name:"setting"';
 if (!app.includes(nextRoute)) throw Error('Management route list changed');
 app = app.replace(nextRoute, '},{path:"/manage/bank",name:"bank",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"绑卡",icon:"el-icon-bank-card"}'+nextRoute);
}
if (!app.includes('path:"/manage/role",name:"role"')) {
 const nextRoute = '},{path:"/manage/setting",name:"setting"';
 if (!app.includes(nextRoute)) throw Error('Role route anchor changed');
 app = app.replace(nextRoute, '},{path:"/manage/role",name:"role",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"角色",icon:"el-icon-lock"}'+nextRoute);
}
if (!app.includes('path:"/manage/audit",name:"audit"')) {
 const nextRoute = '},{path:"/manage/role",name:"role"';
 if (!app.includes(nextRoute)) throw Error('Audit route anchor changed');
 app = app.replace(nextRoute, '},{path:"/manage/audit",name:"audit",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"日志",icon:"el-icon-document"}'+nextRoute);
}
app = app.replace('path:"/manage/audit",name:"audit",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"操作日志",icon:"el-icon-document"}', 'path:"/manage/audit",name:"audit",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"日志",icon:"el-icon-document"}');
app = app.replace('path:"/manage/role",name:"role",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"角色权限",icon:"el-icon-lock"}', 'path:"/manage/role",name:"role",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"角色",icon:"el-icon-lock"}');
const oldWalletRoute = '},{path:"/manage/wallet",name:"wallet",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"钱包",icon:"el-icon-money"}';
app = app.replace(oldWalletRoute, '');
const financeRouteAnchor = '},{path:"/manage/bank",name:"bank"';
const financeRoutes = '},{path:"/manage/finance/recharges",name:"finance-recharges",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"充值订单",icon:"el-icon-document-add"}},{path:"/manage/finance/withdrawals",name:"finance-withdrawals",component:()=>i.e(585).then(i.bind(i,4585)),meta:{title:"提现订单",icon:"el-icon-document-remove"}';
if (!app.includes('path:"/manage/finance/recharges"')) {
 if (!app.includes(financeRouteAnchor)) throw Error('Finance route anchor changed');
 app = app.replace(financeRouteAnchor, financeRoutes + financeRouteAnchor);
}
// Existing bookmarks to the removed wallet page land on finance orders.
if (!app.includes('path:"/manage/wallet",redirect:')) {
 if (!app.includes(financeRouteAnchor)) throw Error('Legacy wallet redirect anchor changed');
 app = app.replace(financeRouteAnchor, '},{path:"/manage/wallet",redirect:"/manage/finance/withdrawals"'+financeRouteAnchor);
}
const oldWalletMenu = 'this.routes=t[0].children.filter(e=>e.path!=="/manage/wallet"||Number((this.$store.state.userInfo||{}).user_id)===1)';
const flatMenu = 'this.routes=t[0].children';
const financeMenu = 'this.routes=t[0].children.filter(e=>!e.path.startsWith("/manage/finance/")&&e.path!=="/manage/wallet"),Number((this.$store.state.userInfo||{}).user_id)===1&&this.routes.splice(1,0,{path:"/manage/finance",meta:{title:"财务",icon:"el-icon-money"},children:[{path:"/manage/finance/recharges",meta:{title:"充值订单",icon:"el-icon-document-add"}},{path:"/manage/finance/withdrawals",meta:{title:"提现订单",icon:"el-icon-document-remove"}}]})';
const rbacMenu = 'this.routes=imgoBuildAdminMenu(this.$store.state.userInfo,t[0].children),imgoEnsureAuthorizedAdminRoute(this)';
app = app.replace('meta:{title:"财务",icon:"el-icon-s-finance"}', 'meta:{title:"财务",icon:"el-icon-money"}');
if (!app.includes(rbacMenu)) {
 const currentMenu = app.includes(financeMenu) ? financeMenu : (app.includes(oldWalletMenu) ? oldWalletMenu : flatMenu);
 if (!app.includes(currentMenu)) throw Error('RBAC sidebar initialization changed');
 app = app.replace(currentMenu, rbacMenu);
}
const oldMenuItem = 'e("el-menu-item",{key:s,attrs:{index:i.path}},[e("i",{class:i.meta.icon}),e("span",{attrs:{slot:"title"},slot:"title"},[t._v(t._s(i.meta.title))])])';
const submenuItem = 'i.children?e("el-submenu",{key:s,attrs:{index:i.path}},[e("span",{slot:"title"},[e("i",{class:i.meta.icon}),e("span",[t._v(t._s(i.meta.title))])]),t._l(i.children,(function(c,n){return e("el-menu-item",{key:n,attrs:{index:c.path}},[e("i",{class:c.meta.icon}),e("span",{attrs:{slot:"title"}},[t._v(t._s(c.meta.title))])])}))],2):' + oldMenuItem;
if (!app.includes('i.children?e("el-submenu"')) {
 if (!app.includes(oldMenuItem)) throw Error('Sidebar item renderer changed');
 app = app.replace(oldMenuItem, submenuItem);
}
app = app.replace('Number(this.$store.state.userInfo.user_id)===1', 'Number((this.$store.state.userInfo||{}).user_id)===1');
// Existing builds already contain this route; keep rebuilding idempotent.
app = app.replace('meta:{title:"绑卡",icon:"el-icon-wallet"}', 'meta:{title:"绑卡",icon:"el-icon-bank-card"}');
if (!app.includes('meta:{title:"绑卡",icon:"el-icon-bank-card"}')) throw Error('Bank menu icon route changed');
if (!app.includes('const ImgoBankApi=')) {
 const apiAnchor = 'var Vs=Hs,Gs={taskApi:Ms,';
 if (!app.includes(apiAnchor)) throw Error('Admin API registry changed');
 app = app.replace(apiAnchor, 'const ImgoBankApi={index:t=>Ti({url:"/manage/bank/index",method:"post",data:t}),detail:t=>Ti({url:"/manage/bank/detail",method:"post",data:t}),edit:t=>Ti({url:"/manage/bank/edit",method:"post",data:t})};'+apiAnchor.replace('taskApi:Ms,','taskApi:Ms,bankApi:ImgoBankApi,'));
}
const walletApi = 'const ImgoWalletApi={index:t=>Ti({url:"/manage/wallet/index",method:"post",data:t}),detail:t=>Ti({url:"/manage/wallet/detail",method:"post",data:t}),account:t=>Ti({url:"/manage/wallet/account",method:"post",data:t}),credit:t=>Ti({url:"/manage/wallet/credit",method:"post",data:t}),review:t=>Ti({url:"/manage/wallet/review",method:"post",data:t}),freeze:t=>Ti({url:"/manage/wallet/freeze",method:"post",data:t}),entries:t=>Ti({url:"/manage/wallet/entries",method:"post",data:t}),recharges:t=>Ti({url:"/manage/wallet/recharges",method:"post",data:t}),recharge:t=>Ti({url:"/manage/wallet/recharge",method:"post",data:t}),withdraw:t=>Ti({url:"/manage/wallet/withdraw",method:"post",data:t})};';
if (!app.includes('const ImgoWalletApi=')) {
 const walletApiAnchor = 'var Vs=Hs,Gs={taskApi:Ms,bankApi:ImgoBankApi,';
 if (!app.includes(walletApiAnchor)) throw Error('Wallet API registry anchor changed');
 app = app.replace(walletApiAnchor, walletApi + walletApiAnchor.replace('bankApi:ImgoBankApi,', 'bankApi:ImgoBankApi,walletApi:ImgoWalletApi,'));
} else {
 app = app.replace(/const ImgoWalletApi=\{[^;]+\};/, walletApi);
}
const roleApi = 'const ImgoRoleApi={index:t=>Ti({url:"/manage/role/index",method:"post",data:t}),detail:t=>Ti({url:"/manage/role/detail",method:"post",data:t}),save:t=>Ti({url:"/manage/role/save",method:"post",data:t}),setStatus:t=>Ti({url:"/manage/role/setStatus",method:"post",data:t}),del:t=>Ti({url:"/manage/role/del",method:"post",data:t}),permissions:t=>Ti({url:"/manage/role/permissions",method:"post",data:t})};';
if (!app.includes('const ImgoRoleApi=')) {
 const roleApiAnchor = 'var Vs=Hs,Gs={';
 if (!app.includes(roleApiAnchor)) throw Error('Role API registry anchor changed');
 app = app.replace(roleApiAnchor, roleApi + roleApiAnchor.replace('{', '{roleApi:ImgoRoleApi,'));
} else {
 app = app.replace(/const ImgoRoleApi=\{[^;]+\};/, roleApi);
}
const auditApi = 'const ImgoAuditApi={index:t=>Ti({url:"/manage/audit/index",method:"post",data:t})};';
app = app.replace(/const ImgoAuditApi=\{[^;]+\};/, '');
app = app.replaceAll('auditApi:ImgoAuditApi,', '');
if (!app.includes('var Vs=Hs,Gs={')) throw Error('Audit API registry anchor changed');
app = app.replace('var Vs=Hs,Gs={', auditApi + 'var Vs=Hs,Gs={auditApi:ImgoAuditApi,');
const agentSettingApi = 'const ImgoAgentSettingApi={detail:t=>Ti({url:"/manage/agentSetting/detail",method:"post",data:t}),save:t=>Ti({url:"/manage/agentSetting/save",method:"post",data:t})};';
app = app.replace(/const ImgoAgentSettingApi=\{[^;]+\};/, '');
app = app.replaceAll('agentSettingApi:ImgoAgentSettingApi,', '');
if (!app.includes('var Vs=Hs,Gs={')) throw Error('Mentor setting API registry anchor changed');
app = app.replace('var Vs=Hs,Gs={', agentSettingApi + 'var Vs=Hs,Gs={agentSettingApi:ImgoAgentSettingApi,');
if (!app.includes('checkInHistory:t=>Ti(')) {
 const userApiAnchor = 'const Os={getUserList:';
 if (!app.includes(userApiAnchor)) throw Error('Member API registry changed');
 app = app.replace(userApiAnchor, 'const Os={checkInHistory:t=>Ti({url:"/manage/User/checkInHistory",method:"post",data:t}),getUserList:');
}
if (!app.includes('setInviteCode:t=>Ti(')) {
 const userApiAnchor = 'const Os={';
 if (!app.includes(userApiAnchor)) throw Error('Member API registry changed');
 app = app.replace(userApiAnchor, userApiAnchor + 'setInviteCode:t=>Ti({url:"/manage/User/setInviteCode",method:"post",data:t}),');
}
if (!app.includes('googleAuthDetail:t=>Ti(')) {
 const userApiAnchor = 'const Os={';
 if (!app.includes(userApiAnchor)) throw Error('Member Google Auth API registry changed');
 app = app.replace(userApiAnchor, userApiAnchor + 'googleAuthDetail:t=>Ti({url:"/manage/User/googleAuthDetail",method:"post",data:t}),googleAuthBind:t=>Ti({url:"/manage/User/googleAuthBind",method:"post",data:t}),googleAuthUnbind:t=>Ti({url:"/manage/User/googleAuthUnbind",method:"post",data:t}),');
}
if (!app.includes('batchAdd:t=>Ti(')) {
 const userApiAnchor = 'const Os={';
 if (!app.includes(userApiAnchor)) throw Error('Member batch API registry changed');
 app = app.replace(userApiAnchor, userApiAnchor + 'batchAdd:t=>Ti({url:"/manage/User/batchAdd",method:"post",data:t}),');
}
if (!app.includes('batchStatus:t=>Ti(')) {
 const userApiAnchor = 'const Os={';
 if (!app.includes(userApiAnchor)) throw Error('Member batch status API registry changed');
 app = app.replace(userApiAnchor, userApiAnchor + 'batchStatus:t=>Ti({url:"/manage/User/batchStatus",method:"post",data:t}),');
}
if (!app.includes('getSecurity:t=>Ti(')) {
 const configApiAnchor = 'const Ls={';
 if (!app.includes(configApiAnchor)) throw Error('Security settings API registry changed');
 app = app.replace(configApiAnchor, configApiAnchor + 'getSecurity:t=>Ti({url:"/manage/Config/getSecurity",method:"post",data:t}),setSecurity:t=>Ti({url:"/manage/Config/setSecurity",method:"post",data:t}),');
}
if (!app.includes('getSystemAlert:t=>Ti(')) {
 const configApiAnchor = 'const Ls={';
 if (!app.includes(configApiAnchor)) throw Error('System alert API registry changed');
 app = app.replace(configApiAnchor, configApiAnchor + 'getSystemAlert:t=>Ti({url:"/manage/Config/getSystemAlert",method:"post",data:t}),setSystemAlert:t=>Ti({url:"/manage/Config/setSystemAlert",method:"post",data:t}),testSystemAlert:t=>Ti({url:"/manage/Config/testSystemAlert",method:"post",data:t}),');
}
// Backend login always identifies itself explicitly. When the global switch is
// enabled the server validates the user's independently bound TOTP secret.
const googleLoginField = '!t.forget&&t.globalConfig.security.googleAuthEnabled?e("el-form-item",{attrs:{prop:"google_code"}},[e("el-input",{attrs:{type:"text",inputmode:"numeric",autocomplete:"one-time-code",placeholder:"请输入谷歌验证码",maxlength:"6","prefix-icon":"el-icon-key"},nativeOn:{keyup:function(e){return!e.type.indexOf("key")&&t._k(e.keyCode,"enter",13,e.key,"Enter")?null:t.handleLogin.apply(null,arguments)}},model:{value:t.loginForm.google_code,callback:function(e){t.$set(t.loginForm,"google_code",String(e||"").replace(/\\D/g,"").slice(0,6))},expression:"loginForm.google_code"}}),e("div",{staticClass:"imgo-login-google-tip"},[t._v("验证码由该账号独立绑定的 Google Authenticator 生成")])],1):t._e(),';
app = app.replaceAll(googleLoginField, '');
const forgotCodeField = 'e("el-form-item",{directives:[{name:"show",rawName:"v-show",value:t.forget,expression:"forget"}],attrs:{prop:"code"}}';
if (!app.includes(forgotCodeField)) throw Error('Backend login verification field anchor changed');
app = app.replace(forgotCodeField, googleLoginField + forgotCodeField);
const loginFormState = 'loginForm:{account:"",password:"",code:"",rememberMe:!0}';
if (!app.includes(loginFormState) && !app.includes('google_code:"",admin_login:1')) throw Error('Backend login state changed');
app = app.replace(loginFormState, 'loginForm:{account:"",password:"",code:"",google_code:"",admin_login:1,rememberMe:!0}');
const loginHandler = 'handleLogin(){!this.forget||this.loginForm.code?';
if (!app.includes(loginHandler) && !app.includes('this.globalConfig.security.googleAuthEnabled')) throw Error('Backend login handler changed');
app = app.replace(loginHandler, 'handleLogin(){if(!this.forget&&this.globalConfig.security.googleAuthEnabled&&!/^\\d{6}$/.test(this.loginForm.google_code||""))return this.$message.error("请输入6位谷歌验证码");!this.forget||this.loginForm.code?');
const loginPayload = 'const e={account:this.loginForm.account,password:this.loginForm.password,code:this.loginForm.code};';
if (!app.includes(loginPayload) && !app.includes('google_code:this.loginForm.google_code')) throw Error('Backend login payload changed');
app = app.replace(loginPayload, 'const e={account:this.loginForm.account,password:this.loginForm.password,code:this.loginForm.code,google_code:this.loginForm.google_code,admin_login:1};');
// The add-account route shares the proven login form, but gets an explicit,
// mobile-first shell and does not prefill the last account's credentials.
const addAccountRoot = 'e("div",{staticClass:"login-wrapper",class:{"imgo-add-account-login":t.isAddAccount},style:"background-image:url("+t.Background+")"}';
if (!app.includes(addAccountRoot)) {
 const loginRoot = 'e("div",{staticClass:"login-wrapper",style:"background-image:url("+t.Background+")"}';
 if (!app.includes(loginRoot)) throw Error('Add-account login root changed');
 app = app.replace(loginRoot, addAccountRoot);
}
const addAccountBack = 't.isAddAccount?e("button",{staticClass:"imgo-add-account-back",attrs:{type:"button"},on:{click:t.cancelAddAccount}},[e("i",{staticClass:"el-icon-arrow-left"}),t._v("返回账号列表")]):t._e(),';
app = app.replaceAll(addAccountBack, '');
const loginTitleAnchor = 'e("div",{staticClass:"form-box"},[e("div",{staticClass:"form-title"},[e("img",{attrs:{src:t.globalConfig.sysInfo.logo?t.globalConfig.sysInfo.logo:t.$packageData.logo,width:"100",alt:"icon"}})';
if (!app.includes(loginTitleAnchor)) throw Error('Add-account login title changed');
app = app.replace(loginTitleAnchor, loginTitleAnchor.replace('[e("div",{staticClass:"form-title"}', '[' + addAccountBack + 'e("div",{staticClass:"form-title"}'));
const defaultLoginTitle = 'e("p",{staticClass:"mt-10 f-20"},[t._v(t._s(t.globalConfig.sysInfo.name))])';
const contextualLoginTitle = 'e("p",{staticClass:"mt-10 f-20"},[t._v(t._s(t.isAddAccount?"登录其他账号":t.globalConfig.sysInfo.name))])';
if (!app.includes(contextualLoginTitle)) {
 if (!app.includes(defaultLoginTitle)) throw Error('Add-account login heading changed');
 app = app.replace(defaultLoginTitle, contextualLoginTitle);
}
const addAccountNote = 't.isAddAccount?e("div",{staticClass:"imgo-add-account-note"},[e("i",{staticClass:"el-icon-lock"}),e("div",[e("strong",[t._v("账号凭证仅保存在当前设备")]),e("span",[t._v("登录成功后会保存到本机账号列表，方便随时切换。")])])]):t._e(),';
app = app.replaceAll(addAccountNote, '');
const loginFormAnchor = ']),e("el-form",{ref:"loginForm",staticClass:"login-form"';
if (!app.includes(loginFormAnchor)) throw Error('Add-account login form changed');
app = app.replace(loginFormAnchor, ']),' + addAccountNote + 'e("el-form",{ref:"loginForm",staticClass:"login-form"');
const loginComputed = 'computed:{...(0,v.rn)({globalConfig:t=>t.globalConfig})},watch:{$route:';
const addAccountComputed = 'computed:{isAddAccount(){return"1"==String(this.$route.query&&this.$route.query.addAccount||"")},...(0,v.rn)({globalConfig:t=>t.globalConfig})},watch:{$route:';
if (!app.includes(addAccountComputed)) {
 if (!app.includes(loginComputed)) throw Error('Add-account login computed state changed');
 app = app.replace(loginComputed, addAccountComputed);
}
const rememberedAccount = 'const e=o().get("LoginAccount");e&&(this.loginForm.account=e.account,this.loginForm.password=e.password,this.loginForm.rememberMe=!0,this.$refs.account.focus())';
const addAccountRememberedAccount = 'const e=o().get("LoginAccount");e&&!this.isAddAccount?(this.loginForm.account=e.account,this.loginForm.password=e.password,this.loginForm.rememberMe=!0,this.$refs.account.focus()):this.$refs.account.focus()';
if (!app.includes(addAccountRememberedAccount)) {
 if (!app.includes(rememberedAccount)) throw Error('Remembered login account changed');
 app = app.replace(rememberedAccount, addAccountRememberedAccount);
}
const legacyAddAccountCancelMethod = 'cancelAddAccount(){const t=window.ImgoAccountVault&&window.ImgoAccountVault.read()[0];if(t)return window.ImgoAccountVault.switchTo(t),window.location.replace(window.location.origin+window.location.pathname+"#/chat"),void window.location.reload();this.$router.push("/login")},';
const addAccountCancelMethod = 'cancelAddAccount(){const t=window.ImgoAccountVault&&window.ImgoAccountVault.read()[0];if(t)return window.ImgoAccountVault.switchTo(t),void window.location.reload();this.$router.push("/login")},';
app = app.replaceAll(legacyAddAccountCancelMethod, '');
app = app.replaceAll(addAccountCancelMethod, '');
const loginMethodsAnchor = 'methods:{handleLogin(){';
if (!app.includes(loginMethodsAnchor)) throw Error('Add-account login methods changed');
app = app.replace(loginMethodsAnchor, 'methods:{' + addAccountCancelMethod + 'handleLogin(){');
const legacyAddAccountUsernameAttrs = 'attrs:{type:"text","auto-complete":"off",autocomplete:t.isAddAccount?"one-time-code":"username",name:t.isAddAccount?"imgo-add-account":"username",placeholder:"请输入账号","prefix-icon":"el-icon-user"}';
const addAccountUsernameAttrs = 'attrs:{type:"text",autocomplete:t.isAddAccount?"one-time-code":"username",name:t.isAddAccount?"imgo-add-account":"username",placeholder:"请输入账号","prefix-icon":"el-icon-user"}';
app = app.replace(legacyAddAccountUsernameAttrs, addAccountUsernameAttrs);
if (!app.includes(addAccountUsernameAttrs)) {
 const usernameAttrs = 'attrs:{type:"text","auto-complete":"off",placeholder:"请输入账号","prefix-icon":"el-icon-user"}';
 if (!app.includes(usernameAttrs)) throw Error('Add-account username input changed');
 app = app.replace(usernameAttrs, addAccountUsernameAttrs);
}
const legacyAddAccountPasswordAttrs = 'attrs:{type:"password","auto-complete":"off",autocomplete:t.isAddAccount?"new-password":"current-password",name:t.isAddAccount?"imgo-add-password":"password",placeholder:"请输入密码","prefix-icon":"el-icon-lock"}';
const addAccountPasswordAttrs = 'attrs:{type:"password",autocomplete:t.isAddAccount?"new-password":"current-password",name:t.isAddAccount?"imgo-add-password":"password",placeholder:"请输入密码","prefix-icon":"el-icon-lock"}';
app = app.replace(legacyAddAccountPasswordAttrs, addAccountPasswordAttrs);
if (!app.includes(addAccountPasswordAttrs)) {
 const passwordAttrs = 'attrs:{type:"password","auto-complete":"off",placeholder:"请输入密码","prefix-icon":"el-icon-lock"}';
 if (!app.includes(passwordAttrs)) throw Error('Add-account password input changed');
 app = app.replace(passwordAttrs, addAccountPasswordAttrs);
}
// Isolate the theme to the management route shell.
const shell = 'ii=function(){var t=this,e=t._self._c;return e("div",{staticClass:"main-container"}';
if (!app.includes('staticClass:"main-container imgo-admin"')) {
 if (!app.includes(shell)) throw Error('Management shell changed; review theme adapter');
 app = app.replace(shell, shell.replace('main-container','main-container imgo-admin'));
}
// Element UI opens collapsed submenus with mouse hover. Phones have no hover,
// so keep a small piece of explicit state that exposes the selected submenu as
// a touch-friendly action tray above the bottom navigation.
const mobileAdminSubmenuRender = 'e("el-submenu",{key:s,attrs:{index:i.path}},[';
const responsiveAdminSubmenuRender = 'e("el-submenu",{key:s,class:{"imgo-mobile-submenu-open":t.mobileSubmenuPath===i.path},attrs:{index:i.path},nativeOn:{click:function(e){return t.handleMobileSubmenu(i,e)}}},[';
if (!app.includes(responsiveAdminSubmenuRender)) {
 if (!app.includes(mobileAdminSubmenuRender)) throw Error('Management submenu render changed');
 app = app.replace(mobileAdminSubmenuRender, responsiveAdminSubmenuRender);
}
const mobileAdminData = 'data(){return{dialogTableVisible:!1,unread:0,allContacts:[],isCollapse:!1,asideWidth:"200px",active:"",routes:[]}}';
const responsiveAdminData = 'data(){return{dialogTableVisible:!1,unread:0,allContacts:[],isCollapse:!1,asideWidth:"200px",active:"",routes:[],mobileSubmenuPath:""}}';
if (!app.includes(responsiveAdminData)) {
 if (!app.includes(mobileAdminData)) throw Error('Management shell data changed');
 app = app.replace(mobileAdminData, responsiveAdminData);
}
const mobileAdminMethods = 'methods:{handleResize(){window.innerWidth<900?this.isCollapse=!0:this.isCollapse=!1},handleMenuSelect(t){';
const responsiveAdminMethods = 'methods:{handleMobileSubmenu(t,e){if(window.innerWidth>760||!t.children||!t.children.length)return;if(e.target&&e.target.closest&&e.target.closest(".el-menu-item"))return void(this.mobileSubmenuPath="");e.preventDefault(),e.stopPropagation(),this.mobileSubmenuPath=this.mobileSubmenuPath===t.path?"":t.path},closeMobileSubmenu(){this.mobileSubmenuPath=""},handleResize(){this.mobileSubmenuPath="",window.innerWidth<900?this.isCollapse=!0:this.isCollapse=!1},handleMenuSelect(t){';
if (!app.includes(responsiveAdminMethods)) {
 if (!app.includes(mobileAdminMethods)) throw Error('Management shell methods changed');
 app = app.replace(mobileAdminMethods, responsiveAdminMethods);
}
const mobileAdminMounted = 'window.addEventListener("resize",this.handleResize)}';
const responsiveAdminMounted = 'window.addEventListener("resize",this.handleResize),window.addEventListener("click",this.closeMobileSubmenu)}';
if (!app.includes(responsiveAdminMounted)) {
 if (!app.includes(mobileAdminMounted)) throw Error('Management shell mounted hook changed');
 app = app.replace(mobileAdminMounted, responsiveAdminMounted);
}
const mobileAdminDestroyAnchor = responsiveAdminMounted + ',methods:';
const responsiveAdminDestroyAnchor = responsiveAdminMounted + ',beforeDestroy(){window.removeEventListener("resize",this.handleResize),window.removeEventListener("click",this.closeMobileSubmenu)},methods:';
if (!app.includes(responsiveAdminDestroyAnchor)) {
 if (!app.includes(mobileAdminDestroyAnchor)) throw Error('Management shell lifecycle boundary changed');
 app = app.replace(mobileAdminDestroyAnchor, responsiveAdminDestroyAnchor);
}
// A local brand asset avoids broken external/default logo URLs in the admin header.
const logo = 'src:t.globalConfig.sysInfo.logo,alt:"logo"';
if (!app.includes('e.target.src="/assets/img/imgo-mark.svg"')) app = app.replace(logo, 'src:t.globalConfig.sysInfo.logo,alt:"logo"},on:{error:function(e){e.target.onerror=null;e.target.src="/assets/img/imgo-mark.svg"}');

// Member list dates and quota controls: presentation only, original API untouched.
let members = read('public/assets/js/687.70d7eca3.js');
members = members.replace(/\/\* IMGO_MEMBER_BATCH_CREATE_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_BATCH_CREATE_END \*\//, '').replaceAll(',ImgoMemberBatchCreateDialog}', '}');
members = members.replaceAll('t("imgo-member-batch-create-dialog",{ref:"memberBatchCreate",on:{saved:e.handleChange}}),', '');
members = members.replaceAll('t("el-button",{attrs:{type:"success",plain:""},staticClass:"mr-15",on:{click:function(){return e.$refs.memberBatchCreate.open()}}},[e._v("批量创建账号")]),', '');
members = members.replace(/\/\* IMGO_MEMBER_GOOGLE_AUTH_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_GOOGLE_AUTH_END \*\//, '').replaceAll(',ImgoMemberGoogleAuthDialog', '').replaceAll('t("imgo-member-google-auth-dialog",{ref:"memberGoogleAuth"}),', '');
// Remove the member broadcast adapter from a previously generated source bundle.
members = members.replace('t("el-button",{attrs:{type:"primary",plain:""},staticClass:"mr-15",on:{click:function(){return e.$refs.memberBroadcast.open()}}},[e._v("群发")]),', '');
members = members.replace('t("el-button",{attrs:{type:"primary",plain:""},staticClass:"mr-15",on:{click:e.openMemberBroadcast}},[e._v("群发")]),', '');
const previousMemberBroadcastDialogStart = members.indexOf('t("el-dialog",{attrs:{title:"成员群发"');
if (previousMemberBroadcastDialogStart >= 0) {
 const previousMemberDialog = members.indexOf('t("el-dialog",{attrs:{title:e.currentUser.realname+" 的会话管理"', previousMemberBroadcastDialogStart);
 if (previousMemberDialog < 0) throw Error('Previous member broadcast dialog boundary changed');
 members = members.slice(0, previousMemberBroadcastDialogStart) + members.slice(previousMemberDialog);
}
members = members.replace(/,\/\* IMGO_MEMBER_BROADCAST_BEGIN \*\/d=\(function\(\)\{[\s\S]*?\/\* IMGO_MEMBER_BROADCAST_END \*\/,c=s\(1001\)/, ',d=n,c=s(1001)');
members = members.replace(/\/\* IMGO_MEMBER_BROADCAST_DIALOG_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_BROADCAST_DIALOG_END \*\//, '');
members = members.replaceAll(',ImgoMemberBroadcastDialog}', '}');
members = members.replaceAll('t("imgo-member-broadcast-dialog",{ref:"memberBroadcast"}),', '');
const legacyRoleForm = 't("el-form-item",{attrs:{label:"角色",prop:"role"}}';
const legacyRoleStart = members.indexOf(legacyRoleForm);
if (legacyRoleStart >= 0) {
 const legacyRoleEnd = members.indexOf('t("el-form-item",{attrs:{label:"状态",prop:"status"}}', legacyRoleStart);
 if (legacyRoleEnd < 0) throw Error('Legacy member role form boundary changed');
 members = members.slice(0, legacyRoleStart) + members.slice(legacyRoleEnd);
}
members = members.replace(/\/\* IMGO_MEMBER_ROLE_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_ROLE_END \*\//, '').replaceAll(',ImgoMemberRoleSelect}', '}');
members = members.replace(/\/\* IMGO_MEMBER_AGENT_SETTING_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_AGENT_SETTING_END \*\//, '').replaceAll(',ImgoMemberAgentSettingDialog}', '}');
members = members.replace(/\/\* IMGO_MEMBER_GOOGLE_AUTH_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_GOOGLE_AUTH_END \*\//, '').replaceAll(',ImgoMemberGoogleAuthDialog}', '}');
members = members.replaceAll('t("imgo-member-google-auth-dialog",{ref:"memberGoogleAuth"}),', '');
members = members.replaceAll(',ImgoMemberRoleSelect}', '}');
members = members.replaceAll('t("imgo-member-agent-setting-dialog",{ref:"memberAgentSetting",on:{saved:e.handleChange}}),', '');
members = members.replace(/\/\* IMGO_MEMBER_REMARK_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_REMARK_END \*\//, '').replaceAll(',ImgoMemberRemark}', '}');
members = members.replace(/\/\* IMGO_MEMBER_INVITE_CODE_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_INVITE_CODE_END \*\//, '').replaceAll(',ImgoMemberInviteCodeDialog', '').replaceAll('t("imgo-member-invite-code-dialog",{ref:"memberInviteCode"}),', '');
members = members.replace(/\/\* IMGO_INVITE_COPY_BEGIN \*\/[\s\S]*?\/\* IMGO_INVITE_COPY_END \*\//, '').replaceAll(',ImgoInviteCodeCopy', '');
members = members.replaceAll(',ImgoMemberRemark', '');
members = members.replaceAll(',ImgoMemberReferralFilter', '');
const compactMemberColumns = [
 ['prop:"user_id",label:"ID",sortable:"custom",width:"150"', 'prop:"user_id",label:"ID",sortable:"custom",width:"52"'],
 ['prop:"realname",label:"姓名",width:"120"', 'prop:"realname",label:"姓名",width:"78"'],
 ['prop:"account",label:"账号",width:"120"', 'prop:"account",label:"账号",width:"100"'],
 ['prop:"sex",label:"性别",sortable:"custom",width:"120"', 'prop:"sex",label:"性别",sortable:"custom",width:"68"'],
 ['label:"签到信息",width:"188"', 'label:"签到信息",width:"136"'],
 ['prop:"direct_invite_count",label:"直属下级人数",width:"126"', 'prop:"direct_invite_count",label:"直属下级人数",width:"92"'],
 ['prop:"team_count",label:"团队人数",width:"112"', 'prop:"team_count",label:"团队人数",width:"76"'],
 ['prop:"create_time",label:"注册时间",width:"140"', 'prop:"create_time",label:"注册时间",width:"132"'],
 ['prop:"last_login_time",label:"最后登录时间",width:"140"', 'prop:"last_login_time",label:"最后登录时间",width:"132"'],
 ['prop:"remark",label:"备注","min-width":"300"', 'prop:"remark",label:"备注","min-width":"120"'],
 ['prop:"status",label:"状态",width:"120"', 'prop:"status",label:"状态",width:"70"']
];
// The canonical chunk is reused on subsequent builds, so restore its original
// column anchors before stripping and recreating the injected columns.
for (const [oldCompact, original] of [
 ['prop:"sex",label:"性别",sortable:"custom",width:"58"', 'prop:"sex",label:"性别",sortable:"custom",width:"120"'],
 ['prop:"create_time",label:"注册时间",width:"120"', 'prop:"create_time",label:"注册时间",width:"140"'],
 ['prop:"last_login_time",label:"最后登录时间",width:"120"', 'prop:"last_login_time",label:"最后登录时间",width:"140"']
]) members = members.replace(oldCompact, original);
for (const [original, compact] of compactMemberColumns) members = members.replace(compact, original);

// The generated member chunk is also the input for later builds. Strip our
// previous adapter before applying it again so rebuilds remain idempotent.
const checkInModule = read('frontend/checkin-history.js');
const memberRoleModule = read('frontend/member-role-select.js');
const financeModule = read('frontend/member-finance-dialog.js');
const referralFilterModule = read('frontend/member-referral-filter.js');
const referralFilterSource = '/* IMGO_REFERRAL_FILTER_BEGIN */\n' + referralFilterModule + '\n/* IMGO_REFERRAL_FILTER_END */\n';
const referralFilterControl = 't("imgo-member-referral-filter",{attrs:{scope:e.params.referral_scope,"agent-mode":Number((e.$store.state.userInfo||{}).agent_mode)===1,"referrer-account":e.params.referrer_account},on:{"update:scope":function(t){e.$set(e.params,"referral_scope",t)},"update:referrer-account":function(t){e.$set(e.params,"referrer_account",t)},search:function(){return e.handleChange()}}})';
const previousReferralFilterControl = 't("imgo-member-referral-filter",{attrs:{scope:e.params.referral_scope,"referrer-account":e.params.referrer_account},on:{"update:scope":function(t){e.$set(e.params,"referral_scope",t)},"update:referrer-account":function(t){e.$set(e.params,"referrer_account",t)},search:function(){return e.handleChange()}}})';
const financeAction = 'Number((e.$store.state.userInfo||{}).user_id)===1?t("div",{staticClass:"imgo-member-finance-actions"},[t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"recharge")}}},[e._v("充值")]),t("el-button",{attrs:{type:"text",size:"small"},on:{click:function(){return e.$refs.memberFinance.open(s.row,"withdraw")}}},[e._v("提现")])]):e._e()';
members = members.split(checkInModule + '\n').join('');
members = members.split(financeModule + '\n').join('');
members = members.replace(/\/\* IMGO_MEMBER_ROLE_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_ROLE_END \*\//, '');
members = members.replace(/\/\* IMGO_REFERRAL_FILTER_BEGIN \*\/[\s\S]*?\/\* IMGO_REFERRAL_FILTER_END \*\/\n?/, '');
members = members.replace(/\/\/ Vue 2 component embedded in the existing compiled member page\.[\s\S]*?\n(?=s\.r\(t\))/, '');
members = members.split(referralFilterControl + ',').join('');
members = members.split(',' + referralFilterControl).join('');
members = members.split(previousReferralFilterControl + ',').join('');
members = members.split(',' + previousReferralFilterControl).join('');
members = members.replace(/params:\{page:1,limit:20,keywords:"",order_field:"",order_type:1,referral_scope:(?:"(?:direct|all)?"|Number\(\(this\.\$store\.state\.userInfo\|\|\{\}\)\.agent_mode\)===1\?"all":""),referrer_account:""\}/, 'params:{page:1,limit:20,keywords:"",order_field:"",order_type:1}');
members = members.replaceAll('dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog,ImgoMemberReferralFilter}', 'dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog}');
// Remove a previously built finance component even when its source has changed.
members = members.replace(/\n?\/\/ Vue 2 dialog attached to each row's finance buttons on the legacy member page\.[\s\S]*?\n(?=s\.r\(t\))/, '');
members = members.split(financeAction + ',').join('');
members = members.replace(/fixed:"right",label:"操作",width:"(?:235|164)"/, 'fixed:"right",label:"操作",width:"180"');
members = members.replaceAll('dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog}', 'dialogue:o.Z,ImgoCheckInHistory}');
members = members.replaceAll('t("imgo-member-finance-dialog",{ref:"memberFinance"}),', '');
members = members.replaceAll('dialogue:o.Z,ImgoCheckInHistory}', 'dialogue:o.Z}');
members = members.replaceAll('t("imgo-check-in-history",{ref:"checkinHistory"}),', '');
for (const key of ['create_time', 'last_login_time']) {
 const value = 's.row.' + key;
 const formatted = `(v=>!v?'—':/^\\d+$/.test(String(v))?new Date(Number(v)*1000).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai',hour12:false}):v)(${value})`;
 members = members.replace(`e._s(${value})`, `e._s(${formatted})`);
}
// Normalize earlier generated member bundles, which may contain repeated summary injections.
const legacyCheckInColumn = 't("el-table-column",{attrs:{label:"签到信息",width:"188"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("div",{staticClass:"imgo-checkin-summary"},[t("span",{staticClass:"imgo-checkin-days"},[e._v("累计 "+e._s(s.row.checkin_days||0)+" 天")]),t("span",{staticClass:"imgo-checkin-status",class:s.row.checkin_today?"is-signed":"is-unsigned"},[e._v(s.row.checkin_today?"今日已签到":"今日未签到")])]),t("div",{staticClass:"imgo-checkin-last"},[e._v("最近："+e._s(s.row.checkin_last_date||"—"))])]}}])}),';
members = members.split(legacyCheckInColumn).join('');
const legacyCheckInDetail = '"edit"==e.formType?t("el-form-item",{attrs:{label:"签到信息"}},[t("div",{staticClass:"imgo-checkin-detail"},[e._v("累计 "+e._s(e.detail.checkin_days||0)+" 天 · "+(e.detail.checkin_today?"今日已签到":"今日未签到"))]),t("div",{staticClass:"imgo-checkin-last"},[e._v("最近签到："+e._s(e.detail.checkin_last_date||"—"))])]):e._e(),';
members = members.split(legacyCheckInDetail).join('');
// Read-only check-in summary and detailed history live in the existing member page.
const checkInColumnAnchor = 't("el-table-column",{attrs:{prop:"create_time",label:"注册时间"';
if (!members.includes(checkInColumnAnchor)) throw Error('Member table changed; review check-in column adapter');
const checkInColumn = 't("el-table-column",{attrs:{label:"签到信息",width:"188"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("div",{staticClass:"imgo-checkin-summary"},[t("span",{staticClass:"imgo-checkin-days"},[e._v("累计 "+e._s(s.row.checkin_days||0)+" 天")]),t("span",{staticClass:"imgo-checkin-status",class:s.row.checkin_today?"is-signed":"is-unsigned"},[e._v(s.row.checkin_today?"今日已签到":"今日未签到")])]),t("div",{staticClass:"imgo-checkin-last"},[e._v("最近："+e._s(s.row.checkin_last_date||"—"))]),t("el-button",{staticClass:"imgo-checkin-open",attrs:{type:"text",size:"mini"},on:{click:function(){return e.$refs.checkinHistory.open(s.row)}}},[e._v("查看详情")])]}}])}),';
members = members.split(checkInColumn).join('');
members = members.replace(checkInColumnAnchor, checkInColumn + checkInColumnAnchor);
members = members.replace('prop:"invite_code",label:"邀请码",width:"82"', 'prop:"invite_code",label:"邀请码",width:"108"');
const referralColumns = 't("el-table-column",{attrs:{prop:"invite_code",label:"邀请码",width:"108"}}),t("el-table-column",{attrs:{prop:"direct_invite_count",label:"直属下级人数",width:"126"}}),t("el-table-column",{attrs:{prop:"team_count",label:"团队人数",width:"112"}}),';
const referralColumnsWithCopy = 't("el-table-column",{attrs:{label:"邀请码",width:"118"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("imgo-invite-code-copy",{attrs:{code:s.row.invite_code}})]}}])}),t("el-table-column",{attrs:{prop:"direct_invite_count",label:"直属下级人数",width:"126"}}),t("el-table-column",{attrs:{prop:"team_count",label:"团队人数",width:"112"}}),';
members = members.split(referralColumns).join('');
members = members.split(referralColumnsWithCopy).join('');
members = members.replace(checkInColumnAnchor, referralColumnsWithCopy + checkInColumnAnchor);
const checkInDetailAnchor = 't("el-form-item",{attrs:{label:"备注",prop:"remark"}';
// Keep the add-member field and method reproducible across repeated builds.
const parentInviteField = '"add"==e.formType?t("el-form-item",{attrs:{label:"上级邀请码",prop:"parent_invite_code"}},[t("el-input",{attrs:{maxlength:6,placeholder:"选填，填写后将验证上级是否存在",disabled:Number((e.$store.state.userInfo||{}).user_id)!==1&&Number((e.$store.state.userInfo||{}).agent_mode)===1},model:{value:e.detail.parent_invite_code,callback:function(t){e.$set(e.detail,"parent_invite_code",t)},expression:"detail.parent_invite_code"}})]):e._e(),';
members = members.split(parentInviteField).join('');
members = members.replace(checkInDetailAnchor, parentInviteField + checkInDetailAnchor);
const addMethodPattern = /(?:async )?addUser\(\)\{[\s\S]*?\},editUser\(e\)/;
if (!addMethodPattern.test(members)) throw Error('Member add method changed');
members = members.replace(addMethodPattern, read('frontend/member-add-method.js').trim() + ',editUser(e)');
if (!members.includes(checkInDetailAnchor)) throw Error('Member form changed; review check-in detail adapter');
const checkInDetail = '"edit"==e.formType?t("el-form-item",{attrs:{label:"签到信息"}},[t("div",{staticClass:"imgo-checkin-detail"},[e._v("累计 "+e._s(e.detail.checkin_days||0)+" 天 · "+(e.detail.checkin_today?"今日已签到":"今日未签到"))]),t("div",{staticClass:"imgo-checkin-last"},[e._v("最近签到："+e._s(e.detail.checkin_last_date||"—"))])]):e._e(),';
members = members.replace(checkInDetailAnchor, checkInDetail + checkInDetailAnchor);
const referralDetail = '"edit"==e.formType?t("el-form-item",{attrs:{label:"邀请关系"}},[t("div",{staticClass:"imgo-referral-detail"},[e._v("邀请码："+e._s(e.detail.invite_code||"—")+" · 直属下级："+e._s(e.detail.direct_invite_count||0)+" 人 · 团队："+e._s(e.detail.team_count||0)+" 人")])]):e._e(),';
members = members.split(referralDetail).join('');
members = members.replace(checkInDetailAnchor, referralDetail + checkInDetailAnchor);
const memberComponentAnchor = 'n={components:{userSelect:l.Z,dialogue:o.Z},data(){';
if (!members.includes(memberComponentAnchor)) throw Error('Member component changed; review check-in dialog adapter');
members = members.replace(memberComponentAnchor, memberComponentAnchor.replace('dialogue:o.Z}', 'dialogue:o.Z,ImgoCheckInHistory}'));
const memberModuleAnchor = '4368:function(e,t,s){s.r(t)';
if (!members.includes(memberModuleAnchor)) throw Error('Member module changed; review check-in dialog adapter');
members = members.replace(memberModuleAnchor, '4368:function(e,t,s){' + checkInModule + '\ns.r(t)');
const memberDialogAnchor = 't("el-dialog",{attrs:{title:e.currentUser.realname+" 的会话管理"';
if (!members.includes(memberDialogAnchor)) throw Error('Member dialogs changed; review check-in dialog adapter');
members = members.replace(memberDialogAnchor, 't("imgo-check-in-history",{ref:"checkinHistory"}),' + memberDialogAnchor);
const financeColumnAnchor = 't("el-table-column",{attrs:{fixed:"right",label:"操作",width:"180"},scopedSlots:e._u([{key:"default",fn:function(s){return[';
if (!members.includes(financeColumnAnchor)) throw Error('Member operation column changed; review finance adapter');
members = members.replace(financeColumnAnchor, financeColumnAnchor.replace('width:"180"', 'width:"235"') + financeAction + ',');
members = members.replace('dialogue:o.Z,ImgoCheckInHistory}', 'dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog}');
members = members.replace('t("imgo-check-in-history",{ref:"checkinHistory"}),', 't("imgo-check-in-history",{ref:"checkinHistory"}),t("imgo-member-finance-dialog",{ref:"memberFinance"}),');
members = members.replace('4368:function(e,t,s){' + checkInModule + '\n', '4368:function(e,t,s){' + checkInModule + '\n' + financeModule + '\n');
const memberAddButton = 't("el-button",{staticClass:"mr-15",on:{click:e.addUser}},[e._v("添加成员")])';
const memberBroadcastButton = 't("el-button",{attrs:{type:"primary",plain:""},staticClass:"mr-15",on:{click:function(){return e.$refs.memberBroadcast.open()}}},[e._v("群发")])';
const memberBatchButton = 't("el-button",{attrs:{type:"success",plain:""},staticClass:"mr-15",on:{click:function(){return e.$refs.memberBatchCreate.open()}}},[e._v("批量创建账号")])';
if (!members.includes(memberAddButton)) throw Error('Member toolbar changed; review referral filter adapter');
members = members.replace(memberAddButton, memberBroadcastButton + ',' + memberBatchButton + ',' + memberAddButton + ',' + referralFilterControl);
const memberParams = 'params:{page:1,limit:20,keywords:"",order_field:"",order_type:1}';
if (!members.includes(memberParams)) throw Error('Member filter state changed; review referral filter adapter');
members = members.replace(memberParams, 'params:{page:1,limit:20,keywords:"",order_field:"",order_type:1,referral_scope:Number((this.$store.state.userInfo||{}).agent_mode)===1?"all":"",referrer_account:""}');
members = members.replace('dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog}', 'dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog,ImgoMemberReferralFilter}');
members = members.replace('4368:function(e,t,s){' + checkInModule + '\n' + financeModule + '\n', '4368:function(e,t,s){' + checkInModule + '\n' + financeModule + '\n' + referralFilterSource);
const memberBroadcastDialog = 't("imgo-member-broadcast-dialog",{ref:"memberBroadcast"}),';
if (!members.includes(memberDialogAnchor)) throw Error('Member broadcast dialog anchor changed');
members = members.replace(memberDialogAnchor, memberBroadcastDialog + memberDialogAnchor);
members = members.replace('4368:function(e,t,s){', '4368:function(e,t,s){/* IMGO_MEMBER_BROADCAST_DIALOG_BEGIN */'+read('frontend/member-broadcast-dialog.js')+'/* IMGO_MEMBER_BROADCAST_DIALOG_END */');
members = members.replaceAll('attrs:{min:0,max:1e3}', 'attrs:{min:-1,max:1e3}');
const passwordPatches = [
 ['min:6,max:16,message:"长度在 6 到 16 个字符"', 'min:6,max:30,message:"长度在 6 到 30 个字符"'],
 ['this.password.length>16', 'this.password.length>30'],
 ['请输入6-16个字符串的密码', '请输入6-30位密码']
];
for (const [from, to] of passwordPatches) {
 if (!members.includes(from) && !members.includes(to)) throw Error('Member password validation changed: ' + from);
 members = members.replaceAll(from, to);
}
const remarkColumn = 't("el-table-column",{attrs:{prop:"remark",label:"备注","min-width":"300"}})';
const editableRemarkColumn = 't("el-table-column",{attrs:{prop:"remark",label:"备注","min-width":"300"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("imgo-member-remark",{key:s.row.user_id,attrs:{row:s.row}})]}}])})';
if (!members.includes(editableRemarkColumn)) {
 if (!members.includes(remarkColumn)) throw Error('Remark column changed');
 members=members.replace(remarkColumn,editableRemarkColumn);
}
// Replace the full operation cell, retaining each original action and its permissions.
const operationsStart = members.indexOf('t("el-table-column",{attrs:{fixed:"right",label:"操作"');
const operationsEnd = members.indexOf('],1),t("div",{staticClass:"mt-15"}', operationsStart);
if (operationsStart < 0 || operationsEnd < 0) throw Error('Member operation column boundary changed');
members = members.slice(0,operationsStart) + read('frontend/member-actions.js').trim() + members.slice(operationsEnd);
for (const [from, to] of compactMemberColumns) {
 if (!members.includes(from) && !members.includes(to)) throw Error('Member column changed: ' + from);
 members = members.replace(from, to);
}
members=members.replace('4368:function(e,t,s){','4368:function(e,t,s){/* IMGO_MEMBER_REMARK_BEGIN */'+read('frontend/member-remark.js')+'/* IMGO_MEMBER_REMARK_END */');
members=members.replace('ImgoMemberReferralFilter}', 'ImgoMemberReferralFilter,ImgoMemberRemark}');
const inviteDialogAnchor = 't("imgo-member-finance-dialog",{ref:"memberFinance"}),';
if (!members.includes(inviteDialogAnchor)) throw Error('Member dialog insertion point changed');
members = members.replace(inviteDialogAnchor, inviteDialogAnchor + 't("imgo-member-invite-code-dialog",{ref:"memberInviteCode"}),');
members = members.replace('ImgoMemberReferralFilter,ImgoMemberRemark}', 'ImgoMemberReferralFilter,ImgoMemberRemark,ImgoMemberInviteCodeDialog}');
members = members.replace('4368:function(e,t,s){/* IMGO_MEMBER_REMARK_BEGIN */', '4368:function(e,t,s){/* IMGO_MEMBER_INVITE_CODE_BEGIN */'+read('frontend/member-invite-code.js')+'/* IMGO_MEMBER_INVITE_CODE_END *//* IMGO_MEMBER_REMARK_BEGIN */');
members = members.replace('ImgoMemberInviteCodeDialog}', 'ImgoMemberInviteCodeDialog,ImgoInviteCodeCopy}');
members = members.replace('4368:function(e,t,s){/* IMGO_MEMBER_INVITE_CODE_BEGIN */', '4368:function(e,t,s){/* IMGO_INVITE_COPY_BEGIN */'+read('frontend/member-invite-copy.js')+'/* IMGO_INVITE_COPY_END *//* IMGO_MEMBER_INVITE_CODE_BEGIN */');
if (!members.includes('/* IMGO_MEMBER_ROLE_BEGIN */')) {
 members = members.replace('4368:function(e,t,s){', '4368:function(e,t,s){/* IMGO_MEMBER_ROLE_BEGIN */'+memberRoleModule+'/* IMGO_MEMBER_ROLE_END */');
}
if (!members.includes('ImgoMemberRoleSelect}')) {
 members = members.replace('ImgoMemberInviteCodeDialog,ImgoInviteCodeCopy}', 'ImgoMemberInviteCodeDialog,ImgoInviteCodeCopy,ImgoMemberRoleSelect}');
}
members = members.replace('4368:function(e,t,s){', '4368:function(e,t,s){/* IMGO_MEMBER_AGENT_SETTING_BEGIN */'+read('frontend/member-agent-setting.js')+'/* IMGO_MEMBER_AGENT_SETTING_END */');
members = members.replace('ImgoMemberRoleSelect}', 'ImgoMemberRoleSelect,ImgoMemberAgentSettingDialog}');
members = members.replace('ImgoMemberAgentSettingDialog}', 'ImgoMemberAgentSettingDialog,ImgoMemberBroadcastDialog}');
members = members.replace('4368:function(e,t,s){', '4368:function(e,t,s){/* IMGO_MEMBER_GOOGLE_AUTH_BEGIN */'+read('frontend/member-google-auth.js')+'/* IMGO_MEMBER_GOOGLE_AUTH_END */');
members = members.replace('ImgoMemberBroadcastDialog}', 'ImgoMemberBroadcastDialog,ImgoMemberGoogleAuthDialog}');
members = members.replace('ImgoMemberGoogleAuthDialog}', 'ImgoMemberGoogleAuthDialog,ImgoMemberBatchCreateDialog}');
members = members.replace('4368:function(e,t,s){', '4368:function(e,t,s){/* IMGO_MEMBER_BATCH_CREATE_BEGIN */'+read('frontend/member-batch-create.js')+'/* IMGO_MEMBER_BATCH_CREATE_END */');
const agentDialogAnchor = 't("imgo-member-invite-code-dialog",{ref:"memberInviteCode"}),';
if (!members.includes(agentDialogAnchor)) throw Error('Mentor setting dialog insertion point changed');
members = members.replace(agentDialogAnchor, agentDialogAnchor + 't("imgo-member-agent-setting-dialog",{ref:"memberAgentSetting",on:{saved:e.handleChange}}),');
members = members.replace(agentDialogAnchor, agentDialogAnchor + 't("imgo-member-google-auth-dialog",{ref:"memberGoogleAuth"}),');
const googleAuthDialogAnchor = 't("imgo-member-google-auth-dialog",{ref:"memberGoogleAuth"}),';
if (!members.includes(googleAuthDialogAnchor)) throw Error('Member batch dialog insertion point changed');
members = members.replace(googleAuthDialogAnchor, googleAuthDialogAnchor + 't("imgo-member-batch-create-dialog",{ref:"memberBatchCreate",on:{saved:e.handleChange}}),');
const memberRoleColumn = 't("el-table-column",{attrs:{label:"角色",width:"132"},scopedSlots:e._u([{key:"default",fn:function(s){return[t("imgo-member-role-select",{attrs:{row:s.row}})]}}])}),';
if (!members.includes(memberRoleColumn)) {
 const roleStart = members.indexOf('t("el-table-column",{attrs:{prop:"role",label:"角色"');
 const roleEnd = members.indexOf('t("el-table-column",{attrs:{label:"签到信息"', roleStart);
 if (roleStart < 0 || roleEnd < 0) throw Error('Member role column changed');
 members = members.slice(0, roleStart) + memberRoleColumn + members.slice(roleEnd);
}
write('public/assets/js/687.70d7eca3.js',members);
const membersHash=crypto.createHash('sha256').update(members).digest('hex').slice(0,12);
write(`public/assets/js/687.imgo${membersHash}.js`,members);
app=app.replace(/687:"(?:70d7eca3|imgo[a-f0-9]+)"/,`687:"imgo${membersHash}"`);
// Keep the notice popup switch in the existing System Settings form.
let settings = read('public/assets/js/789.34f6ec0f.js');
settings = settings.replace(/\/\* IMGO_SECURITY_SETTINGS_BEGIN \*\/[\s\S]*?\/\* IMGO_SECURITY_SETTINGS_END \*\//, '').replaceAll(',ImgoSecurityPanel}', '}');
settings = settings
 .replace('label:"自动添加客服"', 'label:"自动添加好友"')
 .replace('开启后，用户注册之后自动设置为专属客服。', '开启后，新注册用户会与分配的客服自动成为好友。')
 .replace('通过客服自动发送给新注册的人员', '成为好友后由客服私聊发送给新注册用户');
// Invitation URLs are generated elsewhere; keep only the registration mode selector here.
const invitePanelStart = ',e("div",{directives:[{name:"show",rawName:"v-show",value:2==t.sysInfo.regtype,expression:"sysInfo.regtype==2"}],staticClass:"mt-15"},[';
const invitePanelEnd = 'e("vue-qr",{ref:"qrCode",attrs:{text:t.inviteUrl,width:"200",height:"200",logoSrc:t.sysInfo.logo}})],1)';
const invitePanelFrom = settings.indexOf(invitePanelStart);
if (invitePanelFrom < 0) throw Error('Registration invitation panel changed');
const invitePanelTo = settings.indexOf(invitePanelEnd, invitePanelFrom);
if (invitePanelTo < 0) throw Error('Registration invitation panel end changed');
settings = settings.slice(0, invitePanelFrom) + settings.slice(invitePanelTo + invitePanelEnd.length);
const noticeDefaultAnchor = 'multipleLogin:"0"},chatInfo:';
if (!settings.includes(noticeDefaultAnchor)) throw Error('System settings defaults changed');
settings = settings.replace(noticeDefaultAnchor, 'multipleLogin:"0",noticePopup:"1",showScan:"1",showGroupQr:"1",clientDownloadUrl:""},chatInfo:');
const noticeFormAnchor = 'e("el-form-item",{attrs:{label:"系统状态",prop:"state"}';
if (!settings.includes(noticeFormAnchor)) throw Error('System settings form changed');
const noticeSwitch = 'e("el-form-item",{attrs:{label:"系统公告滚动条",prop:"noticePopup"}},[e("el-switch",{attrs:{"active-value":"1","inactive-value":"0"},model:{value:t.sysInfo.noticePopup,callback:function(e){t.$set(t.sysInfo,"noticePopup",e)},expression:"sysInfo.noticePopup"}}),e("span",{staticClass:"ml-10 c-999 f-12"},[t._v("开启后，聊天页面顶部滚动显示最新系统公告")])],1),';
const visibilitySwitch = (label, prop, hint) => `e("el-form-item",{attrs:{label:"${label}",prop:"${prop}"}},[e("el-switch",{attrs:{"active-value":"1","inactive-value":"0"},model:{value:t.sysInfo.${prop},callback:function(e){t.$set(t.sysInfo,"${prop}",e)},expression:"sysInfo.${prop}"}}),e("span",{staticClass:"ml-10 c-999 f-12"},[t._v("${hint}")])],1),`;
const scanSwitch = visibilitySwitch('显示扫一扫', 'showScan', '控制“我的”常用功能和消息页右上角＋菜单中的扫一扫');
const groupQrSwitch = visibilitySwitch('显示群二维码', 'showGroupQr', '控制群聊信息页的群二维码入口');
const clientDownloadField = 'e("el-form-item",{attrs:{label:"客户端下载地址",prop:"clientDownloadUrl"}},[e("el-input",{attrs:{placeholder:"https://example.com/download",maxlength:2048,clearable:true},model:{value:t.sysInfo.clientDownloadUrl,callback:function(e){t.$set(t.sysInfo,"clientDownloadUrl",e)},expression:"sysInfo.clientDownloadUrl"}}),e("div",{staticClass:"c-999 f-12"},[t._v("支持 HTTP/HTTPS；客户端下载入口将跳转至此地址，留空使用默认下载页面")])],1),';
settings = settings.replace(noticeFormAnchor, clientDownloadField + noticeSwitch + scanSwitch + groupQrSwitch + noticeFormAnchor);
const autoGroupLimitInput = 'e("el-input-number",{attrs:{min:5,max:1e3},model:{value:t.chatInfo.autoAddGroup.userMax';
if (!settings.includes(autoGroupLimitInput)) throw Error('Automatic group size input changed');
settings = settings.replace(autoGroupLimitInput, autoGroupLimitInput.replace('min:5,max:1e3', 'min:5,max:1e4'));
const settingsModuleAnchor = '3585:function(t,e,r){"use strict";';
if (!settings.includes(settingsModuleAnchor)) throw Error('System settings module anchor changed');
settings = settings.replace(settingsModuleAnchor, settingsModuleAnchor + '/* IMGO_SECURITY_SETTINGS_BEGIN */'+read('frontend/security-settings.js')+'/* IMGO_SECURITY_SETTINGS_END *//* IMGO_SYSTEM_ALERT_BEGIN */'+read('frontend/system-alert-settings.js')+'/* IMGO_SYSTEM_ALERT_END */');
const settingsComponentsAnchor = 'components:{VueQr:l(),userSelect:c.Z}';
if (!settings.includes(settingsComponentsAnchor)) throw Error('System settings components changed');
settings = settings.replace(settingsComponentsAnchor, 'components:{VueQr:l(),userSelect:c.Z,ImgoSecurityPanel,ImgoSystemAlertPanel}');
const settingsTabsEnd = '],2)],1)],1)],1)},o=[]';
if (!settings.includes(settingsTabsEnd)) throw Error('System settings tabs boundary changed');
const securityTab = 'Number((t.$store.state.userInfo||{}).user_id)===1?e("el-tab-pane",[e("span",{attrs:{slot:"label"},slot:"label"},[e("i",{staticClass:"el-icon-lock"}),t._v(" 安全设置")]),e("imgo-security-panel")],1):t._e()';
const systemAlertTab = 'Number((t.$store.state.userInfo||{}).user_id)===1?e("el-tab-pane",[e("span",{attrs:{slot:"label"},slot:"label"},[e("i",{staticClass:"el-icon-bell"}),t._v(" 系统报警")]),e("imgo-system-alert-panel")],1):t._e()';
settings = settings.replace(settingsTabsEnd, '],2)],1),'+securityTab+','+systemAlertTab+'],1)],1)},o=[]');
const settingsHash = crypto.createHash('sha256').update(settings).digest('hex').slice(0,12);
write(`public/assets/js/789.imgo${settingsHash}.js`, settings);
if (!/789:"(?:34f6ec0f|imgo[a-f0-9]+)"/.test(app)) throw Error('System settings chunk hash map changed');
app = app.replace(/789:"(?:34f6ec0f|imgo[a-f0-9]+)"/, `789:"imgo${settingsHash}"`);
// Quick-chat dialog: semantic trigger, visible close control and viewport sizing.
app = app.replace('"custom-class":"sideMenu-message","show-close":!1', '"custom-class":"sideMenu-message",title:"消息中心","show-close":!0');
app = app.replace('e("span",{staticClass:"message",on:{click:function(e){return t.showMessageBox()}}}', 'e("button",{staticClass:"message imgo-message-trigger",attrs:{type:"button",title:"消息中心","aria-label":"打开消息中心"},on:{click:function(e){return t.showMessageBox()}}}');
// Keep config structurally valid across bootstrap, cached data and socket patches.
const configBegin = '/* IMGO_CONFIG_BEGIN */', configEnd = '/* IMGO_CONFIG_END */';
if (app.includes(configBegin)) app = app.slice(0, app.indexOf(configBegin)) + app.slice(app.indexOf(configEnd) + configEnd.length);
if (!app.includes('const Li=')) throw Error('Config store anchor changed');
app = app.replace('const Li=', configBegin + '\n' + read('frontend/config-state.js') + '\n' + read('frontend/rbac-menu.js') + '\n' + configEnd + 'const Li=');
const configPatches = [
 ['globalConfig:[],wsStatus:', 'globalConfig:imgoNormalizeConfig(null),wsStatus:'],
 ['setGlobalConfig(t,e){t.globalConfig=e}', 'setGlobalConfig(t,e){t.globalConfig=imgoNormalizeConfig(e,t.globalConfig);o().set("globalConfig",t.globalConfig)}'],
 ['e&&(document.title=e.sysInfo.name,this.$store.commit("setGlobalConfig",e))', 'e&&(this.$store.commit("setGlobalConfig",e),document.title=this.$store.state.globalConfig.sysInfo.name)'],
 ['let e=o().get("globalConfig"),s=e.demon_mode', 'let e=imgoNormalizeConfig(o().get("globalConfig")),s=e.demon_mode'],
 ['0==i.code&&(o().set("globalConfig",i.data),t("setGlobalConfig",i.data),e(i))', '0==i.code?(i.data=imgoNormalizeConfig(i.data),t("setGlobalConfig",i.data),e(i)):e(i)'],
 ['0!=t.data.sysInfo.state||this.$router.push', '0!=t.code||!t.data||0!=t.data.sysInfo.state||this.$router.push']
];
for (const [from,to] of configPatches) {
 if (!app.includes(from) && !app.includes(to)) throw Error('Config adapter anchor changed: '+from);
 if (!app.includes(to)) app = app.replace(from,to);
}
// Invalidate the async management chunk in browsers that previously loaded it.
const chunkHash = crypto.createHash('sha256').update(text).digest('hex').slice(0, 12);
const hashPattern = /585:"(?:dea7864f|imgo[a-f0-9]+)"/;
if (!hashPattern.test(app)) throw Error('Management chunk hash map changed');
app = app.replace(hashPattern, `585:"imgo${chunkHash}"`);
write(`public/assets/js/585.imgo${chunkHash}.js`, text);
write('public/assets/js/app.85372e4e.js', app);
write('public/assets/css/imgo-maintenance.css', read('frontend/maintenance.css') + '\n' + read('frontend/admin-theme.css') + '\n' + read('frontend/message-panel.css') + '\n' + read('frontend/bank-panel.css') + '\n' + read('frontend/finance-orders.css') + '\n' + read('frontend/member-finance.css') + '\n' + read('frontend/member-agent-setting.css') + '\n' + read('frontend/member-batch-create.css') + '\n' + read('frontend/chat-design.css') + '\n' + read('frontend/chat-account-switcher.css') + '\n' + read('frontend/login-mobile.css') + '\n' + read('frontend/role-panel.css') + '\n' + read('frontend/audit-panel.css') + '\n' + read('frontend/google-auth.css') + '\n' + read('frontend/system-alert.css') + '\n' + read('frontend/mobile-compat.css') + '\n' + read('frontend/chat-message-comfort.css'));
let index = read('public/index.html');
const version = crypto.createHash('sha256').update(app+text+members+managementGroups+read('frontend/maintenance.css')+read('frontend/admin-theme.css') + '\n' + read('frontend/message-panel.css') + '\n' + read('frontend/bank-panel.css') + '\n' + read('frontend/finance-orders.css') + '\n' + read('frontend/member-finance.css') + '\n' + read('frontend/member-agent-setting.css') + '\n' + read('frontend/member-batch-create.css') + '\n' + read('frontend/chat-design.css') + '\n' + read('frontend/chat-account-switcher.css') + '\n' + read('frontend/login-mobile.css') + '\n' + read('frontend/role-panel.css') + '\n' + read('frontend/audit-panel.css') + '\n' + read('frontend/google-auth.css') + '\n' + read('frontend/system-alert.css') + '\n' + read('frontend/mobile-compat.css') + '\n' + read('frontend/chat-message-comfort.css')).digest('hex').slice(0,12);
index = index.replace(/assets\/js\/app\.85372e4e\.js(?:\?v=[a-z0-9]+)?/g, `assets/js/app.85372e4e.js?v=${version}`);
index = index.replace(/<link[^>]*href="assets\/css\/imgo-maintenance.css[^>]*>/g, '');
index = index.replace('</head>', `<link href="assets/css/imgo-maintenance.css?v=${version}" rel="stylesheet"></head>`);
write('public/index.html', index);
for (const [prefix, keep] of [
 ['173.imgo', `173.imgo${managementGroupsHash}.js`],
 ['585.imgo', `585.imgo${chunkHash}.js`],
 ['687.imgo', `687.imgo${membersHash}.js`],
 ['789.imgo', `789.imgo${settingsHash}.js`]
]) {
 for (const file of fs.readdirSync(path.join(root, 'public/assets/js'))) {
  if (file.startsWith(prefix) && file.endsWith('.js') && file !== keep) fs.unlinkSync(path.join(root, 'public/assets/js', file));
 }
}
console.log('Built management maintenance panel', version);
