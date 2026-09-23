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
const extension = `;${begin}\n${read('frontend/notice-actions.js')};\n${panel};\n${overview};\n${bankPanel};\n${financeOrders};\n${rolePanel};\nconst LegacyManagement = d; d = { name: 'ImgoManagement', render(h) { const bank = this.$route.path === '/manage/bank', finance = this.$route.path.startsWith('/manage/finance/'), role = this.$route.path === '/manage/role'; return h('div', {class: 'imgo-management'}, [role ? h(ImgoRolePanel) : finance ? h(ImgoFinanceShell) : bank ? h(ImgoBankPanel) : h(ImgoOverview, [h(LegacyManagement), h(ImgoMaintenancePanel)])]); } };\n${end}`;
text = text.replace(anchor, anchor + extension);
write(chunk, text);
let app = read('public/assets/js/app.85372e4e.js');
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
 app = app.replace(anchor, anchor.split('=')[0] + '=(function(component){' + start + '\n' + read(source) + (name === 'CHAT' ? '\n' + read('frontend/chat-profile.js') + '\n' + read('frontend/chat-context-remark.js') + '\n' + read('frontend/group-avatar-chat.js') : '') +
  '\nreturn component;' + end + '})(' + variable + '),');
}
// Group avatar settings adapter, installed before Vue normalizes the component.
const groupAvatarStart = '/* IMGO_GROUP_AVATAR_BEGIN */', groupAvatarEnd = '/* IMGO_GROUP_AVATAR_END */';
const groupAvatarWrapper = '(function(component){' + groupAvatarStart;
if (app.includes(groupAvatarWrapper)) {
 const from = app.indexOf(groupAvatarWrapper), to = app.indexOf(groupAvatarEnd, from);
 if (to < 0) throw Error('Incomplete group avatar adapter');
 app = app.slice(0, from) + 'W' + app.slice(to + groupAvatarEnd.length + '})(W)'.length);
}
if (!app.includes('Z=W,X=')) throw Error('Group settings component changed');
app = app.replace('Z=W,X=', 'Z=(function(component){' + groupAvatarStart + '\n' + read('frontend/group-avatar.js') + '\nreturn component;' + groupAvatarEnd + '})(W),X=');
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
if (app.includes(adminProfileTitle)) {
 app = app.replace(adminProfileTitle, profileTitle);
} else if (!app.includes(profileTitle)) {
 if (!app.includes(privateTitle)) throw Error('Private chat title anchor changed');
 app = app.replace(privateTitle, profileTitle+privateTitle);
}
app = app.replace('staticClass:"chat-box"', 'staticClass:"chat-box imgo-chat-design"');
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
const walletApi = 'const ImgoWalletApi={index:t=>Ti({url:"/manage/wallet/index",method:"post",data:t}),detail:t=>Ti({url:"/manage/wallet/detail",method:"post",data:t}),account:t=>Ti({url:"/manage/wallet/account",method:"post",data:t}),credit:t=>Ti({url:"/manage/wallet/credit",method:"post",data:t}),review:t=>Ti({url:"/manage/wallet/review",method:"post",data:t}),entries:t=>Ti({url:"/manage/wallet/entries",method:"post",data:t}),recharges:t=>Ti({url:"/manage/wallet/recharges",method:"post",data:t}),recharge:t=>Ti({url:"/manage/wallet/recharge",method:"post",data:t}),withdraw:t=>Ti({url:"/manage/wallet/withdraw",method:"post",data:t})};';
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
const agentSettingApi = 'const ImgoAgentSettingApi={detail:t=>Ti({url:"/manage/agentSetting/detail",method:"post",data:t}),save:t=>Ti({url:"/manage/agentSetting/save",method:"post",data:t})};';
app = app.replace(/const ImgoAgentSettingApi=\{[^;]+\};/, '');
if (!app.includes('var Vs=Hs,Gs={')) throw Error('Mentor setting API registry anchor changed');
app = app.replace('var Vs=Hs,Gs={', agentSettingApi + 'var Vs=Hs,Gs={agentSettingApi:ImgoAgentSettingApi,');
// Rebuilds start from the previously patched app bundle: remove the old registry entry first.
app = app.replace('agentSettingApi:ImgoAgentSettingApi,agentSettingApi:ImgoAgentSettingApi,', 'agentSettingApi:ImgoAgentSettingApi,');
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
// Isolate the theme to the management route shell.
const shell = 'ii=function(){var t=this,e=t._self._c;return e("div",{staticClass:"main-container"}';
if (!app.includes('staticClass:"main-container imgo-admin"')) {
 if (!app.includes(shell)) throw Error('Management shell changed; review theme adapter');
 app = app.replace(shell, shell.replace('main-container','main-container imgo-admin'));
}
// A local brand asset avoids broken external/default logo URLs in the admin header.
const logo = 'src:t.globalConfig.sysInfo.logo,alt:"logo"';
if (!app.includes('e.target.src="/assets/img/imgo-mark.svg"')) app = app.replace(logo, 'src:t.globalConfig.sysInfo.logo,alt:"logo"},on:{error:function(e){e.target.onerror=null;e.target.src="/assets/img/imgo-mark.svg"}');

