
<template>
	<view class="chat-detail-page">
		<cu-custom class="detail-header" bgColor="bg-white" :isBack="true" :fallbackToHome="true">
			<template #backText></template>
			<template #content>聊天信息</template>
		</cu-custom>
		<view v-if="contact" class="detail-content">
			<view class="group-hero">
				<view class="group-hero-avatar">
					<AuthImage v-if="contact.avatar" class="group-hero-image" :src="contact.avatar" :info="contact" avatar mode="aspectFill" />
					<text v-else class="cuIcon-group"></text>
				</view>
				<view class="group-hero-info">
					<text class="group-hero-name">{{ contact.displayName }}</text>
					<text class="group-hero-subtitle">{{ is_group==1 ? `${groupUserCount} 位成员` : '聊天资料与偏好设置' }}</text>
				</view>
				<text class="group-hero-badge">{{ is_group==1 ? '群聊' : '私聊' }}</text>
			</view>

			<ChatMemberPanel
				:members="userList"
				:count="groupUserCount"
				:is-group="is_group==1"
				:can-add="canInvite"
				:can-manage="is_group==1 && isAuth"
				@member="openChatDetail"
				@add="editUser(2)"
				@manage="manageUser"
				@view-all="manageUser"
			/>

			<view v-if="is_group==1" class="detail-section">
				<view class="section-heading">群聊资料</view>
				<view class="detail-row avatar-row">
					<view class="row-icon name-icon cuIcon-pic"></view>
					<text class="row-label">群头像</text>
					<view class="row-value">{{ isAuth ? '点击更换' : '' }}</view>
					<avatar v-if="isAuth" :key="avatarEditorKey" :avatarSrc="contact.avatar" selWidth="240px" selHeight="480upx" expWidth="240px" expHeight="240px" avatarStyle="width: 76rpx; height: 76rpx; border-radius: 20rpx;" :quality="1" @upload="uploadGroupAvatar" />
					<AuthImage v-else class="avatar-row-image" :src="contact.avatar" :info="contact" avatar mode="aspectFill" />
				</view>
				<view class="detail-row" @tap="open">
					<view class="row-icon name-icon" :class="isAuth ? 'cuIcon-edit' : 'cuIcon-info'"></view>
					<text class="row-label">群聊名称</text>
					<text class="row-value">{{ contact.displayName }}</text>
					<text v-if="isAuth" class="cuIcon-right row-arrow"></text>
				</view>
				<view v-if="canInvite && showGroupQr" class="detail-row" @tap="openQr">
					<view class="row-icon qr-icon cuIcon-qrcode"></view>
					<text class="row-label">群二维码</text>
					<text class="row-value">分享群聊</text>
					<text class="cuIcon-right row-arrow"></text>
				</view>
				<view class="detail-row notice-row" @tap="openModel('notice')">
					<view class="row-icon notice-icon cuIcon-notification"></view>
					<view class="row-main">
						<text class="row-label">群公告</text>
						<text class="notice-preview" :class="{ empty: !contact.notice?.trim() }">{{ contact.notice?.trim() || '暂无公告，点击查看' }}</text>
					</view>
					<text class="cuIcon-right row-arrow"></text>
				</view>
				<view v-if="isAdmin" class="detail-row" @tap="openModel('manage')">
					<view class="row-icon manage-icon cuIcon-settings"></view>
					<text class="row-label">群管理</text>
					<text class="row-value">成员与权限设置</text>
					<text class="cuIcon-right row-arrow"></text>
				</view>
				<uni-popup ref="popup" type="dialog">
					<uni-popup-dialog mode="input" :value="contact.displayName" title="修改群名称" :duration="2000" :before-close="true" @close="closePop" @confirm="editGroupName" />
				</uni-popup>
			</view>

			<view class="detail-section">
				<view class="section-heading">消息设置</view>
				<view class="detail-row setting-row">
					<view class="row-icon mute-icon cuIcon-notificationforbidfill"></view>
					<view class="row-main"><text class="row-label">消息免打扰</text><text class="row-description">关闭此聊天的消息提醒</text></view>
					<switch class="switch detail-switch" @change="setIsNotice" :class="!contact.is_notice?'checked':''" :checked="!contact.is_notice?true:false" />
				</view>
				<view class="detail-row setting-row">
					<view class="row-icon pin-icon cuIcon-top"></view>
					<view class="row-main"><text class="row-label">置顶聊天</text><text class="row-description">始终显示在消息列表前面</text></view>
					<switch class="switch detail-switch" @change="setIsTop" :class="contact.is_top?'checked':''" :checked="contact.is_top?true:false" />
				</view>
			</view>

			<view class="detail-section">
				<view class="section-heading">聊天记录</view>
				<navigator class="record-navigator" :url="`/pages/message/record?id=${contact_id}`">
					<view class="detail-row record-row">
						<view class="row-icon record-icon cuIcon-record"></view>
						<text class="row-label">查看聊天记录</text>
						<text class="cuIcon-right row-arrow"></text>
					</view>
				</navigator>
			</view>

			<view v-if="is_group==1" class="detail-danger-actions">
				<view v-if="isAdmin" class="danger-action clear-action" @tap="clearMessage">清空聊天记录</view>
				<view class="danger-action leave-action" @tap="removeGroup">{{ isAdmin ? '解散群聊' : '退出群聊' }}</view>
			</view>
			<GroupNoticeEditor
				:visible="modelName=='notice'"
				:notice="contact.notice || ''"
				:group-name="contact.displayName || ''"
				:can-edit="isAuth"
				:saving="savingNotice"
				@cancel="closeModel"
				@save="editNotice"
			/>
			<view class="cu-modal bottom-modal manage-modal" :class="modelName=='manage'?'show':''" @tap.self="closeModel">
				<view class="cu-dialog manage-sheet">
					<view class="cu-bar bg-white manage-header">
						<view class="action text-gray" @tap="closeModel">取消</view>
						<text class="manage-title">群管理</text>
						<view class="action text-green" @tap="saveManage">保存</view>
					</view>
					<view class="manage-content">
						<view class="cu-list menu mt-15 bg-white">
							<view class="cu-item">
								<view class="content padding-tb-sm">
									<view>群资料编辑权限</view>
									<view class="text-gray text-sm">群头像、群名称和群公告仅群主与管理员可修改</view>
								</view>
								<view class="action text-gray text-sm">固定</view>
							</view>
							<view class="cu-item">
								<view class="content padding-tb-sm">
									<view>允许群成员邀请</view>
									<view class="text-gray text-sm">启用后，其他成员可以邀请其他人加入群聊</view>
								</view>
								<view class="action">
									<switch class="switch"  @change="setInvite" :class="contact.setting.invite=='1'?'checked':''" :checked="contact.setting.invite=='1'?true:false"></switch>
								</view>
							</view>
							<view class="cu-item">
								<view class="content padding-tb-sm">
									<view>允许成员查看历史消息</view>
									<view class="text-gray text-sm">启用后，新入群的成员可以查看所有的历史记录</view>
								</view>
								<view class="action">
									<switch class="switch"  @change="setHistory" :class="contact.setting.history=='1'?'checked':''" :checked="contact.setting.history=='1'?true:false"></switch>
								</view>
							</view>
							<view class="cu-item">
								<view class="content padding-tb-sm">
									<view>允许添加群成员为好友</view>
									<view class="text-gray text-sm">启用后，成员可以互相查看资料并添加为好友或发消息</view>
								</view>
								<view class="action">
									<switch class="switch"  @change="setProfile" :class="contact.setting.profile=='1'?'checked':''" :checked="contact.setting.profile=='1'?true:false"></switch>
								</view>
							</view>
							<uni-section title="群聊禁言" type="line">
								<radio-group class="block" @change="setSpeak">
									<view class="cu-form-group" v-for="item in radioList" :key="item.value">
										<view class="title">{{item.label}}</view>
										<radio :class="contact.setting.nospeak==item.value?'checked':''" :checked="contact.setting.nospeak==item.value?true:false" :value="item.value.toString()"></radio>
									</view>
								</radio-group>
							</uni-section>
						</view>
					</view>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
	import { useMsgStore } from '@/store/message';
	import pinia from '@/store/index';
	import { useloginStore } from '@/store/login';
	import ChatMemberPanel from '@/components/message/ChatMemberPanel.vue';
	import GroupNoticeEditor from '@/components/message/GroupNoticeEditor.vue';
	import avatar from '@/components/yq-avatar/yq-avatar.vue';
	import { normalizeAvatarUrl } from '@/utils/avatar.js';
	const userStore = useloginStore(pinia)
	const msgStore = useMsgStore(pinia)

	export default {
		components: {
			ChatMemberPanel,
			GroupNoticeEditor,
			avatar
		},
		data() {
			return {
				pageLoading: true,
				contact_id: null, //聊天id,
				is_group:0,
				groupUserCount:0,
				modelName:false,
				savingNotice:false,
				savingAvatar:false,
				avatarEditorKey:0,
				userList: [], //群成员
				allUser:[],
				userInfo:userStore.userInfo,
				chatRecordlist: [{
						text: '文本',
						icon: "icon-wenben",
						type: 'text'
		
					},
					{
						text: '图片',
						icon: "icon-zhaopian",
						type: 'image'
		
					}, {
						text: '文件',
						icon: "icon-wenjian",
						type: 'file'
		
					}, {
						text: '视频',
						icon: "icon-shipin",
						type: 'video'
		
					}, {
						text: '项目',
						icon: "icon-xiangmu_2",
						type: 'project'
		
					}, {
						text: '客户',
						icon: "icon-kehu",
						type: 'leads'
		
					},
				],
				radioList: [{
						label: "关闭",
						value: 0
					},
					{
						label: "仅管理员可发言",
						value: 1
					},
					{
						label: "仅群主可发言",
						value: 2
					},
				],
				isAuth: false, //判断自己是否是群管理或者群主
				contact: null, //联系人相关信息
				isAdmin:false, //如果为真，自己就是群主
				isManage: false, // 如果为真，自己就是管理
				user_ids: [],
				user:[],//全部群成员
			}
		},
		computed: {
			canInvite() {
				return this.is_group == 0 || this.isAuth || String(this.contact?.setting?.invite) === '1'
			},
			showGroupQr() {
				const value = userStore.globalConfig?.sysInfo?.showGroupQr
				return value !== '0' && value !== 0 && value !== false
			}
		},
		onShow() {
			this.isAdmin = false
			this.isManage = false
			this.isAuth = false
			this.getUserlist()
			if (this.is_group == 1 && this.contact_id) this.refreshGroupInfo()
		},
		onLoad: function(options) {
			this.is_group = options.is_group;
			this.contact_id = options.id;
			uni.$on('updateGroup', this.handleGroupUpdated);
			let contact=msgStore.getContact(this.contact_id);
			if(!contact){
				uni.showToast({
					title:'联系人不存在',
					icon:'none',
					duration:1500,
					complete:(res)=>{
						uni.reLaunch({
							url: '/pages/index/index'
						})
					}
				})
				return;
			}
			this.contact=contact;
			if(this.is_group==0){
				contact.userInfo={
					id:contact.user_id,
					account:contact.account,
					displayName:contact.displayName,
					avatar:contact.avatar
				}
				this.allUser.push(contact);
				this.userList.push(contact);
			}
		},
		onUnload() {
			uni.$off('updateGroup', this.handleGroupUpdated);
		},
		methods: {
			handleGroupUpdated(event) {
				const data = event?.data;
				if (event?.type === 'editGroupAvatar' && data?.group_id === this.contact_id && data.avatar && this.contact) {
					this.contact.avatar = normalizeAvatarUrl(data.avatar);
				}
			},
			async uploadGroupAvatar(cropped) {
				if (!this.isAuth || this.savingAvatar || !cropped?.path) return;
				this.savingAvatar = true;
				uni.showLoading({ title: '保存群头像中...' });
				try {
					const file = await new Promise((resolve, reject) => {
						uni.uploadFile({
							url: this.$api.msgApi.uploadUrl,
							filePath: cropped.path,
							name: 'file',
							header: { Authorization: uni.getStorageSync('authToken') },
							formData: { ext: 'png' },
							success: result => {
								try { resolve(JSON.parse(result.data)); } catch (error) { reject(error); }
							},
							fail: reject
						});
					});
					if (file.code !== 0 || !file.data?.file_id) throw new Error(file.msg || '图片上传失败');
					const saved = await this.$api.msgApi.editGroupAvatar({ id: this.contact_id, file_id: file.data.file_id });
					if (saved.code !== 0 || !saved.data?.avatar) throw new Error(saved.msg || '保存群头像失败');
					this.contact.avatar = saved.data.avatar;
					msgStore.updateContacts({ id: this.contact_id, avatar: saved.data.avatar });
					uni.showToast({ title: '群头像已更新', icon: 'none' });
				} catch (error) {
					this.avatarEditorKey += 1;
					uni.showToast({ title: error.message || '更换群头像失败', icon: 'none' });
				} finally {
					uni.hideLoading();
					this.savingAvatar = false;
				}
			},
			async refreshGroupInfo() {
				try {
					const res = await this.$api.msgApi.groupInfo({ group_id: this.contact_id });
					if (res.code !== 0 || !res.data || !this.contact) return;
					const role = Number(res.data.isJoin);
					this.isAdmin = role === 1;
					this.isManage = role === 2;
					this.isAuth = this.isAdmin || this.isManage;
					this.contact.notice = res.data.notice ?? '';
					this.contact.displayName = res.data.displayName || this.contact.displayName;
					this.contact.avatar = res.data.avatar || this.contact.avatar;
					this.groupUserCount = Number(res.data.groupUserCount) || 0;
					msgStore.updateContacts({
						id: this.contact_id,
						notice: this.contact.notice,
						displayName: this.contact.displayName,
						avatar: this.contact.avatar
					});
				} catch (error) {
					console.warn('获取群资料失败', error);
				}
			},
			openModel(model){
				this.modelName=model;
			},
			closeModel(){
				this.modelName=false;
			},
			saveManage(){
				if(!this.isAdmin) return;
				this.$api.msgApi.groupSetting({
					id: this.contact.id,
					setting: this.contact.setting
				})
				this.modelName=false;
			},
			setInvite(e){
				this.contact.setting.invite=e.detail.value ? '1' : '0';
			},
			setHistory(e){
				this.contact.setting.history=e.detail.value ? '1' : '0';
			},
			setProfile(e){
				this.contact.setting.profile=e.detail.value ? '1' : '0';
			},
			setSpeak(e){
				this.contact.setting.nospeak=e.detail.value;
			},
			setIsNotice(e){
				this.contact.is_notice=e.detail.value ? 0 : 1;
				this.$api.msgApi.isNoticeAPI({
					id: this.contact.id,
					is_group:this.contact.is_group,
					is_notice:this.contact.is_notice
				})
			},
			setIsTop(e){
				this.contact.is_top=e.detail.value ? 1 : 0;;
				this.$api.msgApi.setChatTopAPI({
					id: this.contact.id,
					is_group:this.contact.is_group,
					is_top:this.contact.is_top
				})
			},
			async editNotice(notice){
				if(!this.isAuth || this.savingNotice) return;
				this.savingNotice=true;
				try {
					const res = await this.$api.msgApi.setNotice({ id: this.contact.id, notice });
					if (res.code !== 0) {
						uni.showToast({ title: res.msg || '保存群公告失败', icon: 'none' });
						return;
					}
					this.contact.notice=notice;
					msgStore.updateContacts({ id: this.contact.id, notice });
					this.modelName=false;
				} catch (error) {
					uni.showToast({ title:'保存群公告失败，请重试', icon:'none' });
				} finally {
					this.savingNotice=false;
				}
			},
			open() {
				if (!this.isAuth) return;
				this.$refs.popup.open()
			},
			openQr() {
				uni.navigateTo({
					url: '/pages/index/qrcode?group_id='+ this.contact.id
				})
			},
			async editGroupName(value){
				if (!this.isAuth) return;
				const displayName = String(value || '').trim();
				if (!displayName) {
					uni.showToast({ title: '请输入群名称', icon: 'none' });
					return;
				}
				try {
					const res = await this.$api.msgApi.editGroupName({ id: this.contact.id, displayName });
					if (res.code !== 0) {
						uni.showToast({ title: res.msg || '修改群名称失败', icon: 'none' });
						return;
					}
					this.contact.displayName = displayName;
					msgStore.updateContacts({ id: this.contact.id, displayName });
					this.$refs.popup.close();
				} catch (error) {
					uni.showToast({ title: '修改群名称失败', icon: 'none' });
				}
			},
			closePop(){
				this.$refs.popup.close()
			},
			//移除群聊
			removeGroup() {
				// 如果是群主就解散群聊，否则就退出群聊
				let txt="退出群聊";
				if(this.isAdmin) txt="解散群聊";
				uni.showModal({
					title: '确定要'+txt+'吗?',
					success: e => {
						if (e.confirm) {
							if(this.isAdmin){
								this.$api.msgApi.removeGroup({id:this.contact.id}).then((res)=>{
									// 删除之后返回首页
									uni.reLaunch({
										url: '/pages/index/index'
									})
								})
							}else{
								this.$api.msgApi.removeUser({id:this.contact.id,user_id:this.userInfo.user_id}).then((res)=>{
									// 删除之后返回首页
									uni.reLaunch({
										url: '/pages/index/index'
									})
								})
							}
							
						}
					}
				});
				
			},
			clearMessage() {
				// 如果是群主就解散群聊，否则就退出群聊
				if(!this.isAdmin) {
					uni.showToast({
						title:'无权操作',
						icon:'none'
					})
				};
				uni.showModal({
					title: '清除后该群所有人的记录都会被删除，确定继续吗?',
					success: e => {
						if (e.confirm) {
							this.$api.msgApi.clearMessage({id:this.contact.id}).then((res)=>{
								uni.showToast({
									title:'清除成功',
									icon:'none'
								})
							})
						}
					}
				});
				
			},
			// 添加群成员
			editUser(type) {
				this.user_ids = this.allUser.map(item => item.user_id)
				if(this.contact.is_group==0){
					type=1
				}
				uni.navigateTo({
					url: '/pages/index/userSelection?type='+type+'&contact_id=' + this.contact.id
				})
			},
			// 管理群成员
			manageUser() {
				uni.navigateTo({
					url: '/pages/message/group/groupUser?group_id=' + this.contact.id
				})
			},
			// 跳转到聊天记录
			goChatRecord(type) {
				uni.navigateTo({
					url: '/package/message/pages/chatRecord/chatRecord?type=' + type + '&toContactId=' + this.contact_id + '&is_group=1'
				})
			},
			// 获取群成员列表
			getUserlist() {
				if(this.is_group==0) return;
				this.userList = []
				this.$api.msgApi.groupUserList({
					group_id: this.contact_id,
					limit:20,
				}).then(res => {
					this.user = res.data
					if (res.code !== 0) return
					this.allUser=JSON.parse(JSON.stringify(res.data));
					if (res.data.length > 18) {
						if (this.isAuth) {
							this.userList = res.data.splice(0, 18)
						}else if(this.contact.setting.invite){
							this.userList = res.data.splice(0, 19)
						} else {
							this.userList = res.data.splice(0, 20)
						}
					} else {
						this.userList = res.data
					}
					this.groupUserCount=res.count;
					this.pageLoading = false;
				})
			},
			// 打开联系人详情
			openChatDetail(item){
				if(this.userInfo.user_id==item.id) return;
				let friend=msgStore.getContact(item.id);
				if(this.contact.role<3 || this.contact.setting.profile=='1' || friend){
					uni.navigateTo({
						url:"/pages/contacts/detail?id="+item.id
					})
				}else{
					uni.showToast({
						title:'已开启用户隐私！',
						icon:'none'
					})
					return false;
				}
			}
		}
	}
