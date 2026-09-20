<script setup>
import { shallowRef } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import walletApi from '@/api/wallet.js'
import WalletEntryList from '@/components/wallet/WalletEntryList.vue'

const entries = shallowRef([])
const total = shallowRef(0)
const page = shallowRef(0)
const loading = shallowRef(false)
const failed = shallowRef(false)

async function load(reset = false) {
	if (loading.value) return
	if (reset) { page.value = 0; entries.value = []; total.value = 0 }
	loading.value = true
	failed.value = false
	const nextPage = page.value + 1
	try {
		const res = await walletApi.entries({ page: nextPage, limit: 20 })
		if (res.code !== 0) throw new Error(res.msg || '读取余额明细失败')
		entries.value = [...entries.value, ...(Array.isArray(res.data) ? res.data : [])]
		total.value = Number(res.count) || 0
		page.value = nextPage
	} catch (cause) {
		failed.value = true
		uni.showToast({ title: cause.message || '加载失败，请重试', icon: 'none' })
	} finally {
		loading.value = false
	}
}

onShow(() => load(true))
</script>

<template>
	<view class="ledger-page">
		<cu-custom bgColor="bg-white" :isBack="true" :fallbackToHome="true"><template #content>余额明细</template></cu-custom>
		<view class="ledger-body">
			<view class="ledger-heading">余额明细 <text>共 {{ total }} 条变动</text></view>
			<WalletEntryList :entries="entries" />
			<view v-if="!loading && !failed && entries.length === 0" class="ledger-empty">暂无余额变动记录</view>
			<button v-if="failed || entries.length < total" class="ledger-more" :disabled="loading" @tap="load(false)">{{ loading ? '加载中…' : (failed ? '重试' : '加载更多') }}</button>
			<view v-else-if="loading" class="ledger-loading">正在读取明细…</view>
		</view>
	</view>
</template>

<style scoped>
.ledger-page { box-sizing:border-box; min-height:100vh; background:#f4f7fc; color:#25304a; }
.ledger-body { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; padding:30rpx 26rpx 90rpx; }
.ledger-heading { display:flex; align-items:baseline; justify-content:space-between; margin:12rpx 6rpx 30rpx; font-size:33rpx; font-weight:700; }
.ledger-heading text { color:#8b96a8; font-size:22rpx; font-weight:400; }
.ledger-empty { margin-top:24rpx; padding:95rpx 30rpx; border-radius:24rpx; background:#fff; color:#9aa4b5; text-align:center; font-size:25rpx; }
.ledger-more { margin-top:27rpx; border:1rpx solid #dce2f7; border-radius:16rpx; background:#fff; color:#5969d7; font-size:25rpx; }
.ledger-more::after { border:0; }
.ledger-loading { padding:30rpx; color:#8d98a9; text-align:center; font-size:24rpx; }
</style>
