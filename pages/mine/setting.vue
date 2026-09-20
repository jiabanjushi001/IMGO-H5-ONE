<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
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
					<switch class="switch" @change="setVoice" :class="setting.voiceStatus?'checked':''" :checked="setting.voiceStatus"></switch>
				</view>
			</view>
			<view class="cu-item">
				<view class="content">
					<text>震动</text>
				</view>
				<view class="action">
					<switch class="switch" @change="setVibrate" :class="setting.vibrateStatus?'checked':''" :checked="setting.vibrateStatus"></switch>
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
					<switch class="switch" @change="setAvatar" :class="setting.circleAvatar?'checked':''" :checked="setting.circleAvatar"></switch>
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

</style>
