<script setup>
import { computed, shallowRef } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import bankApi from '@/api/bank.js'
import walletApi from '@/api/wallet.js'
import WithdrawalForm from '@/components/wallet/WithdrawalForm.vue'

const loading = shallowRef(true)
const saving = shallowRef(false)
const error = shallowRef(false)
const availableCents = shallowRef(0)
const card = shallowRef({})
const amount = shallowRef('')
const requestId = shallowRef(newRequestId())

function newRequestId() { return `wd-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 14)}` }
function parseAmount(value) {
	const match = /^(\d{1,9})(?:\.(\d{1,2}))?$/.exec(value.trim())
	return match ? Number(match[1]) * 100 + Number((match[2] || '').padEnd(2, '0')) : 0
}
const amountCents = computed(() => parseAmount(amount.value))
const bankReady = computed(() => !!card.value.bound && Number(card.value.status) === 1)
const canSubmit = computed(() => !error.value && bankReady.value && amountCents.value > 0 && amountCents.value <= availableCents.value)
const hint = computed(() => {
	if (error.value) return '信息获取失败，请重试'
	if (!bankReady.value) return '请先绑定银行卡并等待审核通过'
	if (amount.value && !amountCents.value) return '请输入正确的提现金额，最多两位小数'
	if (amountCents.value > availableCents.value) return '提现金额超过可用余额'
	return ''
})

async function load() {
	loading.value = true
	error.value = false
	try {
		const [walletRes, bankRes] = await Promise.all([walletApi.status(), bankApi.get()])
		if (walletRes.code !== 0 || bankRes.code !== 0) throw new Error(walletRes.msg || bankRes.msg || '获取钱包信息失败')
		availableCents.value = Number(walletRes.data.available_cents) || 0
		card.value = bankRes.data || {}
	} catch (cause) {
		error.value = true
		uni.showToast({ title: cause.message || '加载失败', icon: 'none' })
	} finally {
		loading.value = false
	}
}

function openBank() { uni.navigateTo({ url: '/pages/mine/bank-card' }) }
function openHistory() { uni.navigateTo({ url: '/pages/mine/wallet-history' }) }
async function submit() {
	if (!canSubmit.value || saving.value || loading.value) return
	const confirm = await new Promise(resolve => uni.showModal({ title: '确认提现', content: `申请提现 ¥${(amountCents.value / 100).toFixed(2)} 到已绑定银行卡？`, success: res => resolve(!!res.confirm), fail: () => resolve(false) }))
	if (!confirm) return
	saving.value = true
	try {
		const res = await walletApi.withdraw({ amount: amount.value.trim(), request_id: requestId.value })
		if (res.code !== 0) throw new Error(res.msg || '提交提现申请失败')
		requestId.value = newRequestId()
		amount.value = ''
		uni.showToast({ title: '提现申请已提交', icon: 'none' })
		await load()
		openHistory()
	} catch (cause) {
		uni.showToast({ title: cause.message || '提交失败，请重试', icon: 'none' })
	} finally {
		saving.value = false
	}
}

onShow(load)
</script>

<template>
	<view class="withdraw-page">
		<cu-custom bgColor="bg-white" :isBack="true" :fallbackToHome="true"><template #content>提现</template></cu-custom>
		<view class="withdraw-body">
			<view class="withdraw-heading"><view class="withdraw-symbol">¥</view><view><view>钱包提现</view><text>安全提交，记录全程可查</text></view></view>
			<WithdrawalForm v-model:amount="amount" :available-cents="availableCents" :card="card" :loading="loading" :saving="saving" :can-submit="canSubmit" :hint="hint" @submit="submit" @open-bank="openBank" />
			<view v-if="error" class="withdraw-retry" @tap="load">重新加载钱包信息</view>
			<view class="withdraw-record" @tap="openHistory">查看提现记录 <text class="cuIcon-right"></text></view>
		</view>
	</view>
</template>

<style scoped>
.withdraw-page { box-sizing:border-box; min-height:100vh; background:radial-gradient(circle at 100% 0%,rgba(112,121,250,.13),transparent 43%),#f4f7fc; color:#25304a; }
.withdraw-body { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; padding:32rpx 26rpx 80rpx; }
.withdraw-heading { display:flex; align-items:center; gap:20rpx; margin:14rpx 8rpx 30rpx; font-size:34rpx; font-weight:700; }
.withdraw-heading text { display:block; margin-top:6rpx; color:#8490a5; font-size:22rpx; font-weight:400; }
.withdraw-symbol { display:flex; align-items:center; justify-content:center; width:74rpx; height:74rpx; border-radius:22rpx; background:#566fff; color:#fff; font-size:40rpx; }
.withdraw-retry { margin:24rpx auto; color:#5b6ee6; text-align:center; font-size:24rpx; }
.withdraw-record { display:flex; justify-content:center; align-items:center; gap:8rpx; padding:32rpx; color:#5969d7; font-size:24rpx; }
</style>
