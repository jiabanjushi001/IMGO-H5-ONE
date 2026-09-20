<script setup>
import { computed, shallowRef } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import inviteApi from '@/api/invite.js'
import InviteHero from '@/components/invite/InviteHero.vue'
import InviteMetrics from '@/components/invite/InviteMetrics.vue'
import InviteCodeCard from '@/components/invite/InviteCodeCard.vue'
import InviteShareActions from '@/components/invite/InviteShareActions.vue'

const loading = shallowRef(true)
const error = shallowRef(false)
const inviteCode = shallowRef('')
const directCount = shallowRef(0)
const streakDays = shallowRef(0)
const serverURL = shallowRef('')

const shareURL = computed(() => {
	if (!/^[0-9]{6}$/.test(inviteCode.value)) return ''
	// #ifdef H5
	if (typeof window !== 'undefined') {
		return `${window.location.origin}${window.location.pathname}#/pages/login/register?inviteCode=${inviteCode.value}`
	}
	// #endif
	return serverURL.value
})

async function load() {
	loading.value = true
	error.value = false
	try {
		const res = await inviteApi.status()
		if (res.code !== 0) throw new Error(res.msg || '获取邀请信息失败')
		const data = res.data || {}
		if (!/^[0-9]{6}$/.test(data.invite_code || '')) throw new Error('邀请码格式无效')
		inviteCode.value = data.invite_code
		directCount.value = Number(data.direct_count) || 0
		streakDays.value = Number(data.streak_days) || 0
		serverURL.value = data.invite_url || ''
	} catch (cause) {
		error.value = true
	} finally {
		loading.value = false
	}
}

function copy(value, label) {
	if (!value) {
		uni.showToast({ title: '请先获取邀请码', icon: 'none' })
		return
	}
	uni.setClipboardData({
		data: value,
		success: () => uni.showToast({ title: `${label}已复制`, icon: 'none' }),
		fail: () => uni.showToast({ title: '复制失败，请重试', icon: 'none' })
	})
}

onShow(load)
</script>

<template>
	<view class="invite-page">
		<cu-custom bgColor="bg-white" :isBack="true" :fallbackToHome="true">
			<template #backText></template>
			<template #content>邀请好友</template>
		</cu-custom>
		<view class="invite-content">
			<InviteHero />
			<InviteMetrics :direct-count="directCount" :streak-days="streakDays" :loading="loading" :error="error" />
			<view class="invite-code-wrap"><InviteCodeCard :code="inviteCode" :loading="loading" :error="error" @copy-code="copy(inviteCode, '邀请码')" @retry="load" /></view>
			<InviteShareActions :disabled="loading || error" @copy-link="copy(shareURL, '邀请链接')" />
		</view>
	</view>
</template>

<style scoped>
.invite-page { box-sizing:border-box; min-height:100vh; background:radial-gradient(circle at 93% 13%,rgba(118,109,236,.12),transparent 39%),#f4f7fc; color:#182033; }
.invite-content { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; padding:24rpx 24rpx 95rpx; }
.invite-code-wrap { margin:24rpx 0 32rpx; }
</style>
