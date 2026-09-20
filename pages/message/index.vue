<template>
	
	<view class="message-page">
<!-- 		<view class="socket-status pd-10 im-flex justify-between im-align-items-center" v-if="!socketStatus">
			<view class="cuIcon-infofill text-red f-18"></view>
			<view class="c-666 f-12"> WS通信已断开，检查网络设置是否正常</view>
			<view @tap="reconnect()">重连</view>
		</view> -->
		
		<SystemNoticeBar :top="CustomBar" @visibility-change="noticeBarVisible = $event" />
		<view class="pr-10 pl-10 text-gray bg-white im-flex im-space-between im-align-items-center cu-bar fixed message-filter-bar" :style="{ top: (CustomBar + (noticeBarVisible ? 38 : 0)) + 'px' }">
			<im-tab class="mr-10" :values="values" @change="changeChat"></im-tab>
			<view class="socket-status pd-5 im-flex justify-between im-align-items-center radius-10 im-flex1" v-if="!socketStatus">
				<view class="cuIcon-infofill text-red f-18"></view>
				<view class="c-666 f-12">通信断开</view>
				<view class="cuIcon-refresh" @tap="reconnect()"></view>
			</view>
		</view>
		
		<view class="im-message-list conversation-list" :style="{ marginTop: noticeBarVisible ? 'calc(100rpx + 38px)' : '100rpx' }">
			<view class="cu-list menu-avatar" :style="{paddingBottom: paddingB+'px'}" v-if="msgsIn.length>0">
				<view class="cu-item second" :class="[modalName=='move-box-'+ index?'move-cur':'',item.is_top==1 ? 'top-contacts' : '']" v-for="(item, index) in msgsIn"  :key="index"
				 @touchstart="ListTouchStart" @touchmove="ListTouchMove" @touchend="ListTouchEnd" @tap="openChat(item.id)" :data-target="'move-box-' + index">
					<view class="cu-avatar lg" :class="appSetting.circleAvatar?'round':'radius'" :style="[{backgroundImage:'url('+ item.avatar +')'}]">
						<view class="online-status"  v-if="item.is_online && item.is_group==0 && globalConfig.chatInfo.online==1"></view>
					</view>
					<view class="content">
						<view class="c-333">
							<view class="text-overflow f-16" style="width:80%">
								{{item.displayName}}
							</view>
						</view>
						<view class="im-flex im-justify-content-start im-align-items-start lh-20x" style="height: 50rpx;overflow:hidden">
							<view class="text-red f-12 mr-5" v-if="item.is_at">[有{{item.is_at}}人@我] </view>
							<mp-html :content="emojiToHtml(item.lastContent)" class="im-flex f-12 text-gray text-overflow no-click"/>
						</view>
						
					</view>
					<view class="action">
						<view class="text-grey text-xs" >{{from_time(item.lastSendTime)}}</view>
						<view class="cu-tag round sm" :class="item.is_notice ? 'bg-red' : 'bg-notremind'" v-if="item.unread>0">{{item.unread}}</view>
						<view class="c-999" v-if="item.is_notice==0 && item.unread==0">
							<text class="cuIcon-musicforbidfill"></text>
						</view>
					</view>
					<view class="move second">
						<view class="bg-grey" v-if="item.is_top==1" @tap="btnTap(0,item)">取消置顶</view>
						<view class="bg-blue" v-else  @tap="btnTap(0,item)">置顶聊天</view>
						<view class="bg-orange" v-if="item.is_notice==1"  @tap="btnTap(2,item)">免扰</view>
						<view class="bg-orange" v-else  @tap="btnTap(2,item)">取消免扰</view>
					</view>
				</view>
			</view>
			<Empty v-else noDatatext="暂无聊天" textcolor="#999" />
		</view>
	</view>
</template>

