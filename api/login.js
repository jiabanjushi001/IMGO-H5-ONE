// 统一请求路径前缀在libs/axios.js中修改
import {
	postJsonRequest,
	apiUrl
} from '@/utils/request.js';
let LoginApi = {}

/**
 * @desc 登录接口
 * @param {*} 参数 
 */
LoginApi.login = (params) => {
	return postJsonRequest('/common/Pub/login', params)
}


/**
 * @desc 退出接口
 * @param {*} 参数 
 */
LoginApi.logout = (params) => {
	return postJsonRequest('/common/Pub/logout', params)
}

/**
 * 注册用户
 * @param {*} data
 */
LoginApi.register = (params) => {
	return postJsonRequest('/common/pub/register', params)
}

/**
 * @desc 绑定client_id
 * @param {*} 参数 
 */
LoginApi.bindUid = (params) => {
	return postJsonRequest('/common/Pub/bindUid', params)
}

/**
 * @desc 绑定群聊client_id
 * @param {*} 参数 
 */
LoginApi.bindGroup = (params) => {
	return postJsonRequest('common/pub/bindGroup', params)
}

/**
 * 获取全局配置
 * @param {*} data
 */
LoginApi.getSystemInfo = (params) => {
	return postJsonRequest('/common/pub/getSystemInfo', params)
}

/**
 * 发送验证码
 * @param {*} data
 */
LoginApi.sendCode = (params) => {
	return postJsonRequest('/common/pub/sendCode', params)
}

export default LoginApi;
