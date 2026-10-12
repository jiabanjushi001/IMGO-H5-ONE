<script setup>
import { ref, watch } from 'vue'
import { getCallVideos } from '@/api/call-video.js'
import msgApi from '@/api/message.js'

const props = defineProps({ visible: { type: Boolean, default: false } })
const emit = defineEmits(['select'])

const loading = ref(false)
const uploading = ref(false)
const videos = ref([])

watch(() => props.visible, visible => {
	if (visible) loadVideos()
}, { immediate: true })

async function loadVideos() {
	if (loading.value) return
	loading.value = true
	try {
		const response = await getCallVideos()
		videos.value = Number(response && response.code) === 0 && Array.isArray(response.data) ? response.data : []
	} finally {
		loading.value = false
	}
}

function videoExtension(result) {
	const name = String(result.name || result.tempFilePath || '')
	const match = name.match(/\.([a-z0-9]+)(?:$|[?#])/i)
	return match ? match[1].toLowerCase() : 'mp4'
}

function chooseVideo() {
	if (uploading.value) return
	uni.chooseVideo({
		sourceType: ['album'],
		compressed: false,
		success: uploadVideo
	})
}

function uploadVideo(result) {
	if (!result.tempFilePath) return
	uploading.value = true
	uni.showLoading({ title: '上传中…', mask: true })
	uni.uploadFile({
		url: msgApi.uploadUrl,
		filePath: result.tempFilePath,
		name: 'file',
		header: { Authorization: uni.getStorageSync('authToken') },
		formData: { ext: videoExtension(result) },
		success: async response => {
			let body
			try { body = typeof response.data === 'string' ? JSON.parse(response.data) : response.data }
			catch (_) { uni.showToast({ title: '上传返回异常', icon: 'none' }); return }
			if (Number(body && body.code) !== 0 || !Number(body && body.data && body.data.file_id)) {
				uni.showToast({ title: body && body.msg || '视频上传失败', icon: 'none' })
				return
			}
			await loadVideos()
			uni.showToast({ title: '预录视频上传成功', icon: 'none' })
		},
		fail: () => uni.showToast({ title: '视频上传失败，请检查网络', icon: 'none' }),
		complete: () => { uploading.value = false; uni.hideLoading() }
	})
}

function formatSize(value) {
	const size = Number(value || 0)
	if (size < 1024 * 1024) return `${Math.max(1, Math.round(size / 1024))} KB`
	return `${(size / 1024 / 1024).toFixed(1)} MB`
}
</script>

<template>
	<view class="library">
		<view class="library-head">
			<view>
				<text class="library-title">选择预录视频</text>
				<text class="library-subtitle">对方通话页会显示“预录视频”</text>
			</view>
			<button class="upload-button" :loading="uploading" @tap.stop="chooseVideo">{{ uploading ? '上传中' : '上传视频' }}</button>
		</view>
		<view v-if="loading" class="empty">正在加载…</view>
		<view v-else-if="!videos.length" class="empty">
			<text class="cuIcon-video empty-icon"></text>
			<text>还没有预录视频，请先上传</text>
		</view>
		<scroll-view v-else scroll-y class="video-list">
			<view v-for="video in videos" :key="video.file_id" class="video-item" @tap.stop="emit('select', video)">
				<view class="video-icon"><text class="cuIcon-video"></text></view>
				<view class="video-info">
					<text class="video-name">{{ video.name || '未命名视频' }}</text>
					<text class="video-size">{{ formatSize(video.size) }}</text>
				</view>
				<text class="select-text">选择</text>
			</view>
		</scroll-view>
	</view>
</template>

<style scoped>
.library { margin: 0 24rpx 24rpx; overflow: hidden; border: 1rpx solid #e7eafb; border-radius: 22rpx; background: #f8f9ff; }
.library-head { display:flex; align-items:center; justify-content:space-between; gap:18rpx; padding:24rpx; }
.library-head > view { display:flex; min-width:0; flex-direction:column; text-align:left; }
.library-title { color:#26314d; font-size:27rpx; font-weight:700; }
.library-subtitle { margin-top:8rpx; color:#929bb0; font-size:20rpx; }
.upload-button { flex:0 0 auto; height:60rpx; margin:0; padding:0 22rpx; border:0; border-radius:30rpx; color:#fff; background:#566df5; font-size:23rpx; line-height:60rpx; }
.upload-button::after { border:0; }
.video-list { max-height:360rpx; border-top:1rpx solid #e7eafb; }
.video-item { display:flex; align-items:center; gap:18rpx; min-height:102rpx; padding:14rpx 22rpx; border-bottom:1rpx solid #eaedf7; background:#fff; }
.video-item:last-child { border-bottom:0; }
.video-icon { display:flex; align-items:center; justify-content:center; width:66rpx; height:66rpx; border-radius:18rpx; color:#566df5; background:#edf0ff; font-size:30rpx; }
.video-info { display:flex; min-width:0; flex:1; flex-direction:column; text-align:left; }
.video-name { overflow:hidden; color:#27324d; font-size:25rpx; font-weight:600; text-overflow:ellipsis; white-space:nowrap; }
.video-size { margin-top:6rpx; color:#9aa3b6; font-size:20rpx; }
.select-text { color:#566df5; font-size:23rpx; font-weight:600; }
.empty { display:flex; align-items:center; justify-content:center; gap:12rpx; min-height:150rpx; padding:24rpx; border-top:1rpx solid #e7eafb; color:#99a2b6; font-size:23rpx; }
.empty-icon { font-size:30rpx; }
</style>
