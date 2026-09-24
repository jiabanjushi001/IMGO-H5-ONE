<template>
	<view class="invite-actions">
		<view class="invite-steps">
			<view class="invite-steps-title">邀请方式</view>
			<view class="invite-step"><text class="invite-step-number">01</text><text>复制邀请码发送给好友</text></view>
			<view class="invite-step"><text class="invite-step-number">02</text><text>好友完成注册</text></view>
			<view class="invite-step"><text class="invite-step-number">03</text><text>直属好友人数自动更新</text></view>
		</view>
	</view>
</template>

<style scoped>
.invite-steps { padding:30rpx 30rpx 8rpx; border:1rpx solid #e8edf6; border-radius:30rpx; background:#fff; }
.invite-steps-title { margin-bottom:23rpx; color:#2b3651; font-size:28rpx; font-weight:750; }
.invite-step { display:flex; align-items:center; gap:21rpx; min-height:67rpx; margin-bottom:13rpx; color:#56627b; font-size:23rpx; }
.invite-step-number { display:flex; align-items:center; justify-content:center; flex:none; width:48rpx; height:48rpx; border-radius:15rpx; background:#eef1ff; color:#5d71e5; font-size:19rpx; font-weight:800; }
</style>
