<script setup>
import { shallowRef } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import walletApi from '@/api/wallet.js'
import WithdrawalRecordList from '@/components/wallet/WithdrawalRecordList.vue'

const records = shallowRef([])
const total = shallowRef(0)
const page = shallowRef(0)
const loading = shallowRef(false)
const failed = shallowRef(false)

async function load(reset = false) {
	if (loading.value) return
	if (reset) { page.value = 0; records.value = []; total.value = 0 }
	loading.value = true
	failed.value = false
	const nextPage = page.value + 1
	try {
		const res = await walletApi.history({ page: nextPage, limit: 20 })
		if (res.code !== 0) throw new Error(res.msg || '读取提现记录失败')
		records.value = [...records.value, ...(Array.isArray(res.data) ? res.data : [])]
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
	<view class="history-page">
		<cu-custom bgColor="bg-white" :isBack="true" :fallbackToHome="true"><template #content>提现记录</template></cu-custom>
		<view class="history-body">
			<view class="history-title">提现记录 <text>共 {{ total }} 笔申请</text></view>
			<WithdrawalRecordList :records="records" />
			<view v-if="!loading && !failed && records.length === 0" class="history-empty">暂无提现记录</view>
			<button v-if="failed || records.length < total" class="history-more" :disabled="loading" @tap="load(false)">{{ loading ? '加载中…' : (failed ? '重试' : '加载更多') }}</button>
			<view v-else-if="loading" class="history-loading">正在读取记录…</view>
		</view>
	</view>
</template>

<style scoped>
.history-page { box-sizing:border-box; min-height:100vh; background:#f4f7fc; color:#25304a; }
.history-body { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; padding:30rpx 26rpx 90rpx; }
.history-title { display:flex; align-items:baseline; justify-content:space-between; margin:12rpx 6rpx 30rpx; font-size:33rpx; font-weight:700; }
.history-title text { color:#8b96a8; font-size:22rpx; font-weight:400; }
.history-empty { margin-top:24rpx; padding:95rpx 30rpx; border-radius:24rpx; background:#fff; color:#9aa4b5; text-align:center; font-size:25rpx; }
.history-more { margin-top:27rpx; border:1rpx solid #dce2f7; border-radius:16rpx; background:#fff; color:#5969d7; font-size:25rpx; }
.history-more::after { border:0; }
.history-loading { padding:30rpx; color:#8d98a9; text-align:center; font-size:24rpx; }
</style>
