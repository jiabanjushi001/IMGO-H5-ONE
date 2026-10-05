// WebSocket updateConfig sends one section; HTTP supplies the complete config.
function imgoNormalizeConfig(value, previous) {
 const object = v => v !== null && typeof v === 'object' && !Array.isArray(v);
 const defaults = {
  demon_mode: false,
  sysInfo: {name:'Imgo',logo:'',state:1,runMode:2,closeTips:'系统维护中',showScan:'1',showGroupQr:'1'},
  chatInfo: {online:0,webrtc:0,simpleChat:0,redoTime:120,dbDelMsg:0,groupCreateRole:'mentor'},
  security: {googleAuthEnabled:false},
  compass: {status:0,mode:1,list:[]}, fileUpload: {size:50,videoSize:200}
 };
 const safe = v => Object.fromEntries(Object.entries(object(v) ? v : {}).filter(([k]) => !['__proto__','constructor','prototype'].includes(k)));
 const base = safe(previous);
 let incoming = safe(value);
 if (typeof incoming.name === 'string' && Object.prototype.hasOwnProperty.call(incoming,'value')) {
  const section = incoming.name;
  incoming = !['__proto__','constructor','prototype'].includes(section) ? {[section]:incoming.value} : {};
 }
 const result = {...defaults,...base,...incoming};
 for (const key of new Set([...Object.keys(defaults),...Object.keys(base),...Object.keys(incoming)])) {
  if (object(defaults[key]) || object(base[key]) || object(incoming[key])) {
   result[key] = {...safe(defaults[key]),...safe(base[key]),...safe(incoming[key])};
  }
 }
 return result;
}

function imgoCanCreateGroup(user, chatInfo) {
 const role = String(chatInfo && chatInfo.groupCreateRole || 'mentor');
 if (role === 'all') return true;
 return Number(user && user.user_id) === 1 || Number(user && user.agent_mode) === 1;
}

function imgoCanViewMemberAccount(user) {
 return Number(user && user.user_id) === 1 || Number(user && user.admin_role_id) > 0;
}

function imgoMessageSenderLabel(sender, viewer) {
 const value = sender && typeof sender === 'object' ? sender : {};
 const nickname = String(value.displayName || value.realname || '').trim();
 if (!imgoCanViewMemberAccount(viewer)) return nickname;
 const account = String(value.account || '').trim();
 if (account && nickname && account !== nickname) return account + '(' + nickname + ')';
 return account || nickname;
}

if (typeof window !== 'undefined') {
 window.imgoMessageSenderLabel = imgoMessageSenderLabel;
}

function imgoUploadLimitMB(message, value) {
 const config = value !== null && typeof value === 'object' ? value : {};
 const isVideo = message && message.type === 'video';
 const fallback = isVideo ? 200 : 50;
 const parsed = Number(isVideo ? config.videoSize : config.size);
 return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : fallback;
}
