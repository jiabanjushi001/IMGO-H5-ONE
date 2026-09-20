<script setup>
import { computed } from 'vue'

const props = defineProps({
	loading: { type: Boolean, default: false },
	error: { type: Boolean, default: false },
	signed: { type: Boolean, default: false },
	totalDays: { type: Number, default: 0 }
})
const emit = defineEmits(['open'])
const description = computed(() => {
	if (props.error) return '状态获取失败，可进入签到页查看'
	if (props.loading) return '正在获取今日签到状态…'
	return props.signed ? `今天已签到 · 累计 ${props.totalDays} 天` : `今天还未签到 · 累计 ${props.totalDays} 天`
})
const action = computed(() => props.error || props.loading ? '查看' : props.signed ? '查看记录' : '去签到')
</script>

<template>
	<view class="mine-checkin" @tap="emit('open')">
		<view class="mine-checkin-icon"><text class="cuIcon-calendar"></text></view>
		<view class="mine-checkin-copy">
			<text class="mine-checkin-eyebrow">每日打卡</text>
			<text class="mine-checkin-title">每日签到</text>
			<text class="mine-checkin-subtitle">{{ description }}</text>
		</view>
		<view class="mine-checkin-action" :class="{ 'is-signed': signed }">{{ action }}<text class="cuIcon-right"></text></view>
	</view>
</template>

<style scoped>
.mine-checkin { display:flex; align-items:center; gap:20rpx; min-height:138rpx; padding:23rpx 26rpx; border:1rpx solid #e8ecf6; border-radius:30rpx; background:#fff; box-shadow:0 10rpx 30rpx rgba(41,55,101,.055); }
.mine-checkin-icon { flex:none; display:flex; align-items:center; justify-content:center; width:82rpx; height:82rpx; border-radius:25rpx; background:linear-gradient(145deg,#eef1ff,#e6e9ff); color:#596bf2; font-size:39rpx; }
.mine-checkin-copy { display:flex; flex:1; min-width:0; flex-direction:column; gap:3rpx; }
.mine-checkin-eyebrow { color:#909ab1; font-size:18rpx; letter-spacing:1rpx; }
.mine-checkin-title { color:#27314a; font-size:29rpx; font-weight:750; line-height:1.2; }
.mine-checkin-subtitle { overflow:hidden; white-space:nowrap; text-overflow:ellipsis; color:#8b95aa; font-size:20rpx; }
.mine-checkin-action { flex:none; display:flex; align-items:center; gap:4rpx; padding:13rpx 14rpx; border-radius:18rpx; background:#eef0ff; color:#5268eb; font-size:21rpx; font-weight:700; }
.mine-checkin-action.is-signed { background:#eaf7f0; color:#28a06d; }
.mine-checkin-action .cuIcon-right { font-size:15rpx; }
</style>
