// 登录/注册页品牌名与 Logo；默认置空。填写后立即生效，无需改前端源码。
// appLogo 可填相对路径（如 favicon.ico）或完整 http(s) 地址。
window.appName = ''
window.appLogo = ''

// 按顺序填写主站和备用站。每一项的 API 与 WebSocket 必须属于同一套服务。
// 备用站可继续在数组中添加；不要填写尚未部署或仅本机可用的地址。
// 注意：不要把 127.0.0.1 / localhost 写进正式包，否则探测失败会误切到本地导致 wss 连不上。
window.apiServers = [
  //{ httpUrl: 'https://imgo.myad.top', wsUrl: 'wss://imgo.myad.top/wss' },
  { httpUrl: 'https://ipa.1bxms.hynbn.com', wsUrl: 'wss://ipa.1bxms.hynbn.com/wss' },
  { httpUrl: 'https://ipa.imbxms.com', wsUrl: 'wss://ipa.imbxms.com/wss' },
  { httpUrl: 'https://ipa.imbxmscf.com', wsUrl: 'wss://ipa.imbxmscf.com/wss' },
]
