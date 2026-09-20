<script setup>
defineProps({ entries: { type: Array, default: () => [] } })

const labels = {
	credit: '后台入账', recharge: '充值本金', recharge_bonus: '充值赠送',
	withdraw: '提现申请', refund: '提现退回', paid: '确认打款'
}
const money = cents => `¥${(Math.abs(Number(cents || 0)) / 100).toFixed(2)}`
const signedMoney = cents => `${Number(cents) >= 0 ? '+' : '−'}${money(cents)}`
const mainDelta = entry => Number(entry.available_delta) || Number(entry.pending_delta) || 0
const dateText = timestamp => new Date(Number(timestamp) * 1000).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai', hour12: false })
</script>

<template>
	<view class="wallet-entry-list">
		<view v-for="entry in entries" :key="entry.entry_id" class="wallet-entry">
			<view class="wallet-entry-head">
				<view class="wallet-entry-name">{{ labels[entry.event] || '余额变动' }}</view>
				<text class="wallet-entry-amount" :class="{ 'is-negative': mainDelta(entry) < 0 }">{{ signedMoney(mainDelta(entry)) }}</text>
			</view>
			<view class="wallet-entry-line"><text>{{ dateText(entry.created_at) }}</text><text v-if="Number(entry.pending_delta) && Number(entry.available_delta)">处理中 {{ signedMoney(entry.pending_delta) }}</text></view>
			<view v-if="entry.note" class="wallet-entry-note">{{ entry.note }}</view>
		</view>
	</view>
</template>

<style scoped>
.wallet-entry-list { display:flex; flex-direction:column; gap:18rpx; }
.wallet-entry { padding:28rpx; border:1rpx solid #e7ebf5; border-radius:24rpx; background:#fff; box-shadow:0 8rpx 30rpx rgba(38,55,87,.04); }
.wallet-entry-head,.wallet-entry-line { display:flex; justify-content:space-between; align-items:center; gap:16rpx; }
.wallet-entry-name { color:#26324a; font-size:28rpx; font-weight:650; }
.wallet-entry-amount { color:#28845f; font-size:29rpx; font-weight:700; font-variant-numeric:tabular-nums; }
.wallet-entry-amount.is-negative { color:#c8656b; }
.wallet-entry-line { margin-top:12rpx; color:#8a96a8; font-size:21rpx; }
.wallet-entry-note { margin-top:15rpx; padding:13rpx 16rpx; border-radius:12rpx; background:#f7f8fc; color:#77849a; font-size:22rpx; line-height:1.5; overflow-wrap:anywhere; }
</style>
