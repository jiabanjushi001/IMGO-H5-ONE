<template>
	<view class="rules-card">
		<view class="rules-title"><view class="rules-mark"></view>签到说明</view>
		<view class="rule-row"><view class="rule-number">01</view><view><view class="rule-heading">每天一次</view><view class="rule-copy">按北京时间自然日计算，每天只能签到一次。</view></view></view>
		<view class="rule-row"><view class="rule-number">02</view><view><view class="rule-heading">记录每一步</view><view class="rule-copy">累计天数来自实际签到记录，错过一天也不会清零。</view></view></view>
	</view>
</template>

<style scoped>
.rules-card { box-sizing: border-box; margin-top: 26rpx; padding: 34rpx 31rpx 19rpx; border: 1rpx solid #e7ebf4; border-radius: 30rpx; background: #fff; box-shadow: 0 12rpx 36rpx rgba(44,57,98,.07); }
.rules-title { display: flex; align-items: center; gap: 15rpx; margin-bottom: 21rpx; color: #252d45; font-size: 29rpx; font-weight: 700; }
.rules-mark { width: 8rpx; height: 34rpx; border-radius: 7rpx; background: linear-gradient(#526dff,#8968ee); }
.rule-row { display: flex; align-items: flex-start; gap: 20rpx; padding: 17rpx 0 22rpx; }
.rule-row + .rule-row { border-top: 1rpx solid #eef1f6; }
.rule-number { display: flex; align-items: center; justify-content: center; flex: 0 0 54rpx; width: 54rpx; height: 54rpx; border-radius: 17rpx; background: #eef1ff; color: #6173e6; font-size: 20rpx; font-weight: 800; }
.rule-heading { color: #303951; font-size: 25rpx; font-weight: 600; line-height: 1.5; }
.rule-copy { margin-top: 5rpx; color: #8994a8; font-size: 22rpx; line-height: 1.6; }
</style>
