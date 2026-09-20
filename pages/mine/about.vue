<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>关于IM</template>
		</cu-custom>
		<view style="height:150rpx;"></view>
		<view class="padding im-flex im-rows im-justify-content-center mb-10">
			<view class="im-flex im-rows im-justify-content-center">
				<image class="login-logo " :src="globalConfig.sysInfo.logo ?? packData.logo" mode="fixWidth"></image>
				
			</view>
		</view>
		<view class="app-name">{{globalConfig.sysInfo.name ?? packData.name}}</view>
		<view class="app-version mb-20 mt-10 c-999">Version {{packData.version}}</view>
		
		<view class="padding">
			<view class="mb-15">
				<view class="cuIcon-title">支持单聊和群聊</view>
			</view>
			<view class="mb-15">
				<view class="cuIcon-title">支持发送表情、图片、语音、视频和文件消息</view>
			</view>
			<view class="mb-15">
				<view class="cuIcon-title">单聊支持消息已读未读的状态显示，在线状态显示</view>
			</view>
			<view class="mb-15">
				<view class="cuIcon-title">群聊创建、删除和群成员管理、群公告、群禁言等</view>
			</view>
			<view class="mb-15">
				<view class="cuIcon-title">支持置顶联系人，消息免打扰</view>
			</view>
			<view class="mb-15">
				<view class="cuIcon-title">支持一对一音视频通话（已打通web端和移动端）</view>
			</view>
		</view>
		
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	const loginStore = useloginStore(pinia)
	import packageData from "../../package.json"
	export default {
		data() {
			return {
				loginStore:loginStore,
				globalConfig:loginStore.globalConfig,
				packData:packageData
			}
		},
		onShow() {
			
		}, 
		methods: {
			logout(){
				let client_id=uni.getStorageSync('client_id');
				this.$api.LoginApi.logout({client_id:client_id}).then(res => {
					if (res.code == 0) {
						loginStore.logout()
					}
				})
				
			},
			showSetting(){
				uni.showToast({
					title:'请在web端进行资料设置',
					icon:'none'
				})
			},
			editInfo(){
				uni.navigateTo({
					url:"/pages/mine/profile"
				})
			},
			setAvatar(){
				uni.navigateTo({
					url:"/pages/mine/avatar"
				})
			}
		}
	}
</script>

<style>
	.login-logo {
		width: 180rpx;
		height: 180rpx;
		font-size: 80rpx;
		text-align: center;
		line-height: 120rpx;
		border-radius: 18rpx;
	}
	.footer-version{
		width:100%;
		text-align: center;
		position: fixed;
		bottom: 40rpx;
	}
	.app-name{
		font-size:40rpx;
		font-weight: blod;
		text-align: center;
	}
	.app-version{
		text-align: center;
	}
</style>
