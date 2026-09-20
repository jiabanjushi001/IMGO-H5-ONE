<template>
	<view class="scan-page">
		<get-qrcode v-if="canLiveScan" @success="qrcodeSuccess" @error="qrcodeError" />
		<view v-else class="scan-fallback">
			<view class="scan-icon">▣</view>
			<view class="scan-title">扫描二维码</view>
			<view class="scan-hint">{{ cameraMessage }}</view>
		</view>
		<view class="scan-actions">
			<button class="cu-btn bg-green lg" @tap="pickImage(true)">拍照识别</button>
			<button class="cu-btn bg-white lg" @tap="pickImage(false)">从相册选择</button>
		</view>
	</view>
</template>

<script>
	import jsQR from 'jsqr'
	import scan from '@/common/scan.js'
	import getQrcode from '@/components/get-qrcode.vue'

	export default {
		components: { getQrcode },
		data() {
			return {
				canLiveScan: false,
				cameraMessage: '正在检查摄像头…',
				fileInput: null
			}
		},
		mounted() {
			// LAN HTTP is not a secure context; browsers do not expose getUserMedia there.
			this.canLiveScan = !!(window.isSecureContext && navigator.mediaDevices?.getUserMedia)
			if (!this.canLiveScan) {
				this.cameraMessage = '内网 HTTP 无法实时调用摄像头。可以拍照或从相册选择二维码；实时扫码需要 HTTPS。'
			}
			const input = document.createElement('input')
			input.type = 'file'
			input.accept = 'image/*'
			input.style.display = 'none'
			input.addEventListener('change', this.onImageSelected)
			document.body.appendChild(input)
			this.fileInput = input
		},
		unmounted() {
			this.removeFileInput()
		},
		onUnload() {
			this.removeFileInput()
		},
		methods: {
			removeFileInput() {
				if (!this.fileInput) return
				this.fileInput.removeEventListener('change', this.onImageSelected)
				this.fileInput.remove()
				this.fileInput = null
			},
			qrcodeSuccess(data) {
				scan.checkQr(data, true)
			},
			qrcodeError(error) {
				console.warn('摄像头不可用:', error)
				this.canLiveScan = false
				this.cameraMessage = '摄像头不可用或未授权。仍可拍照或从相册选择二维码。'
			},
			pickImage(capture) {
				if (!this.fileInput) return
				this.fileInput.value = ''
				if (capture) this.fileInput.setAttribute('capture', 'environment')
				else this.fileInput.removeAttribute('capture')
				this.fileInput.click()
			},
			async onImageSelected(event) {
				const file = event.target.files?.[0]
				if (!file) return
				if (!file.type.startsWith('image/')) {
					uni.showToast({ title: '请选择图片', icon: 'none' })
					return
				}
				uni.showLoading({ title: '识别中' })
				try {
					const content = await this.decodeImage(file)
					if (content) this.qrcodeSuccess(content)
					else uni.showToast({ title: '未识别到二维码，请重试', icon: 'none' })
				} catch (error) {
					uni.showToast({ title: '图片读取失败，请重试', icon: 'none' })
				} finally {
					uni.hideLoading()
				}
			},
			decodeImage(file) {
				return new Promise((resolve, reject) => {
					const objectURL = URL.createObjectURL(file)
					const image = new Image()
					image.onload = () => {
						URL.revokeObjectURL(objectURL)
						try {
							const scale = Math.min(1, 2048 / Math.max(image.naturalWidth, image.naturalHeight))
							const canvas = document.createElement('canvas')
							canvas.width = Math.max(1, Math.round(image.naturalWidth * scale))
							canvas.height = Math.max(1, Math.round(image.naturalHeight * scale))
							const context = canvas.getContext('2d', { willReadFrequently: true })
							context.drawImage(image, 0, 0, canvas.width, canvas.height)
							const pixels = context.getImageData(0, 0, canvas.width, canvas.height)
							const result = jsQR(pixels.data, pixels.width, pixels.height, { inversionAttempts: 'attemptBoth' })
							resolve(result?.data || '')
						} catch (error) {
							reject(error)
						}
					}
					image.onerror = () => {
						URL.revokeObjectURL(objectURL)
						reject(new Error('image decode failed'))
					}
					image.src = objectURL
				})
			}
		}
	}
</script>

<style scoped>
	.scan-page { min-height: 100vh; background: #f4f7f4; position: relative; }
	.scan-fallback { padding: 28vh 48rpx 0; text-align: center; color: #5a665b; }
	.scan-icon { font-size: 96rpx; color: #39b54a; }
	.scan-title { font-size: 38rpx; font-weight: 600; color: #26342a; margin: 24rpx 0; }
	.scan-hint { font-size: 28rpx; line-height: 1.7; }
	.scan-actions { position: fixed; z-index: 30; bottom: 80rpx; left: 32rpx; right: 32rpx; display: flex; gap: 20rpx; }
	.scan-actions button { flex: 1; }
</style>