// Member list dates and quota controls: presentation only, original API untouched.
let members = read('public/assets/js/687.70d7eca3.js');
const legacyRoleForm = 't("el-form-item",{attrs:{label:"角色",prop:"role"}}';
const legacyRoleStart = members.indexOf(legacyRoleForm);
if (legacyRoleStart >= 0) {
 const legacyRoleEnd = members.indexOf('t("el-form-item",{attrs:{label:"状态",prop:"status"}}', legacyRoleStart);
 if (legacyRoleEnd < 0) throw Error('Legacy member role form boundary changed');
 members = members.slice(0, legacyRoleStart) + members.slice(legacyRoleEnd);
}
members = members.replace(/\/\* IMGO_MEMBER_ROLE_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_ROLE_END \*\//, '').replaceAll(',ImgoMemberRoleSelect}', '}');
members = members.replace(/\/\* IMGO_MEMBER_AGENT_SETTING_BEGIN \*\/[\s\S]*?\/\* IMGO_MEMBER_AGENT_SETTING_END \*\//, '').replaceAll(',ImgoMemberAgentSettingDialog}', '}');
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
if (!members.includes(memberAddButton)) throw Error('Member toolbar changed; review referral filter adapter');
members = members.replace(memberAddButton, memberAddButton + ',' + referralFilterControl);
const memberParams = 'params:{page:1,limit:20,keywords:"",order_field:"",order_type:1}';
if (!members.includes(memberParams)) throw Error('Member filter state changed; review referral filter adapter');
members = members.replace(memberParams, 'params:{page:1,limit:20,keywords:"",order_field:"",order_type:1,referral_scope:Number((this.$store.state.userInfo||{}).agent_mode)===1?"all":"",referrer_account:""}');
members = members.replace('dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog}', 'dialogue:o.Z,ImgoCheckInHistory,ImgoMemberFinanceDialog,ImgoMemberReferralFilter}');
members = members.replace('4368:function(e,t,s){' + checkInModule + '\n' + financeModule + '\n', '4368:function(e,t,s){' + checkInModule + '\n' + financeModule + '\n' + referralFilterSource);
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
const agentDialogAnchor = 't("imgo-member-invite-code-dialog",{ref:"memberInviteCode"}),';
if (!members.includes(agentDialogAnchor)) throw Error('Mentor setting dialog insertion point changed');
members = members.replace(agentDialogAnchor, agentDialogAnchor + 't("imgo-member-agent-setting-dialog",{ref:"memberAgentSetting",on:{saved:e.handleChange}}),');
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
settings = settings.replace(noticeDefaultAnchor, 'multipleLogin:"0",noticePopup:"1",showScan:"1",showGroupQr:"1"},chatInfo:');
const noticeFormAnchor = 'e("el-form-item",{attrs:{label:"系统状态",prop:"state"}';
if (!settings.includes(noticeFormAnchor)) throw Error('System settings form changed');
const noticeSwitch = 'e("el-form-item",{attrs:{label:"系统公告滚动条",prop:"noticePopup"}},[e("el-switch",{attrs:{"active-value":"1","inactive-value":"0"},model:{value:t.sysInfo.noticePopup,callback:function(e){t.$set(t.sysInfo,"noticePopup",e)},expression:"sysInfo.noticePopup"}}),e("span",{staticClass:"ml-10 c-999 f-12"},[t._v("开启后，聊天页面顶部滚动显示最新系统公告")])],1),';
const visibilitySwitch = (label, prop, hint) => `e("el-form-item",{attrs:{label:"${label}",prop:"${prop}"}},[e("el-switch",{attrs:{"active-value":"1","inactive-value":"0"},model:{value:t.sysInfo.${prop},callback:function(e){t.$set(t.sysInfo,"${prop}",e)},expression:"sysInfo.${prop}"}}),e("span",{staticClass:"ml-10 c-999 f-12"},[t._v("${hint}")])],1),`;
const scanSwitch = visibilitySwitch('显示扫一扫', 'showScan', '控制“我的”常用功能和消息页右上角＋菜单中的扫一扫');
const groupQrSwitch = visibilitySwitch('显示群二维码', 'showGroupQr', '控制群聊信息页的群二维码入口');
settings = settings.replace(noticeFormAnchor, noticeSwitch + scanSwitch + groupQrSwitch + noticeFormAnchor);
const autoGroupLimitInput = 'e("el-input-number",{attrs:{min:5,max:1e3},model:{value:t.chatInfo.autoAddGroup.userMax';
if (!settings.includes(autoGroupLimitInput)) throw Error('Automatic group size input changed');
settings = settings.replace(autoGroupLimitInput, autoGroupLimitInput.replace('min:5,max:1e3', 'min:5,max:1e4'));
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
write('public/assets/css/imgo-maintenance.css', read('frontend/maintenance.css') + '\n' + read('frontend/admin-theme.css') + '\n' + read('frontend/message-panel.css') + '\n' + read('frontend/bank-panel.css') + '\n' + read('frontend/finance-orders.css') + '\n' + read('frontend/member-finance.css') + '\n' + read('frontend/member-agent-setting.css') + '\n' + read('frontend/chat-design.css') + '\n' + read('frontend/role-panel.css'));
let index = read('public/index.html');
const version = crypto.createHash('sha256').update(app+text+members+read('frontend/maintenance.css')+read('frontend/admin-theme.css') + '\n' + read('frontend/message-panel.css') + '\n' + read('frontend/bank-panel.css') + '\n' + read('frontend/finance-orders.css') + '\n' + read('frontend/member-finance.css') + '\n' + read('frontend/member-agent-setting.css') + '\n' + read('frontend/chat-design.css') + '\n' + read('frontend/role-panel.css')).digest('hex').slice(0,12);
index = index.replace(/assets\/js\/app\.85372e4e\.js(?:\?v=[a-z0-9]+)?/g, `assets/js/app.85372e4e.js?v=${version}`);
index = index.replace(/<link[^>]*href="assets\/css\/imgo-maintenance.css[^>]*>/g, '');
index = index.replace('</head>', `<link href="assets/css/imgo-maintenance.css?v=${version}" rel="stylesheet"></head>`);
write('public/index.html', index);
for (const [prefix, keep] of [
 ['585.imgo', `585.imgo${chunkHash}.js`],
 ['687.imgo', `687.imgo${membersHash}.js`],
 ['789.imgo', `789.imgo${settingsHash}.js`]
]) {
 for (const file of fs.readdirSync(path.join(root, 'public/assets/js'))) {
  if (file.startsWith(prefix) && file.endsWith('.js') && file !== keep) fs.unlinkSync(path.join(root, 'public/assets/js', file));
 }
}
console.log('Built management maintenance panel', version);
