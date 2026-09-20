// 统一请求路径前缀在libs/axios.js中修改
import {
	postJsonRequest,
	apiUrl
} from '@/utils/request.js';
let friendApi = {}

/**
 * @desc 删除好友
 * @param {*} 参数 
 */
friendApi.delFriend = (params) => {
	return postJsonRequest('/enterprise/friend/del', params)
}

/**
 * @desc 添加好友
 * @param {*} 参数 
 */
friendApi.addFriend = (params) => {
	return postJsonRequest('/enterprise/friend/add', params)
}

/**
 * @desc 新朋友列表
 * @param {*} 参数 
 */
friendApi.applyList = (params) => {
	return postJsonRequest('/enterprise/friend/index', params)
}

/**
 * @desc 同意或者拒绝请求
 * @param {*} 参数 
 */
friendApi.acceptApply = (params) => {
	return postJsonRequest('/enterprise/friend/update', params)
}

/**
 * @desc 设置好有备注
 * @param {*} 参数 
 */
friendApi.setNickname = (params) => {
	return postJsonRequest('/enterprise/friend/setNickname', params)
}

export default friendApi;
