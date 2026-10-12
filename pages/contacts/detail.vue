<template>
	<view>
		<cu-custom bgColor="bg-white"  :isBack="true">
			<template #backText></template>
			<template #content>个人信息</template>
		</cu-custom>
		<view class="padding flex justify-start align-center">
			<view class='cu-avatar lg radius mr-15' @tap="showAvatar(detail)" :style="'background-image:url('+detail.avatar+')'">
			</view>
			<view class='im-flex im-justify-content-start im-columns'>
				<view class="mb-5">{{detail.realname}}</view>
				<view class="text-gray">{{detail.account}}</view>
			</view>
		</view>
		<view class="cu-list menu" v-if="globalConfig.sysInfo.runMode==2 && detail.friend && userInfo.user_id!=detail.user_id">
			<view class="cu-item" @tap="setNickname">
				<view class="content">
					<text class="cuIcon-edit text-green"></text>
					<text>备注</text>
				</view>
				<view class="action">
					<text class="text-grey text-sm">{{detail.friend.nickname || '未设置'}}</text>
					<text class="text-grey text-sm ml-5 cuIcon-write"></text>
				</view>
			</view>
		</view>
		
		<template class="" v-if="userInfo.user_id!=detail.user_id">
			<view class="padding flex flex-direction" v-if="globalConfig.sysInfo.runMode==1 || detail.friend">
				<button class="cu-btn bg-green mt-10 lg" @tap="sendMsg(detail)">发消息</button>
				<button class="cu-btn bg-blue mt-10 lg" v-if="validatePhone" @tap="callPhone()">打电话</button>
				<!-- #ifdef APP | H5 -->
				<button class="cu-btn bg-grey mt-10 lg" @tap="modelName='callRtc'" v-if="parseInt(globalConfig.chatInfo.webrtc) && parseInt(globalConfig.chatInfo.simpleChat)">音视频通话</button>
				<!-- #endif -->
				
				<button class="cu-btn bg-red  mt-10 lg" @tap="delFriend()" v-if="globalConfig.sysInfo.runMode==2">删除好友</button>
			</view>
			<view class="padding flex flex-direction" v-if="globalConfig.sysInfo.runMode==2 && !detail.friend">
				<button class="cu-btn bg-green lg" @tap="addFriend()">加好友</button>
			</view>
		</template>
		<CallModePicker
			:visible="modelName === 'callRtc'"
			:allow-prepared-video="canUsePreparedVideo"
			:show-voice="true"
			@close="modelName = ''"
			@start="startSelectedCall"
		/>
	</view>
</template>

<script>
	import { useMsgStore } from '@/store/message';
	import { useloginStore } from '@/store/login';
	import { normalizeMediaUrl, resolveMediaDisplayUrl } from '@/utils/avatar.js'
	import { canUsePreparedVideo } from '@/utils/call-video-permission.js'
	import CallModePicker from '@/components/call/CallModePicker.vue'
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
	const userStore = useloginStore(pinia)
	export default {
		components: { CallModePicker },
		data() {
			return {
				modelName:'',
				detail:{},
				userInfo:userStore.userInfo,
				globalConfig:userStore.globalConfig
			}
		},
		computed: {
			canUsePreparedVideo() {
				return canUsePreparedVideo(this.userInfo)
			},
		    validatePhone(){
				let reg = /^1[3456789]\d{9}$/;
				return reg.test(this.detail.account);
			}
		},
		onLoad(options) {
			this.$api.msgApi.getUserInfo({user_id:options.id}).then((res)=>{
				if(res.code==0){
					this.detail=res.data;
				}
			})
		}, 
		methods: {
			showAvatar(detail){
				resolveMediaDisplayUrl(detail.avatar).then((url) => {
					if (!url) {
						uni.showToast({ title: '头像加载失败', icon: 'none' })
						return
					}
					uni.previewImage({ urls: [url] })
				}).catch(() => {
					uni.showToast({ title: '头像加载失败', icon: 'none' })
				})
			},
			sendMsg(info){
				uni.reLaunch({
					url:"/pages/message/chat?id="+info.user_id
				})
			},
			callPhone(){
				uni.makePhoneCall({
					phoneNumber: this.detail.account
				});
			},
			startSelectedCall(selection) {
				this.calling(selection.type, selection.videoMode, selection.video)
			},
			calling(is_video, videoMode = '', video = null){
				if(msgStore.webrtcLock){
					return uni.showToast({
						title:'其他终端正在通话中',
						icon:'none'
					})
				}
				this.modelName='';
				let msg_id=this.$util.getUuid();
				const mode = is_video ? (videoMode || 'live') : ''
				const videoFileId = Number(video && video.file_id || 0)
				const videoUrl = video ? normalizeMediaUrl(video.url || '') : ''
				uni.navigateTo({
				  url: '/pages/message/call?msg_id='+msg_id+'&type='+is_video+'&status=1&id='+this.detail.user_id+'&name='+encodeURIComponent(this.detail.realname || '')+'&avatar='+encodeURIComponent(this.detail.avatar || '')+'&token='+encodeURIComponent(uni.getStorageSync('authToken') || '')+'&video_mode='+mode+'&video_file_id='+videoFileId+'&video_url='+encodeURIComponent(videoUrl)
				})
			},
			delFriend(){
				uni.showModal({
					title: '确定要删除该好友吗？',
					success: (res)=>{
						if (res.confirm) {
							let data={ id: this.detail.user_id};
							this.$api.friendApi.delFriend(data).then((res)=>{
								if(res.code==0){
									msgStore.deleteContacts(data);
									uni.reLaunch({
										url: '/pages/index/index'
									})
								}
							})
						}
					},
				})
				
			},
			addFriend(){
				uni.showModal({
					title: '请输入验证信息',
					editable:true,
					success: (res)=>{
						if (res.confirm) {
							if(res.content==''){
								return uni.showToast({
									title:'请输入备注！',
									icon:'error'
								})
							}
							this.$api.friendApi.addFriend({user_id:this.detail.user_id,remark:res.content}).then((e)=>{
								if(e.code==0){
									uni.showToast({
										title:'已申请',
										icon:'none'
									})
								}
							})
						}
					}
				});
			},
			setNickname(){
				let friend_id=this.detail.friend.friend_id ?? '';
				if(!this.detail.friend){
					return uni.showToast({
						title:'无法设置',
						icon:'error'
					})
				}
				uni.showModal({
					title: '请输入备注信息',
					editable:true,
					success: (res)=>{
						if (res.confirm) {
							if(res.content==''){
								return uni.showToast({
									title:'请输入好友备注！',
									icon:'error'
								})
							}
							this.$api.friendApi.setNickname({friend_id:friend_id,nickname:res.content}).then((e)=>{
								if(e.code==0){
									this.detail.friend.nickname=res.content;
									// 修改备注后修改联系人
									msgStore.updateContacts({
										id:this.detaild.user_id,
										displayName:res.content
									})
									uni.showToast({
										title:e.msg,
										icon:'none'
									})
								}
							})
						}
					}
				});
			}
		}
	}
</script>

<style lang="scss" scoped>

</style>
