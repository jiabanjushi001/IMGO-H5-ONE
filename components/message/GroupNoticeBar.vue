<script setup>
import { computed, shallowRef, watch } from 'vue'
import NoticeDetailSheet from './NoticeDetailSheet.vue'

const props = defineProps({
	notice: { type: String, default: '' },
	title: { type: String, default: '' },
	top: { type: Number, default: 0 }
})
const emit = defineEmits(['visibility-change'])
const dismissed = shallowRef(false)
const detailVisible = shallowRef(false)
const noticeParts = computed(() => {
	const raw = props.notice.replace(/\r\n?/g, '\n').trim()
	const lines = raw ? raw.split('\n') : []
	const title = props.title.trim() || lines[0]?.trim() || '群公告'
	const bodyLines = lines[0]?.trim() === title ? lines.slice(1) : lines
	return { title, body: bodyLines.join('\n').trim() }
})
const displayTitle = computed(() => noticeParts.value.title)
const detailContent = computed(() => noticeParts.value.body)
const noticeText = computed(() => {
	const content = props.notice.replace(/\s+/g, ' ').trim()
	const title = props.title.trim()
	return title && !content.startsWith(title) ? `${title}：${content}` : content
})
const visible = computed(() => !!props.notice.trim() && !dismissed.value)

watch([() => props.title, () => props.notice], () => { dismissed.value = false })
watch(visible, value => {
	emit('visibility-change', value)
	if (!value) detailVisible.value = false
}, { immediate: true })

function openDetail() {
	if (visible.value) detailVisible.value = true
}

function dismiss() {
	dismissed.value = true
}
</script>

<template>
	<view v-if="visible" class="system-notice-bar" :style="{ top: props.top + 'px' }">
		<uni-notice-bar :key="noticeText" show-icon scrollable show-close :speed="40" :text="noticeText" style="margin: 0" @click="openDetail" @close="dismiss" />
	</view>
	<NoticeDetailSheet :visible="detailVisible" kind="群公告" :title="displayTitle" :content="detailContent" @close="detailVisible = false" />
</template>

<style scoped>
.system-notice-bar {
	position: fixed;
	left: 0;
	right: 0;
	z-index: 1000;
}

/* uni-notice-bar normally reserves a full banner width before the text.
   At 40px/s that empty stretch lasts several seconds on every loop. */
.system-notice-bar :deep(.uni-noticebar__content-text--scrollable) {
	padding-left: 12px;
	animation-delay: 0s !important;
}
</style>
