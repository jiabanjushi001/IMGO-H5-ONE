<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>群信息</template>
		</cu-custom>
		<view align="center" class="groupInfo">
			<image :src="contact.avatar" mode="widthFix" style="width:120px;height:120px;border-radius: 12rpx;"></image>
			<view class="f-14 mt-10">
				{{contact.name}} ({{contact.groupUserCount}})
			</view>
		</view>
		<view class="padding flex flex-direction mt-10" v-if="!contact.isJoin">
			<button class="cu-btn bg-green lg"  @tap="applyGroup">
				加入群聊
			</button>
		</view>
		<view class="padding flex flex-direction mt-10" v-else>
			<button class="cu-btn bg-green lg"  @tap="openChat">
				进入聊天
			</button>
		</view>
	</view>
</template>
<script>
	import pinia from '@/store/index'
	import { useMsgStore } from '@/store/message';
	const msgStore = useMsgStore(pinia)
	export default {
		data() {
			return {
				contact:{},
				group_id:0,
				inviteToken: '',
				isNavigating: false
			}
		},
		onLoad(options) {
			this.group_id = options.group_id?options.group_id:''
			this.inviteToken = options.token || ''
			this.getGroupInfo()
		},
		methods: {
			getGroupInfo() {
				this.$api.msgApi.groupInfo({
					group_id: this.group_id,
					token: this.inviteToken
				}).then((res) => {
					if (res.code !== 0 || !res.data) return;
					let data=res.data;
					this.contact=data;
					
				})
			},
			async applyGroup() {
				if (this.isNavigating) return;
				if(this.contact.setting && this.contact.setting.invite==0){
					return uni.showToast({
						title:'该群聊已经关闭加群申请',
						icon:'none'
					})
				}
				uni.showLoading({
					title: '加入中'
				});
				try {
					const res = await this.$api.msgApi.joinGroup({
						group_id: this.group_id,
						token:this.inviteToken,
						inviteUid:this.contact.inviteUid
					});
					if (res.code === 0) {
						this.contact.isJoin = 1;
						await this.openChat();
					}
				} catch (e) {
					uni.showToast({ title: '加入群聊失败，请重试', icon: 'none' });
				} finally {
					uni.hideLoading();
				}
			},
			async openChat(){
				if (this.isNavigating) return;
				this.isNavigating = true;
				try {
					// A deep link may bypass the homepage, so hydrate the full contact first.
					const res = await this.$api.msgApi.initContacts();
					if (res.code !== 0 || !Array.isArray(res.data)) throw new Error('contacts unavailable');
					msgStore.initContacts(res.data);
					if (!msgStore.getContact(this.group_id)) throw new Error('group not in contacts');
					uni.navigateTo({
						url: '/pages/message/chat?id=' + encodeURIComponent(this.group_id)
					});
				} catch (e) {
					uni.showToast({ title: '群聊暂不可用，请刷新后重试', icon: 'none' });
				} finally {
					this.isNavigating = false;
				}
			}
		}
	}
</script>
<style scoped>
	.groupInfo{
		margin-top: 100rpx;
	}
</style>
