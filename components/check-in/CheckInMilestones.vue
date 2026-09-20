<script setup>
import { computed } from 'vue'

const props = defineProps({
	totalDays: { type: Number, default: 0 },
	signedToday: { type: Boolean, default: false },
	loading: { type: Boolean, default: false },
	submitting: { type: Boolean, default: false },
	error: { type: Boolean, default: false }
})
const emit = defineEmits(['sign-in', 'retry'])

const milestones = computed(() => Array.from({ length: 7 }, (_, index) => ({
	day: index + 1,
	completed: props.totalDays >= index + 1,
	next: props.totalDays === index
})))
const buttonLabel = computed(() => {
	if (props.loading) return '正在查询…'
	if (props.error) return '重新加载'
	if (props.submitting) return '正在签到…'
	return props.signedToday ? '今日已签到' : '立即签到'
})
const helperText = computed(() => {
	if (props.error) return '状态暂时获取失败，点击上方按钮重试'
	if (props.loading) return '正在确认今日签到状态'
	return props.signedToday ? '今天的签到已完成，明天再来看看吧' : '每天来一次，记录你的每一份坚持'
})

function handleAction() {
	if (props.loading || props.submitting || (props.signedToday && !props.error)) return
	emit(props.error ? 'retry' : 'sign-in')
}
</script>

<template>
	<view class="milestone-card">
		<view class="section-heading">
			<view class="section-heading-mark"></view>
			<view class="section-heading-copy">
				<view class="section-title">7 天累计里程碑</view>
				<view class="section-subtitle">累计天数点亮进度，不计算连续天数</view>
			</view>
		</view>
		<view class="milestone-track">
			<view v-for="step in milestones" :key="step.day" class="milestone" :class="{ 'is-complete': step.completed && !error, 'is-next': step.next && !loading && !error }">
				<text class="milestone-index">{{ step.day }}</text>
				<view class="milestone-symbol">{{ step.completed && !error ? '✓' : step.day }}</view>
				<text class="milestone-caption">{{ step.completed && !error ? '达成' : step.next && !loading && !error ? '下一站' : '待解锁' }}</text>
			</view>
		</view>
		<view class="milestone-progress">
			<view class="milestone-progress-fill" :style="{ width: (error ? 0 : Math.min(totalDays, 7) / 7 * 100) + '%' }"></view>
		</view>
		<view class="milestone-count">{{ loading ? '正在读取进度' : error ? '进度待确认' : totalDays >= 7 ? '已点亮全部基础里程碑' : `已点亮 ${totalDays} / 7 天` }}</view>
		<button class="sign-button" :class="{ 'is-done': signedToday && !error }" :disabled="loading || submitting || (signedToday && !error)" @tap="handleAction">
			<text class="sign-button-icon">{{ signedToday && !error ? '✓' : '✦' }}</text>{{ buttonLabel }}
		</button>
		<view class="sign-helper" :class="{ 'is-error': error }">{{ helperText }}</view>
	</view>
</template>

<style scoped>
.milestone-card { box-sizing: border-box; margin-top: 26rpx; padding: 34rpx 26rpx 32rpx; border: 1rpx solid #e7ebf4; border-radius: 30rpx; background: #fff; box-shadow: 0 12rpx 36rpx rgba(44,57,98,.07); }
.section-heading { display: flex; align-items: flex-start; gap: 15rpx; }
.section-heading-mark { flex: 0 0 8rpx; width: 8rpx; height: 37rpx; margin-top: 3rpx; border-radius: 7rpx; background: linear-gradient(#526dff,#8968ee); }
.section-heading-copy { min-width: 0; }
.section-title { font-size: 31rpx; font-weight: 700; line-height: 1.25; color: #252d45; }
.section-subtitle { margin-top: 7rpx; color: #8c95a8; font-size: 20rpx; line-height: 1.5; }
.milestone-track { display: flex; justify-content: space-between; gap: 8rpx; margin-top: 36rpx; }
.milestone { display: flex; align-items: center; flex: 1; flex-direction: column; min-width: 0; box-sizing: border-box; padding: 18rpx 0 15rpx; border: 1rpx solid transparent; border-radius: 20rpx; background: #f6f8fd; color: #a7afbf; }
.milestone.is-complete { border-color: #e4e7ff; background: #f0f2ff; color: #556bf0; }
.milestone.is-next { border-color: #b6c2ff; background: #f8f9ff; box-shadow: 0 9rpx 23rpx rgba(82,109,255,.1); }
.milestone-index { font-size: 21rpx; font-weight: 700; line-height: 1.2; }
.milestone-symbol { display: flex; align-items: center; justify-content: center; box-sizing: border-box; width: 54rpx; height: 54rpx; margin-top: 13rpx; border-radius: 50%; background: #e9edf6; color: #a3adbe; font-size: 24rpx; font-weight: 700; }
.milestone.is-complete .milestone-symbol { background: linear-gradient(135deg,#566fff,#8367ed); color: #fff; box-shadow: 0 7rpx 13rpx rgba(82,109,255,.23); }
.milestone.is-next .milestone-symbol { border: 2rpx solid #7487fa; background: #fff; color: #526dff; }
.milestone-caption { overflow: hidden; max-width: 100%; margin-top: 12rpx; font-size: 21rpx; line-height: 1.2; white-space: nowrap; }
.milestone-progress { overflow: hidden; height: 10rpx; margin: 30rpx 3rpx 0; border-radius: 10rpx; background: #eef1f8; }
.milestone-progress-fill { height: 100%; border-radius: 10rpx; background: linear-gradient(90deg,#566fff,#8969ed); transition: width .35s ease; }
.milestone-count { margin-top: 12rpx; color: #8a94a8; font-size: 21rpx; text-align: right; }
.sign-button { display: flex; align-items: center; justify-content: center; gap: 12rpx; box-sizing: border-box; width: 100%; height: 94rpx; margin-top: 36rpx; border: 0; border-radius: 23rpx; background: linear-gradient(110deg,#526dff,#7b61e8); box-shadow: 0 14rpx 29rpx rgba(82,109,255,.2); color: #fff; font-size: 29rpx; font-weight: 700; }
.sign-button::after { border: 0; }
.sign-button.is-done { background: #edf8f2; box-shadow: none; color: #279b70; }
.sign-button[disabled]:not(.is-done) { opacity: .66; }
.sign-button-icon { font-size: 28rpx; }
.sign-helper { margin-top: 19rpx; color: #9aa3b3; font-size: 22rpx; line-height: 1.5; text-align: center; }
.sign-helper.is-error { color: #db6375; }
</style>
