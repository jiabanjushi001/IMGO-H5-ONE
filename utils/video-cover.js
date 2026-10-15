import msgApi from '@/api/message.js'
import { getAuthHeaders } from '@/utils/avatar.js'

const PLACEHOLDER_POSTER_PATTERN = /(?:^|\/)video-placeholder\.svg(?:[?#]|$)/i

export const isPlaceholderVideoPoster = (poster = '') => {
	const value = String(poster || '').trim()
	return !value || PLACEHOLDER_POSTER_PATTERN.test(value)
}

const parseUploadResponse = (response) => {
	let body = response
	if (response && Object.prototype.hasOwnProperty.call(response, 'statusCode') && Object.prototype.hasOwnProperty.call(response, 'data')) {
		body = response.data
	}
	if (typeof body === 'string') body = JSON.parse(body)
	if (Number(body && body.code) !== 0) {
		throw new Error(body && body.msg || '视频封面上传失败')
	}
	const file = body && body.data
	const url = typeof file === 'string' ? file : (file && (file.url || file.path))
	if (!url) throw new Error('视频封面地址为空')
	return url
}

const uploadPosterPath = (filePath) => new Promise((resolve, reject) => {
	uni.uploadFile({
		url: msgApi.uploadUrl,
		filePath,
		name: 'file',
		header: getAuthHeaders(),
		formData: { ext: 'jpg' },
		success: (response) => {
			try {
				resolve(parseUploadResponse(response))
			} catch (error) {
				reject(error)
			}
		},
		fail: reject
	})
})

const capturePosterBlob = (result) => new Promise((resolve, reject) => {
	if (typeof document === 'undefined' || typeof URL === 'undefined' || !result?.tempFile) {
		reject(new Error('当前环境不支持生成视频封面'))
		return
	}

	const video = document.createElement('video')
	const objectUrl = URL.createObjectURL(result.tempFile)
	let settled = false
	let timeoutId = null

	const cleanup = () => {
		if (timeoutId) clearTimeout(timeoutId)
		video.removeAttribute('src')
		video.load()
		URL.revokeObjectURL(objectUrl)
	}
	const fail = (error) => {
		if (settled) return
		settled = true
		cleanup()
		reject(error instanceof Error ? error : new Error('视频封面生成失败'))
	}
	const drawFrame = () => {
		if (settled) return
		const sourceWidth = Number(video.videoWidth || result.width || 0)
		const sourceHeight = Number(video.videoHeight || result.height || 0)
		if (!sourceWidth || !sourceHeight) {
			fail(new Error('无法读取视频尺寸'))
			return
		}
		const maxSide = 640
		const scale = Math.min(1, maxSide / Math.max(sourceWidth, sourceHeight))
		const canvas = document.createElement('canvas')
		canvas.width = Math.max(1, Math.round(sourceWidth * scale))
		canvas.height = Math.max(1, Math.round(sourceHeight * scale))
		const context = canvas.getContext('2d')
		if (!context) {
			fail(new Error('无法创建视频封面画布'))
			return
		}
		context.drawImage(video, 0, 0, canvas.width, canvas.height)
		canvas.toBlob((blob) => {
			if (!blob) {
				fail(new Error('视频封面生成失败'))
				return
			}
			settled = true
			cleanup()
			resolve(blob)
		}, 'image/jpeg', 0.82)
	}

	video.muted = true
	video.playsInline = true
	video.preload = 'auto'
	video.addEventListener('error', () => fail(new Error('视频读取失败')), { once: true })
	video.addEventListener('loadeddata', () => {
		const duration = Number(video.duration || result.duration || 0)
		const targetTime = duration > 0.15 ? Math.min(1, Math.max(0.1, duration * 0.05)) : 0
		if (targetTime > 0) {
			video.addEventListener('seeked', drawFrame, { once: true })
			try {
				video.currentTime = targetTime
			} catch (error) {
				drawFrame()
			}
		} else {
			drawFrame()
		}
	}, { once: true })
	timeoutId = setTimeout(() => fail(new Error('视频封面生成超时')), 6000)
	video.src = objectUrl
	video.load()
})

const uploadPosterBlob = async (blob) => {
	const formData = new FormData()
	formData.append('file', blob, `video-cover-${Date.now()}.jpg`)
	formData.append('ext', 'jpg')
	const response = await fetch(msgApi.uploadUrl, {
		method: 'POST',
		headers: getAuthHeaders(),
		body: formData,
		credentials: 'omit'
	})
	if (!response.ok) throw new Error(`视频封面上传失败 (${response.status})`)
	return parseUploadResponse(await response.json())
}

/**
 * 尽量为选中的视频生成并上传真实封面。
 * H5 使用原始 File 抽帧；部分小程序若返回 thumbTempFilePath，则直接上传缩略图。
 */
export const createAndUploadVideoPoster = async (result = {}) => {
	if (result.thumbTempFilePath) return uploadPosterPath(result.thumbTempFilePath)
	if (typeof document === 'undefined' || typeof fetch !== 'function' || !result.tempFile) return ''
	const blob = await capturePosterBlob(result)
	return uploadPosterBlob(blob)
}
