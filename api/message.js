// 统一请求路径前缀在libs/axios.js中修改
import {
	postRequest,
	postJsonRequest,
	apiUrl
} from '@/utils/request.js';
let msgApi = {}

msgApi.uploadUrl=apiUrl+'/common/upload/uploadFile';
// 上传头像
msgApi.uploadAvatar=apiUrl+'/common/upload/uploadAvatar';
/**
 * @desc 普通消息列表（小程序）
 * @param {*} 参数 
 */
msgApi.initContacts = (params) => {
	return postJsonRequest('/enterprise/im/getContacts', params)
}

msgApi.getAdminNotice = () => {
	return postJsonRequest('/enterprise/im/getAdminNotice', {})
}



/**
 * @desc 普通消息列表（小程序）
 * @param {*} 参数 
 */
msgApi.getMessageList = (params) => {
	return postJsonRequest('/enterprise/im/getMessageList', params)
}

/**
 * @desc 设置聊天置顶
 * @param {*} 参数 
 */
msgApi.setChatTopAPI = (params) => {
	return postJsonRequest('/enterprise/im/setChatTop', params)
}

/**
 * @desc 删除聊天
 * @param {*} 参数 
 */
msgApi.delChatAPI = (params) => {
	return postJsonRequest('/enterprise/im/delChat', params)
}

/**
 * @desc 发送文本聊天消息
 * @param {*} 参数 
 */
msgApi.sendMessage = (params) => {
	return postJsonRequest('/enterprise/im/sendMessage', params)
}

/**
 * @desc 转发聊天消息
 * @param {*} 参数 
 */
msgApi.forwardMessage = (params) => {
	return postJsonRequest('/enterprise/im/forwardMessage', params)
}

/**
 * @desc 设置消息已读
 * @param {*} 参数 
 */
msgApi.setMsgIsRead = (params) => {
	return postJsonRequest('/enterprise/im/setMsgIsRead', params)
}

/**
 * @desc 设置艾特消息已读
 * @param {*} 参数 
 */
msgApi.readAtMsg = (params) => {
	return postJsonRequest('/enterprise/im/readAtMsg', params)
}

/**
 * 撤回消息
 * @param {*} data
 */
msgApi.undoMessage = (params) => {
	return postJsonRequest('/enterprise/im/undoMessage', params)
}

/**
 * 删除消息
 * @param {*} data
 */
msgApi.delMessage = (params) => {
	return postJsonRequest('/enterprise/im/delMessage', params)
}


/**
 * 发送ws消息
 * @param {*} data
 */
msgApi.sendToMsg = (params) => {
	return postJsonRequest('/enterprise/im/sendToMsg', params)
}

/**
 * 消息免打扰
 * @param {*} data
 */
msgApi.isNoticeAPI = (params) => {
	return postJsonRequest('/enterprise/im/isNotice', params)
}

/**
 * 更新业务卡片
 * @param {*} data
 */
msgApi.updateCard = (params) => {
	return postJsonRequest('/enterprise/im/updateCard', params)
}

/**
 * 同意或者忽略团队
 * @param {*} data
 */
msgApi.joinGroup = (params) => {
	return postJsonRequest('/enterprise/group/joinGroup', params)
}

/**
 * 获取群成员
 * @param {*} data
 */
msgApi.groupUserList = (params) => {
	return postJsonRequest('/enterprise/group/groupUserList', params)
}

/**
 * 修改群公告
 * @param {*} data
 */
msgApi.setNotice = (params) => {
	return postJsonRequest('/enterprise/group/setNotice', params)
}
/**
 * 群管理信息
 * @param {*} data
 */
msgApi.groupInfo = (params) => {
	return postJsonRequest('/enterprise/group/groupInfo', params)
}

/**
 * 加入群聊
 * @param {*} data
 */
msgApi.joinGroup = (params) => {
	return postJsonRequest('/enterprise/group/joinGroup', params)
}


/**
 * 修改群管理信息
 * @param {*} data
 */
msgApi.groupSetting = (params) => {
	return postJsonRequest('/enterprise/group/groupSetting', params)
}

/**
 * 转让管理权限
 * @param {*} data
 */
msgApi.changeOwner = (params) => {
	return postJsonRequest('/enterprise/group/changeOwner', params)
}

/**
 * 获取全部人员
 * @param {*} data
 */
msgApi.getAllUser = (params) => {
	return postJsonRequest('/enterprise/group/getAllUser', params)
}

/**
 * 创建群聊
 * @param {*} data
 */
msgApi.addGroup = (params) => {
	return postJsonRequest('/enterprise/group/add', params)
}
/**
 * 绑定群聊
 * @param {*} data
 */
msgApi.bindGroup = (params) => {
	return postJsonRequest('/common/index/bindGroup', params)
}

/**
 * 修改群聊名字
 * @param {*} data
 */
msgApi.editGroupName = (params) => {
	return postJsonRequest('/enterprise/group/editGroupName', params)
}
msgApi.editGroupAvatar = (params) => {
	return postJsonRequest('/enterprise/group/editGroupAvatar', params)
}
/**
 * 添加群成员
 * @param {*} data
 */
msgApi.addGroupUser = (params) => {
	return postJsonRequest('/enterprise/group/addGroupUser', params)
}

/**
 * 删除群成员
 * @param {*} data
 */
msgApi.removeUser = (params) => {
	return postJsonRequest('/enterprise/group/removeUser', params)
}

/**
 * 删除群聊
 * @param {*} data
 */
msgApi.removeGroup = (params) => {
	return postJsonRequest('/enterprise/group/removeGroup', params)
}

/**
 * 删除群聊聊天记录
 * @param {*} data
 */
msgApi.clearMessage = (params) => {
	return postJsonRequest('/enterprise/group/clearMessage', params)
}


/**
 * 设置/取消管理员
 * @param {*} data
 */
msgApi.setManager = (params) => {
	return postJsonRequest('/enterprise/group/setManager', params)
}

/**
 * 设置禁言
 * @param {*} data
 */
msgApi.setNoSpeak = (params) => {
	return postJsonRequest('/enterprise/group/setNoSpeak', params)
}

/**
 * 获取成员信息
 * @param {*} data
 */
msgApi.getUserInfo = (params) => {
	return postJsonRequest('/enterprise/im/getUserInfo', params)
}

// 搜索用户
msgApi.searchUser= (params)  =>{
	return postJsonRequest('enterprise/im/searchUser', params)
}

// 修改用户信息
msgApi.updateUserInfo= (params)  =>{
	return postJsonRequest('enterprise/im/updateUserInfo', params)
}

// 修改账号
msgApi.editAccount= (params)  =>{
	return postJsonRequest('enterprise/im/editAccount', params)
}

// 修改密码
msgApi.editPassword= (params)  =>{
	return postJsonRequest('enterprise/im/editPassword', params)
}

// 获取联系人信息
msgApi.contactInfo= (params)  =>{
	return postJsonRequest('enterprise/im/getContactInfo', params)
}

export default msgApi;
