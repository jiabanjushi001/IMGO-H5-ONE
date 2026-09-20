<template>
	<view class="cu-avatar lg" :class="appSetting.circleAvatar?'round':'radius'" @tap="openUserInfo(info)" :style="[{backgroundImage:'url('+ info.avatar +')'}]"></view>
</template>
<script>
	const userInfo=uni.getStorageSync('userInfo');
	const appSetting=uni.getStorageSync('appSetting');
	import { useMsgStore } from '@/store/message';
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
			appSetting:appSetting
		}
	},
	
	methods:{
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