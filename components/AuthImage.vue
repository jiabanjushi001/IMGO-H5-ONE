<script setup>
/**
 * 带鉴权的图片组件：/storage、群头像等自动转 blob 后再显示。
 */
import { ref, watch } from 'vue'
import {
	normalizeMediaUrl,
	normalizeAvatarUrl,
	needsAuthMediaUrl,
	resolveMediaDisplayUrl,
	resolveAvatarDisplayUrl
} from '@/utils/avatar.js'

const props = defineProps({
	src: { type: String, default: '' },
	mode: { type: String, default: 'aspectFill' },
	/** 按头像规则解析（空值可回退默认字母头像） */
	avatar: { type: Boolean, default: false },
	info: { type: Object, default: null }
})

const displaySrc = ref('')

const refresh = (raw) => {
	const value = typeof raw === 'string' ? raw.trim() : ''
	if (!value) {
		displaySrc.value = props.avatar ? normalizeAvatarUrl('', props.info || {}) : ''
		return
	}
	const normalized = props.avatar
		? normalizeAvatarUrl(value, props.info || {})
		: normalizeMediaUrl(value)
	// 需鉴权时先清空/占位，避免裸链 401 闪一下
	displaySrc.value = needsAuthMediaUrl(normalized) ? '' : normalized
	const task = props.avatar
		? resolveAvatarDisplayUrl(value, props.info || {})
		: resolveMediaDisplayUrl(value)
	task.then((url) => {
		const current = props.avatar
			? normalizeAvatarUrl(props.src, props.info || {})
			: normalizeMediaUrl(props.src)
		if (current === normalized) {
			displaySrc.value = url || normalized
		}
	}).catch(() => {
		if (!needsAuthMediaUrl(normalized)) {
			displaySrc.value = normalized
		}
	})
}

watch(() => [props.src, props.avatar, props.info], () => refresh(props.src), {
	immediate: true,
	deep: true
})
</script>

<template>
	<image
		v-if="displaySrc"
		class="auth-image"
		:src="displaySrc"
		:mode="mode"
		v-bind="$attrs"
	/>
</template>

<style scoped>
.auth-image {
	display: block;
}
</style>
