<script setup>
import { computed } from 'vue'

const props = defineProps({
	availableCents: { type: Number, default: 0 },
	pendingCents: { type: Number, default: 0 },
	frozenCents: { type: Number, default: 0 },
	loading: { type: Boolean, default: false },
	error: { type: Boolean, default: false }
})
const emit = defineEmits(['refresh', 'withdraw', 'history', 'entries'])
const balanceText = computed(() => `¥${((props.availableCents + props.frozenCents) / 100).toFixed(2)}`)
const pendingText = computed(() => `¥${(props.pendingCents / 100).toFixed(2)}`)
const adminFrozenText = computed(() => `¥${(props.frozenCents / 100).toFixed(2)}`)
const hasPendingBalance = computed(() => props.pendingCents > 0)
const hasAdminFrozen = computed(() => props.frozenCents > 0)
</script>

<template>
	<view class="wallet-card">
		<view class="wallet-head">
			<view class="wallet-title-wrap"><view class="wallet-icon"><text class="cuIcon-pay"></text></view><text class="wallet-title">钱包余额</text></view>
			<view class="wallet-refresh" :class="{ 'is-disabled': loading }" @tap.stop="!loading && emit('refresh')"><text class="cuIcon-refresh"></text><text>刷新</text></view>
		</view>
		<view class="wallet-main">
			<view class="wallet-balance">
				<text class="wallet-caption">余额</text>
				<text v-if="error" class="wallet-error">获取失败，请刷新重试</text>
				<view v-else class="wallet-amount-row">
					<view class="wallet-total-row">
						<text class="wallet-amount">{{ loading ? '—' : balanceText }}</text>
						<text v-if="!loading && hasAdminFrozen" class="wallet-inline-frozen">（已冻结 {{ adminFrozenText }}）</text>
					</view>
					<view v-if="!loading && hasPendingBalance" class="wallet-status-list">
						<view v-if="hasPendingBalance" class="wallet-status-item is-pending">
							<text class="wallet-status-dot"></text><text>待处理金额：{{ pendingText }}</text>
						</view>
					</view>
				</view>
			</view>
			<button class="wallet-withdraw" :disabled="loading || error" @tap.stop="emit('withdraw')">提现<text class="cuIcon-right"></text></button>
		</view>
		<view class="wallet-divider"></view>
		<view class="wallet-links">
			<view class="wallet-link" @tap="emit('history')"><text class="cuIcon-form wallet-link-icon"></text><text>提现记录</text><text class="cuIcon-right wallet-link-arrow"></text></view>
			<view class="wallet-link" @tap="emit('entries')"><text class="cuIcon-list wallet-link-icon"></text><text>余额明细</text><text class="cuIcon-right wallet-link-arrow"></text></view>
		</view>
	</view>
</template>

<style scoped>
.wallet-card { box-sizing:border-box; padding:26rpx 30rpx 0; margin-bottom:24rpx; border:1rpx solid #e8ecf7; border-radius:32rpx; background:linear-gradient(145deg,#fff 45%,#f8f9ff 100%); box-shadow:0 14rpx 38rpx rgba(41,55,101,.07); }
.wallet-head,.wallet-title-wrap,.wallet-refresh,.wallet-main,.wallet-amount-row,.wallet-total-row,.wallet-links,.wallet-link,.wallet-status-list,.wallet-status-item { display:flex; align-items:center; }
.wallet-head,.wallet-main { justify-content:space-between; }
.wallet-title-wrap { gap:14rpx; }
.wallet-icon { display:flex; align-items:center; justify-content:center; width:53rpx; height:53rpx; border-radius:17rpx; background:#edf0ff; color:#576af0; font-size:29rpx; }
.wallet-title { color:#293450; font-size:27rpx; font-weight:700; }
.wallet-refresh { gap:6rpx; padding:10rpx 14rpx; border-radius:24rpx; background:#f0f2ff; color:#6573d6; font-size:20rpx; }
.wallet-refresh.is-disabled { opacity:.5; }
.wallet-refresh .cuIcon-refresh { font-size:22rpx; }
.wallet-main { gap:24rpx; margin-top:25rpx; }
.wallet-balance { display:flex; flex-direction:column; min-width:0; }
.wallet-caption { color:#8e98ad; font-size:21rpx; }
.wallet-amount-row { flex-direction:column; align-items:flex-start; min-width:0; }
.wallet-total-row { flex-wrap:wrap; gap:0 8rpx; min-width:0; }
.wallet-amount { margin-top:3rpx; color:#1e2943; font-size:64rpx; line-height:1.2; font-weight:750; font-variant-numeric:tabular-nums; letter-spacing:-1rpx; }
.wallet-inline-frozen { margin-top:9rpx; color:#e0525a; font-size:23rpx; line-height:1.3; font-weight:650; white-space:nowrap; }
.wallet-status-list { flex-wrap:wrap; gap:8rpx 18rpx; margin-top:9rpx; }
.wallet-status-item { gap:8rpx; font-size:21rpx; line-height:1.35; font-weight:500; white-space:nowrap; }
.wallet-status-dot { width:12rpx; height:12rpx; border-radius:50%; }
.wallet-status-item.is-pending { color:#9a741b; }
.wallet-status-item.is-pending .wallet-status-dot { background:#f2b134; box-shadow:0 0 0 5rpx rgba(242,177,52,.13); }
.wallet-error { margin-top:12rpx; color:#c25860; font-size:23rpx; }
.wallet-withdraw { flex:none; display:flex; align-items:center; justify-content:center; gap:7rpx; min-width:116rpx; height:64rpx; line-height:64rpx; margin:0; padding:0 20rpx; border:0; border-radius:19rpx; background:#526dff; box-shadow:0 9rpx 20rpx rgba(82,109,255,.2); color:#fff; font-size:24rpx; font-weight:650; }
.wallet-withdraw::after { border:0; }
.wallet-withdraw[disabled] { opacity:.5; }
.wallet-withdraw .cuIcon-right { font-size:16rpx; }
.wallet-divider { height:1rpx; margin-top:24rpx; background:#e9edf7; }
.wallet-links { justify-content:space-between; }
.wallet-link { box-sizing:border-box; gap:10rpx; width:50%; min-height:78rpx; color:#46536d; font-size:22rpx; }
.wallet-link:first-child { padding-right:18rpx; border-right:1rpx solid #e9edf7; }
.wallet-link:last-child { padding-left:18rpx; }
.wallet-link-icon { color:#6579e8; font-size:26rpx; }
.wallet-link-arrow { margin-left:auto; color:#a5aec0; font-size:16rpx; }
</style>
