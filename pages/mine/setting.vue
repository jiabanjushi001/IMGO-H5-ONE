<template>
	<view class="setting-page">
		<cu-custom bgColor="text-white" bgStyle="background:linear-gradient(126deg,#4c63e9 0%,#5e70f5 55%,#6b6eeb 100%);color:#fff;" :isBack="true">
			<template #backText></template>
			<template #content>通用设置</template>
		</cu-custom>
		
		<view class="cu-bar bg-white solid-bottom margin-top">
			<view class="action">新消息</view>
		</view>
		
		<view class="cu-list menu">
			<view class="cu-item">
				<view class="content">
					<text>声音</text>
				</view>
				<view class="action">
					<switch class="switch theme-switch" @change="setVoice" :class="setting.voiceStatus?'checked':''" :checked="setting.voiceStatus" color="#4c63e9"></switch>
				</view>
			</view>
			<view class="cu-item">
				<view class="content">
					<text>震动</text>
				</view>
				<view class="action">
					<switch class="switch theme-switch" @change="setVibrate" :class="setting.vibrateStatus?'checked':''" :checked="setting.vibrateStatus" color="#4c63e9"></switch>
				</view>
			</view>
		</view>
		
		<view class="cu-bar bg-white solid-bottom margin-top">
			<view class="action">其他设置</view>
		</view>
		
		<view class="cu-list menu">
			<view class="cu-item">
				<view class="content">
					<text>圆形头像</text>
				</view>
				<view class="action">
					<switch class="switch theme-switch" @change="setAvatar" :class="setting.circleAvatar?'checked':''" :checked="setting.circleAvatar" color="#4c63e9"></switch>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	const loginStore = useloginStore(pinia)
	export default {
		data() {
			return {
				loginStore:loginStore,
				globalConfig:loginStore.globalConfig,
				setting:{
					voiceStatus:true,
					vibrateStatus:false,
					circleAvatar:false
				}
			}
		},
		created() {
			let setting=uni.getStorageSync('appSetting') ?? '';
			if(setting){
				this.setting=setting;
			}
		},
		methods: {
			setVoice(e){
				this.setting.voiceStatus=e.detail.value
				this.saveSet();
			},
			setVibrate(e){
				this.setting.vibrateStatus=e.detail.value
				this.saveSet();
			},
			setAvatar(e){
				this.setting.circleAvatar=e.detail.value
				this.saveSet();
			},
			saveSet(){
				loginStore.setAppSetting(this.setting)
			}
		}
	}
</script>

<style>
.setting-page {
	min-height: 100vh;
	background: #f4f6fb;
}
.mine-theme-header .cu-bar {
	background: linear-gradient(126deg, #4c63e9 0%, #5e70f5 55%, #6b6eeb 100%) !important;
	color: #fff;
}
.mine-theme-header .action,
.mine-theme-header .content,
.mine-theme-header .cuIcon-back,
.mine-theme-header .cu-custom .action,
.mine-theme-header .cu-custom .content {
	color: #fff !important;
}
switch.theme-switch.checked .uni-switch-input,
switch.theme-switch[checked] .uni-switch-input,
switch.theme-switch.checked .wx-switch-input,
switch.theme-switch[checked] .wx-switch-input,
.theme-switch.checked .uni-switch-input,
.theme-switch[checked] .uni-switch-input {
	background-color: #4c63e9 !important;
	border-color: #4c63e9 !important;
	color: #ffffff !important;
}
</style>
