<template>
	<view class="invite-hero">
		<view class="invite-hero-orbit invite-hero-orbit-one"></view>
		<view class="invite-hero-orbit invite-hero-orbit-two"></view>
		<view class="invite-hero-kicker">INVITE FRIENDS</view>
		<view class="invite-hero-title">好朋友，一起聊</view>
		<view class="invite-hero-description">分享专属邀请码，邀请朋友加入你的圈子</view>
		<view class="invite-hero-foot">
			<view class="invite-hero-badge"><text class="cuIcon-friendadd"></text></view>
			<text class="invite-hero-foot-note">每一份邀请，都是新的连接</text>
		</view>
	</view>
</template>

<style scoped>
.invite-hero { position:relative; overflow:hidden; min-height:332rpx; padding:39rpx 35rpx 28rpx; border-radius:36rpx; background:linear-gradient(125deg,#4167e6 0%,#5e6df0 54%,#8264e8 100%); color:#fff; box-shadow:0 18rpx 44rpx rgba(67,86,192,.2); }
.invite-hero-orbit { position:absolute; border-radius:50%; border:45rpx solid rgba(255,255,255,.06); }
.invite-hero-orbit-one { right:-122rpx; top:-200rpx; width:405rpx; height:405rpx; }
.invite-hero-orbit-two { left:-186rpx; bottom:-228rpx; width:340rpx; height:340rpx; }
.invite-hero-kicker { position:relative; font-size:19rpx; font-weight:650; letter-spacing:4rpx; opacity:.72; }
.invite-hero-title { position:relative; margin-top:16rpx; font-size:49rpx; line-height:1.22; font-weight:800; letter-spacing:1rpx; }
.invite-hero-description { position:relative; margin-top:16rpx; color:rgba(255,255,255,.83); font-size:22rpx; }
.invite-hero-foot { position:relative; display:flex; align-items:center; gap:14rpx; margin-top:39rpx; padding-top:19rpx; border-top:1rpx solid rgba(255,255,255,.2); }
.invite-hero-badge { display:flex; align-items:center; justify-content:center; width:60rpx; height:60rpx; border-radius:19rpx; background:rgba(255,255,255,.17); font-size:30rpx; }
.invite-hero-foot-note { color:rgba(255,255,255,.83); font-size:22rpx; }
</style>
