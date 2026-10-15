<script setup>
import { computed, ref, watch } from 'vue'
import { resolveMediaDisplayUrl } from '@/utils/avatar.js'
import { isPlaceholderVideoPoster } from '@/utils/video-cover.js'

const props = defineProps({
	src: { type: String, default: '' },
	poster: { type: String, default: '' },
	width: { type: [Number, String], default: 0 },
	height: { type: [Number, String], default: 0 }
})
const emit = defineEmits(['play'])

const displayPoster = ref('')
const displayVideo = ref('')
const posterFailed = ref(false)
let resolveVersion = 0

const coverStyle = computed(() => {
	const naturalWidth = Number(props.width || 0)
	const naturalHeight = Number(props.height || 0)
	if (!naturalWidth || !naturalHeight) return { width: '400rpx', height: '225rpx' }
	const ratio = naturalWidth / naturalHeight
	if (ratio >= 1) {
		return { width: '400rpx', height: `${Math.max(180, Math.min(360, 400 / ratio))}rpx` }
	}
	return { width: `${Math.max(180, Math.min(400, 360 * ratio))}rpx`, height: '360rpx' }
})

async function resolveCover() {
	const currentVersion = ++resolveVersion
	displayPoster.value = ''
	displayVideo.value = ''
	if (!posterFailed.value && !isPlaceholderVideoPoster(props.poster)) {
		try {
			const resolved = await resolveMediaDisplayUrl(props.poster)
			if (currentVersion === resolveVersion) displayPoster.value = resolved
			return
		} catch (error) {
			// 封面请求失败时继续使用视频真实画面。
		}
	}
	if (!props.src) return
	try {
		const resolved = await resolveMediaDisplayUrl(props.src)
		if (currentVersion === resolveVersion) displayVideo.value = resolved
	} catch (error) {
		if (currentVersion === resolveVersion) displayVideo.value = props.src
	}
}

function handlePosterError() {
	posterFailed.value = true
	displayPoster.value = ''
	resolveCover()
}

watch(() => [props.src, props.poster], () => {
	posterFailed.value = false
	resolveCover()
}, { immediate: true })
</script>

<template>
	<view class="video-message-cover" :style="coverStyle" @tap.stop="emit('play')">
		<image
			v-if="displayPoster"
			:src="displayPoster"
			class="cover-media"
			mode="aspectFill"
			@error="handlePosterError"
		/>
		<video
			v-else-if="displayVideo"
			:src="displayVideo"
			class="cover-media cover-video"
			:controls="false"
			:autoplay="false"
			:muted="true"
			:loop="false"
			:initial-time="0.1"
			:show-center-play-btn="false"
			:enable-progress-gesture="false"
			object-fit="cover"
			playsinline
		/>
		<view v-else class="cover-loading"></view>
		<view class="cover-shadow">
			<view class="cuIcon-video cover-play f-28 c-white"></view>
		</view>
	</view>
</template>

<style scoped>
.video-message-cover {
	position: relative;
	overflow: hidden;
	max-width: 400rpx;
	max-height: 360rpx;
	border-radius: 10rpx;
	background: linear-gradient(145deg, #dfe4ef, #cbd3e2);
}

.cover-media,
.cover-loading {
	display: block;
	width: 100%;
	height: 100%;
}

.cover-video {
	pointer-events: none;
}

.cover-shadow {
	position: absolute;
	inset: 0;
	z-index: 2;
	background: rgba(20, 27, 42, 0.24);
}

.cover-play {
	position: absolute;
	top: 50%;
	left: 50%;
	padding: 0 4rpx 0 6rpx;
	transform: translate(-50%, -50%);
}
</style>
