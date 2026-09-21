<template>
	<view class="cu-avatar lg" :class="appSetting.circleAvatar?'round':'radius'" @tap="openUserInfo(info)" :style="[{backgroundImage:'url('+ displayAvatar +')'}]"></view>
</template>
<script>
	const userInfo=uni.getStorageSync('userInfo');
	const appSetting=uni.getStorageSync('appSetting');
	import { useMsgStore } from '@/store/message';
	import { normalizeAvatarUrl, resolveAvatarDisplayUrl } from '@/utils/avatar.js'
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
export default{
	name  : "im-touch",
	props : {
		info:{type:Object, default:function(){return {};}},
		circleAvatar:{type:Boolean, default:false},
		profile:{type:Boolean, default:false},
	},
	data() {
		return {
			toucheTimer  : 0,
			fingerRes    : [],
			distance     : 0,
			taptimer     : 100,
			appSetting:appSetting,
			displayAvatar: ''
		}
	},
	watch: {
		info: {
			immediate: true,
			deep: true,
			handler(value) {
				this.refreshDisplayAvatar(value)
			}
		}
	},
	
	methods:{
		refreshDisplayAvatar(info) {
			const normalized = normalizeAvatarUrl(info && info.avatar, info || {})
			this.displayAvatar = normalized
			resolveAvatarDisplayUrl(normalized).then((url) => {
				if (normalizeAvatarUrl(this.info && this.info.avatar, this.info || {}) === normalized) {
					this.displayAvatar = url
				}
			}).catch(() => {})
		},
		// 打开用户详情
		openUserInfo(item){
			let friend=msgStore.getContact(item.user_id);
			if(!this.profile && !friend){
				uni.showToast({
					title:'已开启用户隐私！',
					icon:'none'
				})
				return false;
			}
			if(item.id==userInfo.user_id) return;
			uni.redirectTo({
				url:"/pages/contacts/detail?id="+this.info.id
			})
		}
	}
}
</script>
<style scoped></style>
