// 统一请求路径前缀在libs/axios.js中修改
import {
	postRequest,
	postJsonRequest,
	apiUrl
} from '@/utils/request.js';
let emojiApi = {}

emojiApi.uploadEmoji=apiUrl+'/common/upload/uploadEmoji';

/**
 * @desc 表情列表
 * @param {*} 参数 
 */
emojiApi.emojiList = (params) => {
	return postJsonRequest('/enterprise/emoji/index', params)
}

/**
 * @desc 添加表情
 * @param {*} 参数 
 */
emojiApi.addEmoji = (params) => {
	return postJsonRequest('/enterprise/emoji/add', params)
}

/**
 * @desc 删除表情
 * @param {*} 参数 
 */
emojiApi.delEmoji = (params) => {
	return postJsonRequest('/enterprise/emoji/del', params)
}


/**
 * @desc 移动表情
 * @param {*} 参数 
 */
emojiApi.moveEmoji = (params) => {
	return postJsonRequest('/enterprise/emoji/move', params)
}


export default emojiApi;
