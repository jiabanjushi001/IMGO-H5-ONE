<script setup>
defineProps({
	date: { type: String, default: '' },
	signedToday: { type: Boolean, default: false },
	totalDays: { type: Number, default: 0 },
	loading: { type: Boolean, default: false },
	error: { type: Boolean, default: false }
})
</script>

<template>
	<view class="checkin-hero">
		<view class="hero-orbit hero-orbit-one"></view>
		<view class="hero-orbit hero-orbit-two"></view>
		<view class="hero-body">
			<view class="hero-copy">
				<view class="hero-eyebrow"><text class="hero-eyebrow-dot"></text> DAILY CHECK-IN</view>
				<view class="hero-title"><view>把每一天</view><view>都点亮</view></view>
				<view class="hero-description">今天的坚持，也值得被记住。</view>
				<view class="hero-date"><text class="cuIcon-calendar"></text>{{ date || '北京时间' }}</view>
			</view>
			<view class="hero-art" aria-hidden="true">
				<view class="hero-art-glow"></view>
				<view class="hero-calendar">
					<view class="hero-calendar-rings"><text></text><text></text></view>
					<view class="hero-calendar-mark">✓</view>
				</view>
				<view class="hero-spark hero-spark-one">✦</view>
				<view class="hero-spark hero-spark-two">✦</view>
			</view>
		</view>
		<view class="hero-stats">
			<view class="hero-stat">
				<text class="hero-stat-label">累计签到</text>
				<view class="hero-stat-value">{{ loading || error ? '—' : totalDays }}<text v-if="!loading && !error" class="hero-stat-unit">天</text></view>
			</view>
			<view class="hero-stat-divider"></view>
			<view class="hero-stat">
				<text class="hero-stat-label">今日状态</text>
				<view class="hero-stat-status"><text class="hero-status-dot" :class="{ 'is-signed': signedToday && !loading && !error, 'is-unknown': error }"></text>{{ loading ? '查询中' : error ? '待确认' : signedToday ? '已签到' : '待签到' }}</view>
			</view>
		</view>
	</view>
</template>

<style scoped>
.checkin-hero {
	position: relative;
	overflow: hidden;
	box-sizing: border-box;
	min-height: 420rpx;
	padding: 38rpx 36rpx 32rpx;
	border: 1rpx solid rgba(255, 255, 255, .55);
	border-radius: 36rpx;
	background: linear-gradient(128deg, #526dff 0%, #665ff1 55%, #805be3 100%);
	box-shadow: 0 24rpx 52rpx rgba(75, 84, 185, .22);
	color: #fff;
}
.hero-orbit { position: absolute; border: 1rpx solid rgba(255,255,255,.14); border-radius: 50%; pointer-events: none; }
.hero-orbit-one { width: 360rpx; height: 360rpx; top: -190rpx; right: -70rpx; }
.hero-orbit-two { width: 430rpx; height: 430rpx; top: -210rpx; right: -128rpx; }
.hero-body { position: relative; display: flex; align-items: center; min-height: 260rpx; }
.hero-copy { position: relative; z-index: 1; flex: 1; min-width: 0; }
.hero-eyebrow { display: flex; align-items: center; gap: 10rpx; color: rgba(255,255,255,.78); font-size: 19rpx; font-weight: 700; letter-spacing: 2.2rpx; }
.hero-eyebrow-dot { width: 9rpx; height: 9rpx; border-radius: 50%; background: #bdffdf; }
.hero-title { margin-top: 18rpx; font-size: 52rpx; line-height: 1.22; font-weight: 800; letter-spacing: 1rpx; }
.hero-description { margin-top: 18rpx; color: rgba(255,255,255,.79); font-size: 22rpx; line-height: 1.5; }
.hero-art { position: relative; flex: 0 0 198rpx; height: 216rpx; margin-left: 6rpx; }
.hero-art-glow { position: absolute; inset: 15rpx -5rpx -3rpx 0; border-radius: 50%; background: rgba(255,255,255,.18); filter: blur(14rpx); }
.hero-calendar { position: absolute; top: 36rpx; left: 17rpx; display: flex; align-items: center; justify-content: center; box-sizing: border-box; width: 163rpx; height: 150rpx; border: 10rpx solid rgba(255,255,255,.45); border-radius: 28rpx; background: linear-gradient(150deg,#ffffff,#e7eaff); box-shadow: 0 19rpx 38rpx rgba(42,35,135,.26); transform: rotate(9deg); }
.hero-calendar-rings { position: absolute; top: -20rpx; left: 37rpx; display: flex; gap: 42rpx; }
.hero-calendar-rings text { width: 10rpx; height: 28rpx; border-radius: 8rpx; background: #dfe3ff; box-shadow: 0 2rpx 2rpx rgba(0,0,0,.08); }
.hero-calendar-mark { display: flex; align-items: center; justify-content: center; width: 78rpx; height: 78rpx; border-radius: 50%; background: #586eff; color: white; font-size: 55rpx; font-weight: 700; box-shadow: 0 11rpx 21rpx rgba(77,100,225,.25); }
.hero-spark { position: absolute; color: #d7e2ff; font-size: 34rpx; }
.hero-spark-one { top: 0; right: 8rpx; }
.hero-spark-two { bottom: 7rpx; left: 1rpx; font-size: 25rpx; }
.hero-stats { position: relative; z-index: 1; display: flex; align-items: center; min-height: 102rpx; margin-top: 20rpx; padding: 18rpx 26rpx; border: 1rpx solid rgba(255,255,255,.2); border-radius: 25rpx; background: rgba(255,255,255,.15); box-sizing: border-box; }
.hero-stat { min-width: 0; flex: 1; }
.hero-stat-label { font-size: 21rpx; color: rgba(255,255,255,.73); }
.hero-stat-value { margin-top: 4rpx; font-size: 42rpx; font-weight: 800; line-height: 1; }
.hero-stat-unit { margin-left: 5rpx; font-size: 22rpx; font-weight: 500; }
.hero-stat-divider { width: 1rpx; height: 49rpx; margin: 0 25rpx; background: rgba(255,255,255,.3); }
.hero-stat-status { display: flex; align-items: center; gap: 9rpx; margin-top: 8rpx; font-size: 25rpx; font-weight: 700; white-space: nowrap; }
.hero-status-dot { width: 10rpx; height: 10rpx; border-radius: 50%; background: #ffe39b; }
.hero-status-dot.is-signed { background: #a2ffd1; }
.hero-status-dot.is-unknown { background: #dbe0ff; }
.hero-date { display: flex; align-items: center; gap: 9rpx; margin-top: 18rpx; color: rgba(255,255,255,.72); font-size: 20rpx; white-space: nowrap; }
</style>
