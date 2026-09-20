// 引入其他模块的接口

import msgApi from '@/api/message.js';  //  消息
import LoginApi from '@/api/login.js';  //登录相关
import friendApi from '@/api/friend.js';  //登录相关
import emojiApi from '@/api/emoji.js';  //登录相关
import bankApi from '@/api/bank.js';
// 导出接口
export default {
	msgApi,
	LoginApi,
	friendApi,
	emojiApi,
	bankApi
}
