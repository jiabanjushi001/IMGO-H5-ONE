import config from '@/common/config.js'

const inviteCodeMarker = 'inviteCode='
const apiAvatarPathPattern = /\/(?:avatar|storage)\/[^"'()\s,]*/i
const runtimeAvatarPathPattern = /\/(?:avatar|storage)\/[^"'()\s,]*/i
const absoluteAvatarUrlPattern = /https?:\/\/[^"'()\s]+\/(?:avatar|storage)\/[^"'()\s,]*/i

let avatarRepairObserver
const authBlobCache = new Map()
const authBlobPending = new Map()

const decodeSafely = (value) => {
	try {
		return decodeURIComponent(value)
	} catch (error) {
		return value
	}
}

const getApiBase = () => {
	const runtime = typeof window !== 'undefined' ? String(window.httpUrl || '').trim() : ''
	const base = (runtime || String(config.apiUrl || '')).replace(/\/+$/, '')
	return base
}

const getAuthHeader = () => {
	try {
		if (typeof uni !== 'undefined' && typeof uni.getStorageSync === 'function') {
			const token = uni.getStorageSync('authToken')
			if (token) return token
		}
	} catch (error) {}
	try {
		return window.localStorage.getItem('authToken') || ''
	} catch (error) {
		return ''
	}
}

const getAvatarIdentity = (info = {}) => {
	const name = info.displayName || info.realname || info.account || info.name || '用户'
	const id = info.user_id ?? info.id ?? 0
	return {
		name: encodeURIComponent(String(name)),
		id: encodeURIComponent(String(id))
	}
}

export const getDefaultAvatarUrl = (info = {}) => {
	const identity = getAvatarIdentity(info)
	return `${getApiBase()}/avatar/${identity.name}/120/${identity.id}`
}

const extractApiAvatarPath = (value) => {
	const match = value.match(apiAvatarPathPattern)
	return match ? match[0] : ''
}

/** 将任意媒体/头像路径规范为可请求的绝对地址（空值不回退默认头像） */
export const normalizeMediaUrl = (url = '') => {
	let value = typeof url === 'string' ? url.trim() : ''
	if (!value) return ''

	const apiBaseUrl = getApiBase()
	const markerIndex = value.indexOf(inviteCodeMarker)
	if (markerIndex !== -1) {
		value = decodeSafely(value.slice(markerIndex + inviteCodeMarker.length))
	}

	if (/^(?:data:|blob:)/i.test(value)) {
		return value
	}

	const apiPath = extractApiAvatarPath(value)
	if (apiPath) {
		return `${apiBaseUrl}${apiPath}`
	}

	if (/^(?:https?:\/\/|\/\/)/i.test(value)) {
		return value
	}

	if (value.startsWith('/')) {
		return `${apiBaseUrl}${value}`
	}

	return `${apiBaseUrl}/${value}`
}

export const normalizeAvatarUrl = (avatar, info = {}) => {
	const value = typeof avatar === 'string' ? avatar.trim() : ''
	if (!value) return getDefaultAvatarUrl(info)
	return normalizeMediaUrl(value)
}

/** 聊天图片/视频封面等：storage 鉴权转 blob，其它原样 */
export const resolveMediaDisplayUrl = (url) => {
	const absoluteUrl = normalizeMediaUrl(url)
	if (!absoluteUrl) return Promise.resolve('')
	return ensureAuthedDisplayUrl(absoluteUrl).catch(() => absoluteUrl)
}

export const normalizeAvatarData = (value, seen = new WeakSet()) => {
	if (!value || typeof value !== 'object' || seen.has(value)) {
		return value
	}

	seen.add(value)

	if (Array.isArray(value)) {
		value.forEach((item) => normalizeAvatarData(item, seen))
		return value
	}

	if (Object.prototype.hasOwnProperty.call(value, 'avatar')) {
		value.avatar = normalizeAvatarUrl(value.avatar, value)
	}

	Object.keys(value).forEach((key) => {
		if (key !== 'avatar') {
			normalizeAvatarData(value[key], seen)
		}
	})

	return value
}

const getRuntimeFallbackUrl = (sourcePath = '') => {
	const value = String(sourcePath)
	const groupId = value.match(/\/avatar\/group-(\d+)/i)
	const storageId = value.match(/\/storage\/image\/[^/]+\/([^/]+)\//i)
	const avatarId = value.match(/\/avatar\/[^/]+\/\d+\/([^/?#]+)/i)
	const id = groupId?.[1] || storageId?.[1] || avatarId?.[1] || 0
	const name = groupId ? `group-${groupId[1]}` : (avatarId ? '用户' : '用户')
	return getDefaultAvatarUrl({ id, name })
}

const repairStoredAvatarData = (key) => {
	try {
		const raw = window.localStorage.getItem(key)
		if (!raw) return

		const stored = JSON.parse(raw)
		normalizeAvatarData(stored?.data !== undefined ? stored.data : stored)
		window.localStorage.setItem(key, JSON.stringify(stored))
	} catch (error) {
		// 缓存内容异常时不影响应用启动。
	}
}

/**
 * 需要带 Authorization 才能下载的媒体：
 * - /storage/ 上传文件
 * - /avatar/group-* 群头像（无 token 返回 401）
 * - /avatar/...?v= 带版本号的自定义头像（通常同样需鉴权）
 * 公开字母头像 /avatar/name/120/id 不需要。
 */
export const needsAuthMediaUrl = (url = '') => {
	const value = String(url || '')
	if (/\/storage\//i.test(value)) return true
	if (/\/avatar\/group-/i.test(value)) return true
	if (/\/avatar\//i.test(value) && /[?&]v=/i.test(value)) return true
	return false
}

/**
 * /storage、需鉴权的 /avatar 无法用 img/CSS 带头，
 * 用带鉴权的 fetch 转成 blob: URL 后再显示。
 */
export const ensureAuthedDisplayUrl = (absoluteUrl) => {
	if (!absoluteUrl || /^(?:blob:|data:)/i.test(absoluteUrl)) {
		return Promise.resolve(absoluteUrl)
	}
	if (!needsAuthMediaUrl(absoluteUrl)) {
		return Promise.resolve(absoluteUrl)
	}
	if (authBlobCache.has(absoluteUrl)) {
		return Promise.resolve(authBlobCache.get(absoluteUrl))
	}
	if (authBlobPending.has(absoluteUrl)) {
		return authBlobPending.get(absoluteUrl)
	}

	const task = fetch(absoluteUrl, {
		headers: (() => {
			const auth = getAuthHeader()
			return auth ? { Authorization: auth } : {}
		})(),
		credentials: 'omit',
		cache: 'no-store',
	}).then(async (response) => {
		if (!response.ok) throw new Error(`avatar HTTP ${response.status}`)
		const blob = await response.blob()
		const blobUrl = URL.createObjectURL(blob)
		authBlobCache.set(absoluteUrl, blobUrl)
		authBlobPending.delete(absoluteUrl)
		return blobUrl
	}).catch((error) => {
		authBlobPending.delete(absoluteUrl)
		throw error
	})

	authBlobPending.set(absoluteUrl, task)
	return task
}

export const getAuthHeaders = () => {
	const auth = getAuthHeader()
	return auth ? { Authorization: auth } : {}
}

/**
 * 带鉴权下载媒体文件。
 * WebView/H5：返回可播放/可预览的 blob: URL；原生端：uni.downloadFile 临时路径。
 */
export const downloadAuthedFile = (url) => {
	const absoluteUrl = normalizeMediaUrl(url)
	if (!absoluteUrl) {
		return Promise.reject(new Error('empty media url'))
	}
	if (/^(?:blob:|data:|file:|wxfile:)/i.test(absoluteUrl)) {
		return Promise.resolve({ tempFilePath: absoluteUrl, displayUrl: absoluteUrl })
	}

	const canFetchBlob = typeof fetch === 'function' && typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function'

	if (needsAuthMediaUrl(absoluteUrl) && canFetchBlob) {
		return ensureAuthedDisplayUrl(absoluteUrl).then((blobUrl) => ({
			tempFilePath: blobUrl,
			displayUrl: blobUrl
		}))
	}

	if (!needsAuthMediaUrl(absoluteUrl) && canFetchBlob) {
		return Promise.resolve({ tempFilePath: absoluteUrl, displayUrl: absoluteUrl })
	}

	return new Promise((resolve, reject) => {
		uni.downloadFile({
			url: absoluteUrl,
			header: getAuthHeaders(),
			success: (res) => {
				if (res.statusCode === 200) {
					resolve({ tempFilePath: res.tempFilePath, displayUrl: res.tempFilePath })
				} else {
					reject(new Error(`download HTTP ${res.statusCode}`))
				}
			},
			fail: reject
		})
	})
}

/** 触发浏览器下载（H5）；其它端返回临时路径供调用方保存 */
export const triggerAuthedDownload = (url, fileName = 'file') => {
	return downloadAuthedFile(url).then(({ displayUrl, tempFilePath }) => {
		if (typeof document !== 'undefined') {
			const tempLink = document.createElement('a')
			tempLink.style.display = 'none'
			tempLink.href = displayUrl
			tempLink.setAttribute('download', fileName || 'file')
			tempLink.setAttribute('target', '_blank')
			document.body.appendChild(tempLink)
			tempLink.click()
			document.body.removeChild(tempLink)
		}
		return { displayUrl, tempFilePath }
	})
}

/** 组件侧异步解析可直接绑定的头像地址（storage → blob） */
export const resolveAvatarDisplayUrl = (avatar, info = {}) => {
	const absoluteUrl = normalizeAvatarUrl(avatar, info)
	return ensureAuthedDisplayUrl(absoluteUrl).catch(() => absoluteUrl)
}

const extractBackgroundUrl = (background = '') => {
	const match = background.match(/url\(\s*(['"]?)([^"')]+)\1\s*\)/i)
	return match ? match[2] : ''
}

const toAbsoluteAvatarUrl = (value = '') => {
	if (!value) return ''
	if (/^(?:blob:|data:)/i.test(value)) return value
	if (/^https?:\/\//i.test(value)) return value
	const path = extractApiAvatarPath(value)
	if (path) return `${getApiBase()}${path}`
	if (value.startsWith('/')) return `${getApiBase()}${value}`
	return value
}

const applyBackgroundUrl = (element, url) => {
	element.style.backgroundImage = `url("${url}")`
}

const readElementBackgroundUrl = (element) => {
	const fromImage = extractBackgroundUrl(element.style.backgroundImage || '')
	if (fromImage) return fromImage
	return extractBackgroundUrl(element.style.background || '')
}

const repairAvatarBackground = (element) => {
	if (!element?.style) return

	const rawUrl = readElementBackgroundUrl(element)
	if (!rawUrl) return
	if (/^(?:blob:|data:)/i.test(rawUrl)) return

	const absoluteUrl = toAbsoluteAvatarUrl(rawUrl)
	if (!/\/(?:avatar|storage)\//i.test(absoluteUrl)) return

	const sourcePath = extractApiAvatarPath(absoluteUrl) || absoluteUrl
	const fallback = needsAuthMediaUrl(absoluteUrl) ? getRuntimeFallbackUrl(sourcePath) : ''

	if (needsAuthMediaUrl(absoluteUrl)) {
		const cached = authBlobCache.get(absoluteUrl)
		// Vue 可能把已修好的 blob 又覆盖回需鉴权地址，需立刻还原
		if (cached) {
			applyBackgroundUrl(element, cached)
			element.dataset.avatarAuthSrc = absoluteUrl
			element.dataset.avatarAuthReady = '1'
			return
		}
		if (element.dataset.avatarAuthSrc === absoluteUrl && element.dataset.avatarAuthReady === 'pending') {
			return
		}
		element.dataset.avatarAuthSrc = absoluteUrl
		element.dataset.avatarAuthReady = 'pending'
		if (fallback) applyBackgroundUrl(element, fallback)
		ensureAuthedDisplayUrl(absoluteUrl).then((blobUrl) => {
			if (element.dataset.avatarAuthSrc !== absoluteUrl) return
			applyBackgroundUrl(element, blobUrl)
			element.dataset.avatarAuthReady = '1'
		}).catch(() => {
			if (fallback) applyBackgroundUrl(element, fallback)
			element.dataset.avatarAuthReady = '0'
		})
		return
	}

	applyBackgroundUrl(element, absoluteUrl)
}

const repairAvatarImage = (image) => {
	if (!image?.getAttribute) return

	const source = image.getAttribute('src') || ''
	if (!source || /^(?:blob:|data:)/i.test(source)) return

	const absoluteUrl = toAbsoluteAvatarUrl(source)
	if (!/\/(?:avatar|storage)\//i.test(absoluteUrl)) return

	const sourcePath = extractApiAvatarPath(absoluteUrl) || absoluteUrl
	// 仅头像容器失败时回退字母图；聊天图片等保持空白等待鉴权结果
	const useAvatarFallback = !!(image.closest && (
		image.closest('.cu-avatar') ||
		image.closest('.member-avatar') ||
		image.closest('.group-hero-avatar') ||
		image.classList?.contains('avatar-row-image') ||
		image.classList?.contains('member-avatar-image') ||
		image.classList?.contains('mine-hero-avatar-image') ||
		image.classList?.contains('my-avatar')
	))
	const fallback = useAvatarFallback && needsAuthMediaUrl(absoluteUrl)
		? getRuntimeFallbackUrl(sourcePath)
		: ''

	if (needsAuthMediaUrl(absoluteUrl)) {
		const cached = authBlobCache.get(absoluteUrl)
		if (cached) {
			image.setAttribute('src', cached)
			image.dataset.avatarAuthSrc = absoluteUrl
			image.dataset.avatarAuthReady = '1'
			return
		}
		if (image.dataset.avatarAuthSrc === absoluteUrl && image.dataset.avatarAuthReady === 'pending') {
			return
		}
		image.dataset.avatarAuthSrc = absoluteUrl
		image.dataset.avatarAuthReady = 'pending'
		if (fallback) image.setAttribute('src', fallback)
		ensureAuthedDisplayUrl(absoluteUrl).then((blobUrl) => {
			if (image.dataset.avatarAuthSrc !== absoluteUrl) return
			image.setAttribute('src', blobUrl)
			image.dataset.avatarAuthReady = '1'
		}).catch(() => {
			if (fallback) image.setAttribute('src', fallback)
			image.dataset.avatarAuthReady = '0'
		})
		return
	}

	if (source !== absoluteUrl) {
		image.setAttribute('src', absoluteUrl)
	}
}

const repairRenderedAvatars = (root) => {
	if (!root) return

	if (root.nodeType === 1) {
		repairAvatarBackground(root)
		if (String(root.tagName).toLowerCase() === 'img') {
			repairAvatarImage(root)
		}
	}

	if (!root.querySelectorAll) return

	root.querySelectorAll('.cu-avatar, [style*="background-image"]').forEach(repairAvatarBackground)
	root.querySelectorAll('img').forEach(repairAvatarImage)
}

/** 批量给列表项补可显示地址（原字段不变，写入 display 字段） */
export const mapAuthDisplayField = async (list, sourceField = 'src', displayField = 'displaySrc') => {
	const arr = Array.isArray(list) ? list : []
	await Promise.all(arr.map(async (item) => {
		if (!item || typeof item !== 'object') return
		const raw = item[sourceField]
		if (!raw) {
			item[displayField] = ''
			return
		}
		try {
			item[displayField] = await resolveMediaDisplayUrl(raw)
		} catch (error) {
			item[displayField] = raw
		}
	}))
	return arr
}

export const installAvatarRepair = () => {
	if (
		typeof window === 'undefined' ||
		typeof document === 'undefined' ||
		typeof MutationObserver === 'undefined'
	) {
		return
	}

	repairStoredAvatarData('userInfo')
	repairStoredAvatarData('allContacts')

	const start = () => {
		repairRenderedAvatars(document)
		if (avatarRepairObserver) return

		avatarRepairObserver = new MutationObserver((records) => {
			records.forEach((record) => {
				if (record.type === 'attributes') {
					repairRenderedAvatars(record.target)
				}
				record.addedNodes.forEach(repairRenderedAvatars)
			})
		})

		avatarRepairObserver.observe(document.documentElement, {
			childList: true,
			subtree: true,
			attributes: true,
			attributeFilter: ['style', 'src']
		})
	}

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', start, { once: true })
	} else {
		start()
	}
}