<script>
	import statusPoint from '@/components/status.vue'
	import imTab from '@/components/message/im-tab.vue'
	import SystemNoticeBar from '@/components/message/SystemNoticeBar.vue'
	import emoji from '@/utils/emoji.js'
	import { storeToRefs } from 'pinia';
	import { useMsgStore } from '@/store/message';
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
	const {unread,msgAt} = storeToRefs(msgStore);
	const userStore = useloginStore(pinia);
	export default {
		components: {
			statusPoint,imTab,SystemNoticeBar
		},
		data() {
			return {
				navCurrent: 0,
				pageLoading: true,
				socketStatus:true,
				socketStatusHandler:null,
				damping   : 0.29,
				moveIndex : -1,
				touchStart:false,
				modalName: null,
				listTouchStart: 0,
				listTouchDirection: null,
				emojiMap:[],
				chatStatus:true,
				paddingB:0,
				msgAt:msgStore.msgAt,
				appSetting:userStore.appSetting,
				globalConfig:userStore.globalConfig,
				noticeBarVisible:false,
			active:0,
			triggered:true,
			values:[
					{
						id:1,
						name:'所有',
						count:0
					},
					{
						id:2,
						name:'未读',
						count:unread
					},
					{
						id:3,
						name:'@我',
						count:msgAt
					}
				]
			}
		},
		computed:{
			msgsIn(){
				let index=this.active;
				let contactList = [];
				const contacts = msgStore.contacts;
				if(index==1){
					contactList = contacts.filter(obj => obj.unread>0);
				}else if(index==2){
					contactList = contacts.filter(obj => obj.is_at>0);
				}else{
					contactList = contacts.filter(obj => obj.lastContent);
				}
				return contactList;
			}
		},
		mounted(){
			this.moveIndex = -1;
			// 监听ws网路连接状况
			this.socketStatusHandler = (e)=>{
				this.socketStatus=e;
			}
			uni.$on('socketStatus', this.socketStatusHandler)
		},
		beforeUnmount(){
			if (this.socketStatusHandler) {
				uni.$off('socketStatus', this.socketStatusHandler)
				this.socketStatusHandler = null
			}
		},
		created:function(){
			let emojiMap=[];
			// 解析所有表情
			emoji.forEach(function (item) {
				let child=item.children;
			  if(child.length>0){
				  child.forEach(function (val) {
					  let name=val.name;
					  let src=val.src;
					  emojiMap[name]=src;
				  })
			  }
			});
			this.emojiMap=emojiMap;
			// #ifdef H5
			this.paddingB=this.inlineTools;
			// #endif
			
			// #ifndef H5
			this.paddingB=this.navBarHeight+this.inlineTools;
			// #endif
		},
		methods: {
			initContacts(){
				uni.$emit('initContacts',true)	
				 this.triggered = true;
				 setTimeout(() => {
					this.triggered = false;
				}, 1000)
			    
			},
			changeChat(item,index){
				this.active=index;
			},
			btnTap: function(index, contact) {
				// 第一个按钮被点击 置顶、取消置顶消息
				if (index == 0) {
					contact.is_top=contact.is_top==0 ? 1 : 0;
					this.$api.msgApi.setChatTopAPI({
						  id: contact.id,
						  is_top:contact.is_top,
						  is_group: contact.is_group
						}).then(res => {
						  if (res.code == 0) {
							msgStore.updateContacts(contact);
						  }
						});
				}
				// 第三个按钮被打开 [ 删除消息 ]
				else if (index == 1) {
					uni.showModal({
						title: '确定要删除吗?',
						success: e => {
							if (e.confirm) {
								// this.$api.msgApi.delChatAPI({
								// 	  id: contact.id,
								// 	  is_group: contact.is_group
								// 	}).then(res => {
								// 	  if (res.code == 0) {
								// 		msgStore.deleteContacts(contact);
								// 	  }
								// 	});
							}
						}
					});
				}
				// 消息免打扰
				else if (index == 2) {
					contact.is_notice=contact.is_notice==0 ? 1 : 0;
					this.$api.msgApi.isNoticeAPI({
						  id: contact.id,
						  is_notice:contact.is_notice,
						  is_group: contact.is_group
						}).then(res => {
						  if (res.code == 0) {
							msgStore.updateContacts(contact);
						  }
						});
				}
			},
			reconnect(){
				uni.showLoading({
					title:'重连中...'
				})
				this.socketIo.connectSocketInit({type:'ping'});
				setTimeout(()=>{
					uni.hideLoading()
				},1500)
			},
			// 自动解析消息中的表情
			emojiToHtml(str){
				if(!str){
					return;
				}
				let emojiMap=this.emojiMap;
				return str.replace(/\[!(\w+)\]/gi, function (str, match) {
					var file = match;
					return emojiMap[file] ? "<img class='mr-5' style=\"width:18px;height:18px\" emoji-name=\"".concat(match, "\" src=\"").concat(emojiMap[file], "\" />") : "[!".concat(match, "]");
				  });
			},
			// ListTouch触摸开始
			ListTouchStart(e) {
				this.listTouchStart = e.touches[0].pageX
			},
			// ListTouch计算方向
			ListTouchMove(e) {
				let cux=e.touches[0].pageX;
				let direction=cux- this.listTouchStart;
				// 左滑的距离需要达到100才能滑动出菜单,否则在页面自然滚动的时候就会吧菜单给拖出来了
				if(Math.abs(direction)>100 && direction<0){
					this.listTouchDirection='left';
				}else{
					this.listTouchDirection='right';
				}
			},
			// ListTouch计算滚动
			ListTouchEnd(e) {
				if (this.listTouchDirection == 'left') {
					this.modalName = e.currentTarget.dataset.target
					this.chatStatus=false;
				} else {
					this.modalName = null
				}
				this.listTouchDirection = null
			},
			openChat(id){
				// 如果左滑工具栏在开启的情况下不能够点击进入聊天
				if(this.chatStatus){
					uni.navigateTo({
						url:"/pages/message/chat?id=" + id,
						animationType:"slide-in-right"
					})
				}else{
					this.chatStatus=true;
				}
			},
			from_time(time){
				return this.$util.timeFormat(time);
			},
		}
	}
</script>

<style lang="scss">
	.container {
		padding: 20px;
		font-size: 14px;
		line-height: 24px;
	}
	.border-b{
		border-bottom: 1px solid #eee;
	}
	.socket-status{
		background-color: #fde0de;
		height:72rpx;
		border-radius: 36rpx;
		padding:10rpx 16rpx;
	}
	.text-overflow ::v-deep uni-text , .text-overflow ::v-deep uni-text span{
		overflow: hidden !important;
		text-overflow: ellipsis;
		white-space: nowrap !important;
		width:300rpx;
	}
	.top-contacts{
		background-color: #f5f5f5 !important;
	}
	.cu-list.menu-avatar>.cu-item{
		padding-right:30rpx;
		.content{
			line-height: unset;
		}
	}
	.no-click{
		pointer-events:none;
		height:50rpx;
		flex:1;
	}
	.text-overflow ::v-deep ._block ._a{
		color: #aaaaaa !important;
	}
	.bg-notremind{
		background-color: #d1d1d1;
		color: #ffffff;
	}
	.cu-bar.fixed, .nav.fixed{
		z-index:900;
	}
</style>
