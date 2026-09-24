import { defineStore } from 'pinia'
import utils from '@/utils/utils.js';
import msgApi from '@/api/message.js';
import { normalizeAvatarData } from '@/utils/avatar.js';

const isFileTransferAssistant = (contact) => String(contact?.id) === '-1' && Number(contact?.is_group) === 3;

const storedContacts = () => {
	const contacts = uni.getStorageSync('allContacts');
	return Array.isArray(contacts) ? contacts.filter(contact => !isFileTransferAssistant(contact)) : [];
};

export const useMsgStore = defineStore({
  id: 'message', // id必填，且需要唯一
  state: () => {
    return {
      pushSocket: '',
      chatSocket: '',
      instantSocket: '',
	  webrtc:'',
	  topContacts:[],
      contacts: [], //所有联系人
	  chatList:[], //聊天列表
      wsSendData: '',
      unread: 0,
	  sysUnread:0,
	  msgAt:0,
	  newMessage:{},
	  msgList:[],
	  webrtcLock:false
    }
  },
  // actions 用来修改 state
    actions: {
		catchSocketAction(data){
			this.chatSocket = data;
			if (data.is_group == 2) {
				this.unread += 1;
			}
		},
		updateUnread (data) {
			this.unread = parseInt(data);
		},
		/** 进入会话时清该联系人未读，并重算总未读。返回是否有变化。 */
		clearContactUnread(contactId) {
			const id = String(contactId || '')
			if (!id) return false
			const contacts = storedContacts()
			const contact = contacts.find((item) => String(item.id) === id)
			if (!contact || !(Number(contact.unread) > 0)) return false
			this.updateContacts({
				id,
				unread: 0
			})
			return true
		},
		//初始化联系人
		initContacts (data) {
			data = Array.isArray(data) ? data.filter(contact => !isFileTransferAssistant(contact)) : [];
			normalizeAvatarData(data);
			let contacts=utils.sortContacts(data);
			let topContacts=[];
			let otherContacts=[];
			let unread=0;
			let msgAt=0;
			contacts.forEach((item, index) => {
				if (item.lastContent && item.is_notice==1) {
					unread += item.unread;
				}
				if (item.is_at) {
					msgAt += item.is_at;
				}
				if(item.lastContent && !['text','event','location','contact'].includes(item.type)){
					item.lastContent=utils.getMsgType(item.type);
				}
				if(item.index=='群聊'){
					item.index="#";
				}
				if (item.is_top == 1) {
					topContacts.push(item)
				}else{
					otherContacts.push(item)
				}
			})
			this.unread=unread;
			this.msgAt=msgAt;
			this.contacts =topContacts.concat(otherContacts);
			uni.setStorageSync('allContacts',this.contacts);
		},
		//更新联系人
		updateContacts (data) {
			const contacts = storedContacts();
			// 更新联系人
			contacts.forEach((item, index) => {
				let contact = contacts[index];
				if (item.id == data.id) {
					contacts[index] = Object.assign(contact, data);
				}
			})
			this.initContacts(contacts);
		},
		//添加联系人
		appendContacts (data) {
			const contacts = storedContacts();
			// 检查是否有该联系人,有就更新,没有就增加
			const contact=contacts.filter(item=> item.id==data.id);
			if(contact.length>0){
				return this.updateContacts(data);
			}
			contacts.push(data);
			
			this.initContacts(contacts);
		},
		//删除联系人
		deleteContacts (data) {
			const contacts = JSON.parse(JSON.stringify(this.contacts));
			const newContacts = contacts.filter(obj => obj.id != data.id);
			this.contacts=newContacts;
			uni.setStorageSync('allContacts',this.contacts);
		},
		// 初始化当前页面的消息列表
		initMsg(data){
			this.msgList=normalizeAvatarData(data);
		},
		// 检查是否存在此消息,有的则跳过,没有就更新
		checkMsg(msg){
			normalizeAvatarData(msg);
			let msgList=this.msgList;
			let hasMsg=false;
			msgList.forEach((item, index) => {
				if (item.id==msg.id) {
				  hasMsg=true;
				}
			})
			if(!hasMsg){
				this.msgList.push(msg);
			}
		},
		getContact(id,message){
			let contacts=storedContacts();
			normalizeAvatarData(contacts);
			if(!id){
				return contacts;
			}else{
				let contact = contacts.find(obj => obj.id == id);
				if(!contact && message && message.contactInfo){
					contact = message.contactInfo
					this.appendContacts(contact);
				}
				return contact;
			}
		},
		getChatList(){
			let contacts=storedContacts();
			this.chatList =contacts.filter(obj => obj.lastContent);
			return this.chatList;
		},
		// 新消息推送
		appendMsg(message){
			this.newMessage=normalizeAvatarData(message);
		},
		wsSend (data) {
			this.wsSendData = data;
		}
    }
})
