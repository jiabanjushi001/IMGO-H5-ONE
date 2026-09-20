<script setup>
import { shallowRef } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import checkInApi from '@/api/check-in.js'
import CheckInHero from '@/components/check-in/CheckInHero.vue'
import CheckInMilestones from '@/components/check-in/CheckInMilestones.vue'
import CheckInRules from '@/components/check-in/CheckInRules.vue'

const loading = shallowRef(true)
const submitting = shallowRef(false)
const error = shallowRef(false)
const signedToday = shallowRef(false)
const totalDays = shallowRef(0)
const date = shallowRef('')

function updateStatus(data) {
	signedToday.value = !!data.signed_today
	totalDays.value = Number(data.total_days) || 0
	date.value = data.date || ''
}

async function load() {
	if (submitting.value) return
	loading.value = true
	error.value = false
	try {
		const res = await checkInApi.status()
		if (res.code !== 0) throw new Error(res.msg || '获取签到状态失败')
		updateStatus(res.data || {})
	} catch (cause) {
		error.value = true
	} finally {
		loading.value = false
	}
}

async function submit() {
	if (loading.value || submitting.value || signedToday.value || error.value) return
	submitting.value = true
	try {
		const res = await checkInApi.submit()
		if (res.code !== 0) throw new Error(res.msg || '签到失败')
		const data = res.data || {}
		updateStatus(data)
		uni.showToast({ title: data.already_signed ? '今天已经签到过了' : '签到成功', icon: 'none' })
	} catch (cause) {
		uni.showToast({ title: cause.message || '签到失败，请重试', icon: 'none' })
	} finally {
		submitting.value = false
	}
}

onShow(load)
</script>

<template>
	<view class="checkin-page">
		<cu-custom bgColor="bg-white" :isBack="true" :fallbackToHome="true">
			<template #backText></template>
			<template #content>每日签到</template>
		</cu-custom>
		<view class="checkin-content">
			<CheckInHero :date="date" :signed-today="signedToday" :total-days="totalDays" :loading="loading" :error="error" />
			<CheckInMilestones
				:total-days="totalDays"
				:signed-today="signedToday"
				:loading="loading"
				:submitting="submitting"
				:error="error"
				@sign-in="submit"
				@retry="load"
			/>
			<CheckInRules />
		</view>
	</view>
</template>

<style scoped>
.checkin-page {
	box-sizing: border-box;
	min-height: 100vh;
	background: radial-gradient(circle at 96% 18%, rgba(124, 92, 255, .12), transparent 38%), #f4f7fc;
	color: #182033;
}
.checkin-content {
	box-sizing: border-box;
	width: 100%;
	max-width: 900rpx;
	margin: 0 auto;
	padding: 28rpx 24rpx 90rpx;
}
</style>
