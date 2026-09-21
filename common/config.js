// 非 H5 平台沿用原有本地 API；H5 必须从外部 config.js 获取服务器地址。
let apiUrl = 'http://127.0.0.1:8088';
// #ifdef H5
// H5 启动前从独立的 config.js 读取地址；构建产物不包含这个文件。
// URL 构造器同时校验端口范围；仅检查 http 前缀会放过 80881 等无效端口。
function runtimeEndpoint(value, protocols) {
  if (typeof value !== 'string' || !value.trim()) return '';
  try {
    const parsed = new URL(value.trim());
    return protocols.includes(parsed.protocol) && parsed.hostname && !parsed.username && !parsed.password
      ? value.trim().replace(/\/+$/, '') : '';
  } catch (error) {
    return '';
  }
}
function resolveApiUrl() {
  const configuredServer = Array.isArray(window.apiServers)
    ? window.apiServers.find(server => runtimeEndpoint(server?.httpUrl, ['http:', 'https:']))
    : null;
  return runtimeEndpoint(window.httpUrl, ['http:', 'https:']) ||
    runtimeEndpoint(configuredServer?.httpUrl, ['http:', 'https:']);
}
function resolveWssUrl(apiBase) {
  const configuredServer = Array.isArray(window.apiServers)
    ? window.apiServers.find(server => runtimeEndpoint(server?.httpUrl, ['http:', 'https:']) === apiBase) ||
      window.apiServers.find(server => runtimeEndpoint(server?.wsUrl, ['ws:', 'wss:']))
    : null;
  return runtimeEndpoint(window.wsUrl, ['ws:', 'wss:']) ||
    runtimeEndpoint(configuredServer?.wsUrl, ['ws:', 'wss:']) ||
    (apiBase ? apiBase.replace(/^http/, 'ws') + '/wss' : '');
}
const configuredApiUrl = resolveApiUrl();
if (!configuredApiUrl) throw new Error('缺少有效的 config.js API 配置，H5 已停止启动');
apiUrl = configuredApiUrl;
// #endif
let wssUrl = apiUrl.replace(/^http/, 'ws') + '/wss';
// #ifdef H5
wssUrl = resolveWssUrl(apiUrl) || wssUrl;
// #endif

// 是否开启H5的调试模式
let isVConsole = false
// #ifdef MP-WEIXIN
try {
	if (typeof wx !== 'undefined' && typeof wx.getAccountInfoSync === 'function') {
		let env = wx.getAccountInfoSync()
		if (env.miniProgram.envVersion == 'develop') {
			isVConsole = true
		}
	}
} catch (e) {}
// #endif
// #ifdef APP-PLUS || H5
if (process.env.NODE_ENV === 'development') {
	// #ifdef H5
	// H5 调试台改为显式开启，避免调试按钮遮挡真实界面。
	isVConsole = new URLSearchParams(window.location.search).get('debug') === '1';
	// #endif
	// #ifdef APP-PLUS
	isVConsole = true
	// #endif
}
// #endif

const now = Date.now || function() {
	return new Date().getTime();
};
const isArray = Array.isArray || function(obj) {
	return obj instanceof Array;
};

// app更新的配置
const updateConfig = {
	url:apiUrl+'/common/pub/checkVersion',  //检查版本的接口
	bgColor:'',  //升级主色，按钮背景颜色
	iconUrl:'' //升级小图标
}

/* 检查版本需要返回的数据说明
 * | 参数名称        | 一定返回     | 类型        | 描述
 * | -------------|--------- | --------- | ------------- |
 * | versionCode     | y        | int       | 版本号 ：20240331       |
 * | versionName     | y        | String    | 版本名称 :4.0.1     |
 * | versionInfo     | y        | String    | 版本信息 ： 修复了bug     |
 * | updateType      | y        | String    | forcibly = 强制更新, solicit = 弹窗确认更新, silent = 静默更新 |
 * | downloadUrl     | y        | String    | 版本下载链接（IOS安装包更新请放跳转store应用商店链接,安卓apk和wgt文件放文件下载链接）  |
 */
export default {
	apiUrl,
	wssUrl,
	/** 运行时再取一次，避免构造时拿到过期的备用站地址 */
	getWssUrl() {
		// #ifdef H5
		try {
			const api = resolveApiUrl() || this.apiUrl
			return resolveWssUrl(api) || this.wssUrl
		} catch (e) {
			return this.wssUrl
		}
		// #endif
		// #ifndef H5
		return this.wssUrl
		// #endif
	},
	now,
	isArray,
	isVConsole,
	updateConfig
}
