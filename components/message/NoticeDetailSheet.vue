<script setup>
import { computed } from 'vue'

const props = defineProps({
	visible: { type: Boolean, default: false },
	kind: { type: String, default: '系统公告' },
	title: { type: String, default: '' },
	content: { type: String, default: '' }
})
const emit = defineEmits(['close'])
const contentHeight = computed(() => {
	const lines = props.content.split(/\r?\n/).reduce((total, line) => total + Math.max(1, Math.ceil([...line].length / 22)), 0)
	return Math.min(560, Math.max(96, lines * 48 + 12))
})
</script>

<template>
	<view v-if="props.visible" class="cu-modal notice-detail-modal show" @tap.self="emit('close')">
		<view class="cu-dialog notice-detail-card">
			<view class="notice-detail-header">
				<text class="notice-detail-kind"><text class="cuIcon-notification"></text>{{ props.kind }}</text>
				<view class="notice-detail-close cuIcon-close" role="button" aria-label="关闭公告" @tap="emit('close')"></view>
			</view>
			<text v-if="props.title.trim()" class="notice-detail-title" :class="{ 'centered-title': props.kind === '系统公告' || props.kind === '群公告' }">{{ props.title.trim() }}</text>
			<scroll-view v-if="props.content.trim()" scroll-y class="notice-detail-scroll" :style="{ height: contentHeight + 'rpx' }">
				<text class="notice-detail-content">{{ props.content }}</text>
			</scroll-view>
			<view class="notice-detail-footer" @tap="emit('close')">我知道了</view>
		</view>
	</view>
</template>

<style scoped>
.notice-detail-modal { z-index:1200; background:rgba(20,29,49,.55); }
.notice-detail-card { box-sizing:border-box; width:calc(100% - 64rpx); max-width:620px; padding:31rpx 34rpx 28rpx; border-radius:30rpx; background:#fff; text-align:left; box-shadow:0 20rpx 70rpx rgba(17,28,59,.2); }
.notice-detail-header { display:flex; align-items:center; justify-content:space-between; gap:18rpx; }
.notice-detail-kind { display:flex; align-items:center; gap:12rpx; color:#5269e9; font-size:25rpx; font-weight:650; }
.notice-detail-kind .cuIcon-notification { font-size:29rpx; }
.notice-detail-close { display:flex; align-items:center; justify-content:center; width:54rpx; height:54rpx; border-radius:16rpx; background:#f3f5fa; color:#8792aa; font-size:24rpx; }
.notice-detail-title { display:block; margin-top:28rpx; color:#1f2b43; font-size:33rpx; font-weight:750; line-height:1.4; overflow-wrap:anywhere; }
.notice-detail-title.centered-title { padding:0 10rpx 22rpx; border-bottom:1rpx solid #e9edf7; color:#1a2948; font-size:36rpx; font-weight:800; line-height:1.45; letter-spacing:.03em; text-align:center; }
.centered-title + .notice-detail-scroll { margin-top:22rpx; }
.notice-detail-scroll { box-sizing:border-box; max-height:48vh; margin-top:26rpx; }
.notice-detail-content { display:block; padding-right:6rpx; color:#42506a; font-size:27rpx; line-height:1.75; white-space:pre-wrap; overflow-wrap:anywhere; }
.notice-detail-footer { display:flex; align-items:center; justify-content:center; height:78rpx; margin-top:33rpx; border-radius:19rpx; background:linear-gradient(125deg,#5069ef,#745fec); color:#fff; font-size:27rpx; font-weight:650; }
</style>