</script>

<style lang="scss" scoped>
.chat-detail-page {
	box-sizing:border-box;
	min-height:100vh;
	background:radial-gradient(circle at 93% 8%,rgba(117,104,246,.11),transparent 30%),#f4f7fc;
}
:deep(.detail-header .cu-bar) {
	border-bottom:1rpx solid #e9edf6;
	background:rgba(255,255,255,.95) !important;
	box-shadow:0 8rpx 24rpx rgba(39,53,94,.04);
	color:#1d263b;
}
:deep(.detail-header .content) { font-weight:700; color:#1d263b; }
.detail-content { display:flex; flex-direction:column; gap:22rpx; padding:26rpx 22rpx 64rpx; }
.group-hero { position:relative; display:flex; overflow:hidden; align-items:center; gap:20rpx; box-sizing:border-box; min-height:175rpx; padding:27rpx 29rpx; border-radius:32rpx; background:linear-gradient(125deg,#5269ed 0%,#675feb 58%,#8066ef 100%); color:#fff; box-shadow:0 18rpx 42rpx rgba(75,83,178,.19); }
.group-hero::after { position:absolute; top:-110rpx; right:-100rpx; width:250rpx; height:250rpx; border:35rpx solid rgba(255,255,255,.08); border-radius:50%; content:''; pointer-events:none; }
.group-hero-avatar { z-index:1; display:flex; overflow:hidden; flex:none; align-items:center; justify-content:center; width:104rpx; height:104rpx; border:5rpx solid rgba(255,255,255,.65); border-radius:29rpx; background:rgba(255,255,255,.16); box-shadow:0 12rpx 28rpx rgba(40,41,116,.16); font-size:42rpx; }
.group-hero-image { width:100%; height:100%; }
.group-hero-info { z-index:1; display:flex; overflow:hidden; flex:1; flex-direction:column; gap:10rpx; }
.group-hero-name { display:block; overflow:hidden; white-space:nowrap; text-overflow:ellipsis; font-size:34rpx; font-weight:750; }
.group-hero-subtitle { font-size:23rpx; color:rgba(255,255,255,.78); }
.group-hero-badge { z-index:1; flex:none; padding:8rpx 15rpx; border:1rpx solid rgba(255,255,255,.25); border-radius:20rpx; background:rgba(255,255,255,.14); font-size:20rpx; }
.detail-section { overflow:hidden; border:1rpx solid #e9edf7; border-radius:30rpx; background:#fff; box-shadow:0 12rpx 34rpx rgba(48,61,107,.055); }
.section-heading { padding:27rpx 29rpx 12rpx; color:#1b243a; font-size:29rpx; font-weight:700; }
.detail-row { display:flex; align-items:center; gap:17rpx; box-sizing:border-box; min-height:100rpx; margin:0 27rpx; padding:16rpx 0; }
.detail-row + .detail-row { border-top:1rpx solid #edf0f7; }
.avatar-row-image { flex:none; width:76rpx; height:76rpx; border-radius:20rpx; }
.avatar-row :deep(.my-avatar) { display:block; flex:none; }
.record-navigator { display:block; margin:0 27rpx; }
.record-row { margin:0; }
.record-row .row-arrow { margin-left:auto; }
.row-icon { display:flex; flex:none; align-items:center; justify-content:center; width:54rpx; height:54rpx; border-radius:17rpx; font-size:28rpx; }
.name-icon { background:#eef1ff; color:#536dff; }
.qr-icon { background:#ecf7ff; color:#479ac9; }
.notice-icon { background:#fff4e8; color:#ed9d43; }
.manage-icon { background:#f3efff; color:#886ad4; }
.mute-icon { background:#f3f4ff; color:#6e76d9; }
.pin-icon { background:#eaf8f2; color:#44af80; }
.record-icon { background:#eff3ff; color:#5675db; }
.row-label { flex:none; color:#29334a; font-size:26rpx; font-weight:550; }
.row-value { overflow:hidden; flex:1; white-space:nowrap; text-overflow:ellipsis; text-align:right; color:#919ab0; font-size:23rpx; }
.row-arrow { flex:none; color:#b7c0d1; font-size:20rpx; }
.notice-row { align-items:flex-start; min-height:135rpx; }
.notice-row .row-icon,.notice-row .row-arrow { margin-top:6rpx; }
.row-main { display:flex; overflow:hidden; flex:1; flex-direction:column; gap:7rpx; }
.notice-preview { display:-webkit-box; overflow:hidden; -webkit-box-orient:vertical; -webkit-line-clamp:2; color:#8b94a8; font-size:22rpx; line-height:1.45; overflow-wrap:anywhere; }
.notice-preview.empty { color:#b0b8c9; }
.setting-row { min-height:109rpx; }
.row-description { color:#a1a9b9; font-size:21rpx; }
.detail-switch { flex:none; margin-right:0; transform:scale(.85); transform-origin:right center; }
.detail-danger-actions { display:flex; flex-direction:column; gap:15rpx; margin-top:3rpx; }
.danger-action { display:flex; align-items:center; justify-content:center; min-height:91rpx; border:1rpx solid #e9edf7; border-radius:25rpx; background:#fff; font-size:25rpx; font-weight:600; }
.clear-action { color:#db965a; }
.leave-action { color:#e45e70; }
.manage-modal { background:rgba(21,29,52,.46); }
.manage-modal .manage-sheet { overflow:hidden; border-radius:34rpx 34rpx 0 0; background:#f7f9fd; }
.manage-header { display:flex; justify-content:space-between; color:#263149; }
.manage-title { font-size:30rpx; font-weight:700; }
.manage-content { max-height:65vh; overflow-y:auto; padding:0 17rpx calc(20rpx + env(safe-area-inset-bottom)); text-align:left; }
.manage-content .cu-list.menu { border-radius:24rpx; }
.manage-content .cu-list.menu > .cu-item { padding:10rpx 23rpx; }
.manage-content .cu-list.menu > .cu-item .content { font-size:25rpx; }
.manage-content .cu-form-group { min-height:90rpx; }
</style>
