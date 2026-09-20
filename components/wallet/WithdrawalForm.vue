<script setup>
import { computed } from 'vue'

const props = defineProps({
	amount: { type: String, default: '' },
	availableCents: { type: Number, default: 0 },
	card: { type: Object, default: () => ({}) },
	loading: { type: Boolean, default: false },
	saving: { type: Boolean, default: false },
	canSubmit: { type: Boolean, default: false },
	hint: { type: String, default: '' }
})
const emit = defineEmits(['update:amount', 'submit', 'open-bank'])
const availableText = computed(() => `¥${(props.availableCents / 100).toFixed(2)}`)
const bankReady = computed(() => !!props.card.bound && Number(props.card.status) === 1)
const bankTitle = computed(() => bankReady.value ? `${props.card.bank_name || '银行卡'} ${props.card.receipt_account_masked || ''}` : '请先绑定并通过审核的银行卡')
</script>

<template>
	<view class="withdraw-form">
		<view class="withdraw-balance"><text>可提现余额</text><text class="withdraw-balance-amount">{{ loading ? '—' : availableText }}</text></view>
		<view class="withdraw-field">
			<text class="withdraw-label">提现金额</text>
			<view class="withdraw-input-row"><text>¥</text><input :value="amount" type="digit" maxlength="12" placeholder="请输入金额" @input="emit('update:amount', $event.detail.value)" /></view>
			<text class="withdraw-hint">{{ hint || '最低提现 ¥0.01，金额不能超过可提现余额' }}</text>
		</view>
		<view class="withdraw-bank" @tap="emit('open-bank')">
			<view><text class="cuIcon-card"></text><text>{{ bankTitle }}</text></view><text class="cuIcon-right"></text>
		</view>
		<button class="withdraw-submit" :disabled="!canSubmit || saving || loading" @tap="emit('submit')">{{ saving ? '提交中…' : '提交提现申请' }}</button>
		<view class="withdraw-note">提交后金额会暂时冻结；审核拒绝会退回钱包。审核通过后由管理员处理打款。</view>
	</view>
</template>

<style scoped>
.withdraw-form { padding:34rpx; border:1rpx solid #e8edf6; border-radius:28rpx; background:#fff; box-shadow:0 14rpx 42rpx rgba(39,54,91,.05); }
.withdraw-balance { display:flex; justify-content:space-between; align-items:center; color:#738096; font-size:24rpx; }
.withdraw-balance-amount { color:#25304a; font-size:34rpx; font-weight:700; }
.withdraw-field { padding-top:44rpx; }
.withdraw-label { color:#2a344b; font-size:29rpx; font-weight:650; }
.withdraw-input-row { display:flex; align-items:center; gap:12rpx; padding:24rpx 0 20rpx; border-bottom:2rpx solid #e6eaf4; color:#252d45; font-size:56rpx; font-weight:700; }
.withdraw-input-row input { flex:1; min-width:0; height:72rpx; color:#252d45; font-size:53rpx; font-weight:700; }
.withdraw-hint { display:block; margin-top:16rpx; min-height:34rpx; color:#8792a5; font-size:22rpx; }
.withdraw-bank { display:flex; justify-content:space-between; align-items:center; gap:16rpx; margin-top:40rpx; padding:25rpx 0; border-top:1rpx solid #edf0f5; border-bottom:1rpx solid #edf0f5; color:#3f4a61; font-size:24rpx; }
.withdraw-bank > view { display:flex; align-items:center; gap:16rpx; min-width:0; }
.withdraw-bank > view text:last-child { overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.withdraw-bank .cuIcon-card { color:#5b6ef0; font-size:34rpx; }
.withdraw-bank .cuIcon-right { color:#a5aec0; }
.withdraw-submit { margin-top:48rpx; height:90rpx; line-height:90rpx; border:0; border-radius:18rpx; background:#566fff; color:#fff; font-size:29rpx; font-weight:700; }
.withdraw-submit::after { border:0; }
.withdraw-submit[disabled] { opacity:.48; }
.withdraw-note { margin-top:24rpx; color:#8994a6; font-size:22rpx; line-height:1.7; }
</style>
