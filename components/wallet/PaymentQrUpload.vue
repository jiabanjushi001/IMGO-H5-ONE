<script setup>
import { ref, watch } from 'vue'
import { paymentQrUploadUrl } from '@/api/payment-method.js'
import { resolveMediaDisplayUrl } from '@/utils/avatar.js'

const props = defineProps({
	src: { type: String, default: '' },
	label: { type: String, default: '收款码' }
})
const emit = defineEmits(['uploaded'])
const uploading = ref(false)
const displaySrc = ref('')

watch(() => props.src, async value => {
	if (!value) { displaySrc.value = ''; return }
	try { displaySrc.value = await resolveMediaDisplayUrl(value) }
	catch (_) { displaySrc.value = value }
}, { immediate: true })

function extension(result) {
	const file = result.tempFiles && result.tempFiles[0]
	const type = String(file && file.type || '').toLowerCase()
	if (type.includes('png')) return 'png'
	if (type.includes('webp')) return 'webp'
	return 'jpg'
}

function choose() {
	if (uploading.value) return
	uni.chooseImage({ count: 1, sizeType: ['compressed'], sourceType: ['album', 'camera'], success: upload })
}

function upload(result) {
	const filePath = result.tempFilePaths && result.tempFilePaths[0]
	if (!filePath) return
	uploading.value = true
	uni.showLoading({ title: '上传中…', mask: true })
	uni.uploadFile({
		url: paymentQrUploadUrl,
		filePath,
		name: 'file',
		header: { Authorization: uni.getStorageSync('authToken') },
		formData: { ext: extension(result) },
		success: async response => {
			let body
			try { body = typeof response.data === 'string' ? JSON.parse(response.data) : response.data }
			catch (_) { uni.showToast({ title: '上传返回异常', icon: 'none' }); return }
			const file = body && body.data
			if (Number(body && body.code) !== 0 || !Number(file && file.file_id) || !file.url) {
				uni.showToast({ title: body && body.msg || '收款码上传失败', icon: 'none' })
				return
			}
			try { displaySrc.value = await resolveMediaDisplayUrl(file.url) }
			catch (_) { displaySrc.value = file.url }
			emit('uploaded', { fileId: Number(file.file_id), url: file.url })
		},
		fail: () => uni.showToast({ title: '收款码上传失败，请检查网络', icon: 'none' }),
		complete: () => { uploading.value = false; uni.hideLoading() }
	})
}
</script>

<template>
	<view class="qr-upload" :class="{ 'has-image': !!displaySrc }" @tap="choose">
		<image v-if="displaySrc" class="qr-image" :src="displaySrc" mode="aspectFit" />
		<view v-else class="qr-placeholder">
			<text class="cuIcon-cameraadd qr-icon"></text>
			<text class="qr-title">{{ uploading ? '上传中…' : `上传${props.label}` }}</text>
			<text class="qr-hint">请上传清晰、完整的二维码图片</text>
		</view>
		<text v-if="displaySrc" class="qr-replace">点击更换</text>
	</view>
</template>

<style scoped>
.qr-upload { position:relative; display:flex; align-items:center; justify-content:center; width:100%; height:340rpx; overflow:hidden; box-sizing:border-box; border:2rpx dashed #cad2f7; border-radius:24rpx; background:linear-gradient(145deg,#f8f9ff,#eef1ff); }
.qr-upload.has-image { border-style:solid; background:#fff; }
.qr-image { width:100%; height:100%; }
.qr-placeholder { display:flex; flex-direction:column; align-items:center; padding:30rpx; text-align:center; }
.qr-icon { display:flex; align-items:center; justify-content:center; width:82rpx; height:82rpx; border-radius:24rpx; background:#fff; color:#5269ed; box-shadow:0 10rpx 28rpx rgba(65,79,177,.12); font-size:40rpx; }
.qr-title { margin-top:18rpx; color:#29344d; font-size:27rpx; font-weight:700; }
.qr-hint { margin-top:10rpx; color:#8b96a8; font-size:21rpx; }
.qr-replace { position:absolute; right:16rpx; bottom:16rpx; padding:8rpx 18rpx; border-radius:999rpx; color:#fff; background:rgba(31,41,74,.68); font-size:21rpx; }
</style>
