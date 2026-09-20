<script setup>
import { computed } from 'vue'

const props = defineProps({
	directCount: { type: Number, default: 0 },
	streakDays: { type: Number, default: 0 },
	loading: { type: Boolean, default: false },
	error: { type: Boolean, default: false }
})

const metrics = computed(() => [
	{ key: 'friends', title: '已邀请好友', value: props.directCount, unit: '人', icon: 'cuIcon-friendadd' },
	{ key: 'streak', title: '连续签到', value: props.streakDays, unit: '天', icon: 'cuIcon-calendar' }
])
</script>

<template>
	<view class="invite-metrics">
		<view v-for="metric in metrics" :key="metric.key" class="metric-card" :class="`metric-${metric.key}`">
			<view class="metric-icon"><text :class="metric.icon"></text></view>
			<view class="metric-copy">
				<text class="metric-label">{{ metric.title }}</text>
				<view class="metric-value">{{ loading || error ? '—' : metric.value }}<text v-if="!loading && !error" class="metric-unit">{{ metric.unit }}</text></view>
			</view>
		</view>
	</view>
</template>

<style scoped>
.invite-metrics { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:17rpx; margin-top:24rpx; }
.metric-card { display:flex; align-items:center; gap:16rpx; box-sizing:border-box; min-width:0; min-height:146rpx; padding:19rpx 17rpx; border:1rpx solid #e8ecf7; border-radius:23rpx; background:#fff; box-shadow:0 9rpx 23rpx rgba(43,55,104,.05); }
.metric-icon { display:flex; flex:none; align-items:center; justify-content:center; width:62rpx; height:62rpx; border-radius:18rpx; background:#edf1ff; color:#566ff1; font-size:32rpx; }
.metric-streak .metric-icon { background:#e8f5ff; color:#397ed7; }
.metric-copy { min-width:0; }
.metric-label { display:block; color:#777f94; font-size:21rpx; white-space:nowrap; }
.metric-value { margin-top:5rpx; color:#273663; font-size:35rpx; font-weight:800; line-height:1.2; font-variant-numeric:tabular-nums; white-space:nowrap; }
.metric-unit { margin-left:3rpx; font-size:22rpx; font-weight:650; }
@media screen and (max-width:350px) {
	.metric-card { gap:10rpx; padding:16rpx 12rpx; }
	.metric-icon { width:52rpx; height:52rpx; font-size:27rpx; }
	.metric-label { font-size:19rpx; }
	.metric-value { font-size:31rpx; }
}
</style>
