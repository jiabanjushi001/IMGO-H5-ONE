<script setup>
import { computed, ref, watch } from 'vue'
import { identityUploadUrl } from '@/api/identity.js'
import { resolveMediaDisplayUrl } from '@/utils/avatar.js'

const props = defineProps({
	label: { type: String, required: true },
	hint: { type: String, default: '' },
	side: { type: String, required: true },
	src: { type: String, default: '' },
	readonly: { type: Boolean, default: false }
})
const emit = defineEmits(['uploaded'])

const uploading = ref(false)
const displaySrc = ref('')
const hasImage = computed(() => !!displaySrc.value)

watch(() => props.src, async (value) => {
	if (!value) {
		displaySrc.value = ''
		return
	}
	try {
		displaySrc.value = await resolveMediaDisplayUrl(value)
	} catch (error) {
		displaySrc.value = value
	}
}, { immediate: true })

function imageExtension(result) {
	const file = result.tempFiles && result.tempFiles[0]
	const type = String(file && file.type || '').toLowerCase()
	if (type.includes('png')) return 'png'
	if (type.includes('webp')) return 'webp'
	const path = String(result.tempFilePaths && result.tempFilePaths[0] || '')
	const match = path.match(/\.([a-z0-9]+)(?:[?#]|$)/i)
	return match ? match[1].toLowerCase() : 'jpg'
}

function selectImage() {
	if (props.readonly || uploading.value) {
		if (hasImage.value) preview()
		return
	}
	uni.chooseImage({
		count: 1,
		sizeType: ['compressed'],
		sourceType: ['album', 'camera'],
		success: upload
	})
}

function upload(result) {
	const filePath = result.tempFilePaths && result.tempFilePaths[0]
	if (!filePath) return
	uploading.value = true
	uni.showLoading({ title: '上传中…', mask: true })
	uni.uploadFile({
		url: identityUploadUrl,
		filePath,
		name: 'file',
		header: { Authorization: uni.getStorageSync('authToken') },
		formData: { ext: imageExtension(result) },
		success: async response => {
			let body
			try {
				body = typeof response.data === 'string' ? JSON.parse(response.data) : response.data
			} catch (error) {
				uni.showToast({ title: '上传返回异常', icon: 'none' })
				return
			}
			const file = body && body.data
			if (Number(body && body.code) !== 0 || !file || !Number(file.file_id) || !file.url) {
				uni.showToast({ title: body && body.msg || '照片上传失败', icon: 'none' })
				return
			}
			try {
				displaySrc.value = await resolveMediaDisplayUrl(file.url)
			} catch (error) {
				displaySrc.value = file.url
			}
			emit('uploaded', { side: props.side, fileId: Number(file.file_id), url: file.url })
			uni.showToast({ title: `${props.label}上传成功`, icon: 'none' })
		},
		fail: () => uni.showToast({ title: '照片上传失败，请检查网络', icon: 'none' }),
		complete: () => {
			uploading.value = false
			uni.hideLoading()
		}
	})
}

function preview() {
	if (!displaySrc.value) return
	uni.previewImage({ current: displaySrc.value, urls: [displaySrc.value] })
}
</script>

<template>
	<view class="document-upload" :class="{ 'has-image': hasImage, 'is-readonly': readonly }" @tap="selectImage">
		<image v-if="hasImage" :src="displaySrc" mode="aspectFill" class="document-image" />
		<view v-else class="document-placeholder">
			<text class="cuIcon-cameraadd document-icon"></text>
			<text class="document-label">{{ uploading ? '上传中…' : label }}</text>
			<text class="document-hint">{{ hint }}</text>
		</view>
		<view v-if="hasImage" class="document-overlay">
			<text class="document-overlay-label">{{ readonly ? '点击查看' : '重新上传' }}</text>
		</view>
	</view>
</template>

<style scoped>
.document-upload { position:relative; overflow:hidden; display:flex; align-items:center; justify-content:center; width:100%; height:326rpx; border:2rpx dashed #cad2f7; border-radius:24rpx; background:linear-gradient(145deg,#f8f9ff,#eef1ff); box-sizing:border-box; }
.document-upload:active { opacity:.84; }
.document-upload.has-image { border-style:solid; border-color:#d9def3; background:#eef1f8; }
.document-image { width:100%; height:100%; display:block; }
.document-placeholder { display:flex; flex-direction:column; align-items:center; padding:28rpx; text-align:center; }
.document-icon { display:flex; align-items:center; justify-content:center; width:86rpx; height:86rpx; margin-bottom:18rpx; border-radius:25rpx; color:#5269ed; background:#fff; box-shadow:0 10rpx 28rpx rgba(65,79,177,.12); font-size:42rpx; }
.document-label { color:#27334d; font-size:28rpx; font-weight:700; }
.document-hint { margin-top:10rpx; color:#8a95a8; font-size:22rpx; line-height:1.5; }
.document-overlay { position:absolute; right:16rpx; bottom:16rpx; padding:8rpx 18rpx; border-radius:999rpx; color:#fff; background:rgba(31,41,74,.68); font-size:21rpx; backdrop-filter:blur(8px); }
.document-upload.is-readonly .document-overlay { background:rgba(76,99,233,.78); }
</style>
