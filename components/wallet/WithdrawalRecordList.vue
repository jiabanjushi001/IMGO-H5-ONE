<script setup>
const props = defineProps({ records: { type: Array, default: () => [] } })
const statusName = status => ['待处理', '已打款', '已拒绝'][Number(status)] || '未知'
const dateText = timestamp => {
	const seconds = Number(timestamp)
	return seconds > 0 ? new Date(seconds * 1000).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false }) : '—'
}
const amountText = cents => `¥${(Number(cents || 0) / 100).toFixed(2)}`
</script>

<template>
	<view class="record-list">
		<view v-for="record in props.records" :key="record.withdrawal_id" class="record-card">
			<view class="record-top"><text class="record-amount">{{ amountText(record.amount_cents) }}</text><text class="record-status" :class="`is-${record.status}`">{{ statusName(record.status) }}</text></view>
			<view class="record-line"><text>申请时间</text><text>{{ dateText(record.created_at) }}</text></view>
			<view class="record-line"><text>收款银行</text><text>{{ record.bank_name || '银行卡' }}（尾号 {{ record.account_last4 || '—' }}）</text></view>
			<view v-if="Number(record.processed_at) > 0" class="record-line"><text>处理时间</text><text>{{ dateText(record.processed_at) }}</text></view>
			<view v-if="record.remark" class="record-remark">备注：{{ record.remark }}</view>
		</view>
	</view>
</template>

<style scoped>
.record-list { display:flex; flex-direction:column; gap:20rpx; }
.record-card { padding:30rpx; border:1rpx solid #e7ebf5; border-radius:24rpx; background:#fff; box-shadow:0 8rpx 30rpx rgba(38,55,87,.04); }
.record-top { display:flex; align-items:center; justify-content:space-between; gap:20rpx; margin-bottom:22rpx; }
.record-amount { color:#26324a; font-size:38rpx; font-weight:700; font-variant-numeric:tabular-nums; }
.record-status { padding:7rpx 17rpx; border-radius:30rpx; background:#fff5dd; color:#a77316; font-size:22rpx; }
.record-status.is-1 { background:#e8f7ef; color:#23875b; }
.record-status.is-2 { background:#fff0f0; color:#c4575d; }
.record-line { display:flex; justify-content:space-between; gap:20rpx; padding:7rpx 0; color:#8792a5; font-size:23rpx; }
.record-line text:last-child { color:#4c5970; text-align:right; overflow-wrap:anywhere; }
.record-remark { margin-top:15rpx; padding:16rpx; border-radius:12rpx; background:#f7f8fc; color:#737f94; font-size:22rpx; line-height:1.6; }
</style>
