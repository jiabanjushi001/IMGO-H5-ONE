<script setup>
defineProps({ disabled: { type: Boolean, default: false } })
const emit = defineEmits(['copy-link'])
</script>

<template>
	<view class="invite-actions">
		<button class="invite-share-button" :disabled="disabled" @tap="emit('copy-link')"><text class="cuIcon-link"></text>复制邀请链接<text class="cuIcon-right"></text></button>
		<view class="invite-share-hint">发送给朋友，打开链接即可填写邀请码注册</view>
		<view class="invite-steps">
			<view class="invite-steps-title">邀请方式</view>
			<view class="invite-step"><text class="invite-step-number">01</text><text>复制邀请码或邀请链接</text></view>
			<view class="invite-step"><text class="invite-step-number">02</text><text>好友完成注册</text></view>
			<view class="invite-step"><text class="invite-step-number">03</text><text>直属好友人数自动更新</text></view>
		</view>
	</view>
</template>

<style scoped>
.invite-share-button { display:flex; align-items:center; justify-content:center; gap:12rpx; width:100%; height:92rpx; line-height:92rpx; margin:0; border:0; border-radius:25rpx; background:linear-gradient(112deg,#4c6bf0,#6a66ed); box-shadow:0 13rpx 30rpx rgba(78,101,227,.21); color:#fff; font-size:28rpx; font-weight:750; }
.invite-share-button::after { border:0; }
.invite-share-button[disabled] { opacity:.5; }
.invite-share-button .cuIcon-right { margin-left:5rpx; font-size:17rpx; }
.invite-share-hint { margin:18rpx 0 36rpx; text-align:center; color:#9aa4b8; font-size:20rpx; }
.invite-steps { padding:30rpx 30rpx 8rpx; border:1rpx solid #e8edf6; border-radius:30rpx; background:#fff; }
.invite-steps-title { margin-bottom:23rpx; color:#2b3651; font-size:28rpx; font-weight:750; }
.invite-step { display:flex; align-items:center; gap:21rpx; min-height:67rpx; margin-bottom:13rpx; color:#56627b; font-size:23rpx; }
.invite-step-number { display:flex; align-items:center; justify-content:center; flex:none; width:48rpx; height:48rpx; border-radius:15rpx; background:#eef1ff; color:#5d71e5; font-size:19rpx; font-weight:800; }
</style>
