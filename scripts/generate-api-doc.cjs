const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const source = fs.readFileSync(path.join(root, 'internal/server/routes.go'), 'utf8')
const routes = []
const addRoute = (prefix, actions, meta) => {
  for (const action of actions.trim().split(/\s+/)) {
    routes.push({ path: `${prefix}/${action}`.toLowerCase(), ...meta })
  }
}
for (const match of source.matchAll(/\badd\("([^"]+)",\s*(true|false),\s*(true|false),\s*"([^"]+)"/g)) {
  addRoute(match[1], match[4], { public: match[2] === 'true', super: match[3] === 'true', permission: '' })
}
for (const match of source.matchAll(/\baddManage\("([^"]+)",\s*"([^"]+)",\s*"([^"]+)"/g)) {
  addRoute(match[1], match[3], { public: false, super: false, permission: match[2] })
}
routes.push({ path: '/manage/index/overview', public: false, super: false, permission: 'manage.overview' })
routes.push({ path: '/view', public: true, super: false, permission: '' })
routes.push({ path: '/downapp', public: true, super: false, permission: '' })

const unique = [...new Map(routes.map(route => [route.path, route])).values()]
const actionNames = {
  index: '列表/首页', detail: '详情', add: '新增', edit: '编辑', del: '删除', save: '保存', status: '状态', get: '读取', update: '更新',
  login: '登录', logout: '退出登录', register: '注册', sendcode: '发送验证码', getsysteminfo: '获取系统信息', checkversion: '检查版本',
  getcontacts: '会话联系人', getcontactinfo: '联系人详情', sendmessage: '发送消息', forwardmessage: '转发消息', getuserinfo: '用户资料',
  searchuser: '搜索用户', userlist: '用户列表', getmessagelist: '消息列表', setmsgisread: '消息已读', setting: '聊天设置',
  undomessage: '撤回消息', removemessage: '移除消息', isnotice: '消息免打扰', setchattop: '置顶会话', delchat: '删除会话',
  sendtomsg: '发送到会话', editpassword: '修改密码', updateuserinfo: '更新用户资料', editaccount: '修改账号', readatmsg: '读取 @ 消息',
  getadminnotice: '系统公告', delmessage: '删除消息', getquickreplies: '快捷用语', savequickreplies: '保存快捷用语',
  getalluser: '可选群成员', groupuserlist: '群成员列表', groupinfo: '群资料', editgroupname: '修改群名', editgroupavatar: '修改群头像',
  addgroupuser: '添加群成员', setmanager: '设置群管理员', removeuser: '移除群成员', setnospeak: '群禁言', removegroup: '退出/解散群',
  setnotice: '群公告', groupsetting: '群设置', joingroup: '加入群聊', changeowner: '转让群主', clearmessage: '清空消息',
  uploadfile: '上传文件', uploadimage: '上传图片', uploadavatar: '上传头像', uploademoji: '上传表情',
  getapplymsg: '好友申请', setnickname: '好友备注', move: '排序', submit: '提交', withdraw: '提现', history: '历史记录', entries: '流水',
  getinfo: '读取配置', getconfig: '读取配置项', getallconfig: '全部配置', setconfig: '保存配置', getinvitelink: '邀请链接', sendtestemail: '测试邮件',
  getsecurity: '读取安全设置', setsecurity: '保存安全设置', overview: '概况统计', noticelist: '公告列表', delnotice: '删除公告', publishnotice: '发布公告',
  gettasklist: '任务列表', starttask: '启动任务', stoptask: '停止任务', settaskconfig: '任务配置', gettasklog: '任务日志', cleartasklog: '清理任务日志',
  setremark: '设置备注', setinvitecode: '修改邀请码', setstatus: '修改状态', setrole: '设置角色', checkinhistory: '签到记录', batchadd: '提交批量创建账号任务', batchstatus: '查询批量创建账号进度',
  broadcast: '群发', broadcastoptions: '群发可选成员', googleauthdetail: '谷歌验证状态', googleauthbind: '绑定谷歌验证', googleauthunbind: '解绑谷歌验证',
  permissions: '权限目录', account: '账户汇总', credit: '余额调整', review: '提现审核', freeze: '冻结/解除提现订单', recharges: '充值记录', recharge: '充值',
  dealmsg: '处理消息', gettasklog: '任务日志', setnumberjoin: '群号加入开关', delgroupuser: '移除群成员', createuser: '创建用户', binduid: '绑定 WebSocket 用户',
  bindgroup: '绑定群连接', offline: '离线通知', avatar: '头像', download: '下载', scanqr: '扫码解析', downloadapp: '下载客户端', downapp: '下载客户端'
}
const areaNames = { config: '设置', index: '概况', user: '成员', group: '群聊', message: '消息', task: '任务', role: '角色', audit: '日志', wallet: '钱包', bank: '银行卡', files: '文件', agentsetting: '导师设置', im: '聊天', friend: '好友', emoji: '表情', checkin: '签到', invite: '邀请' }
const describe = route => {
  const parts = route.path.split('/').filter(Boolean)
  const action = parts[parts.length - 1]
  const area = areaNames[parts[1]] || areaNames[parts[2]] || parts.slice(0, -1).join('/')
  return `${area} - ${actionNames[action] || action}`
}
const authLabel = route => route.public ? '公开' : route.super ? '超级管理员' : route.permission ? `登录 + ${route.permission}` : '登录'
const backend = unique.filter(route => route.path.startsWith('/manage/')).sort((a, b) => a.path.localeCompare(b.path))
const h5 = unique.filter(route => !route.path.startsWith('/manage/')).sort((a, b) => a.path.localeCompare(b.path))
const fixed = [
  ['GET', '/health', '公开', '存活检查'], ['GET', '/ready', '公开', '数据库就绪检查'], ['GET', '/metrics', '公开', '运行指标'],
  ['GET', '/captcha', '公开', '图形验证码'], ['GET', '/captcha/:config', '公开', '指定配置图形验证码'],
  ['GET', '/wss', 'WebSocket Token', 'WebSocket 连接'], ['GET', '/ws', 'WebSocket Token', 'WebSocket 兼容地址']
]
const lines = []
const generatedDate = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())
lines.push('IMGO API 接口清单', `生成日期：${generatedDate}`, '')
lines.push('说明：', '- 普通 HTTP 接口推荐使用 POST；下载、健康检查、验证码和 WebSocket 按表中方法调用。', '- 需要登录的接口使用请求头：Authorization: bearer <token>。', '- 后台 IP 白名单只限制后台登录和 /manage/*；H5/用户端接口与 WebSocket 不受影响。', '- 本文件由 scripts/generate-api-doc.cjs 根据 internal/server/routes.go 生成。', '')
lines.push(`一、后台 API（${backend.length} 个）`, 'METHOD | PATH | 鉴权 | 功能')
for (const route of backend) lines.push(`POST | ${route.path} | ${authLabel(route)} | ${describe(route)}`)
lines.push('', `二、H5 / 用户端 API（${h5.length} 个）`, 'METHOD | PATH | 鉴权 | 功能')
for (const route of h5) lines.push(`POST | ${route.path} | ${authLabel(route)} | ${describe(route)}`)
lines.push('', `三、基础服务接口（${fixed.length} 个）`, 'METHOD | PATH | 鉴权 | 功能')
for (const row of fixed) lines.push(row.join(' | '))
lines.push('', `合计：${backend.length + h5.length + fixed.length} 个接口。`, '')
fs.writeFileSync(path.join(root, 'API.txt'), lines.join('\n'))
console.log(`Generated API.txt: backend=${backend.length}, h5=${h5.length}, fixed=${fixed.length}`)
