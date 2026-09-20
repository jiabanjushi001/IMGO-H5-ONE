<script setup>
import { computed, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue'
import msgApi from '@/api/message.js'
import { useloginStore } from '@/store/login'
import pinia from '@/store/index'
import NoticeDetailSheet from './NoticeDetailSheet.vue'

const props = defineProps({
	top: { type: Number, default: 0 }
})
const emit = defineEmits(['visibility-change'])
const loginStore = useloginStore(pinia)
const notice = ref(null)
const noticeKey = shallowRef('')
const dismissed = shallowRef(false)
const detailVisible = shallowRef(false)

const noticeText = computed(() => {
	if (!notice.value) return ''
	return String(notice.value.notice || '').replace(/\s+/g, ' ').trim()
})
const visible = computed(() =>
	String(loginStore.globalConfig.sysInfo?.noticePopup) === '1' &&
	!dismissed.value &&
	!!noticeText.value
)

watch(visible, value => {
	emit('visibility-change', value)
	if (!value) detailVisible.value = false
}, { immediate: true })
watch(() => loginStore.globalConfig.sysInfo?.noticePopup, value => {
	if (String(value) === '1') loadNotice()
})

async function loadNotice() {
	if (!uni.getStorageSync('authToken')) return
	try {
		const config = await loginStore.getGlobalConfig()
		if (!config || String(config.sysInfo?.noticePopup) !== '1') return
		const res = await msgApi.getAdminNotice()
		const latest = res.code === 0 ? res.data : null
		if (!latest || !String(latest.notice || '').trim()) {
			notice.value = null
			return
		}
		const key = `${latest.create_time}:${latest.title}:${latest.notice}`
		if (key !== noticeKey.value) dismissed.value = false
		noticeKey.value = key
		notice.value = latest
	} catch (error) {
		console.warn('加载系统公告失败', error)
	}
}

function dismiss() {
	dismissed.value = true
}

function openDetail() {
	if (visible.value) detailVisible.value = true
}

onMounted(() => {
	uni.$on('adminNoticePublished', loadNotice)
	uni.$on('systemNoticeRefresh', loadNotice)
	loadNotice()
})
onUnmounted(() => {
	uni.$off('adminNoticePublished', loadNotice)
	uni.$off('systemNoticeRefresh', loadNotice)
})
</script>

<template>
	<view v-if="visible" class="system-notice-bar" :style="{ top: props.top + 'px' }">
		<uni-notice-bar :key="noticeKey" show-icon scrollable show-close :speed="40" :text="noticeText" style="margin: 0" @click="openDetail" @close="dismiss" />
	</view>
	<NoticeDetailSheet :visible="detailVisible" kind="系统公告" :title="String(notice?.title || '')" :content="String(notice?.notice || '')" @close="detailVisible = false" />
</template>

<style scoped>
.system-notice-bar {
	position: fixed;
	left: 0;
	right: 0;
	z-index: 1000;
}

.system-notice-bar :deep(.uni-noticebar__content-text--scrollable) {
	padding-left: 12px;
	animation-delay: 0s !important;
}
</style>
