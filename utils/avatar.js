import config from '@/common/config.js'

const apiBaseUrl = String(config.apiUrl || '').replace(/\/$/, '')
const inviteCodeMarker = 'inviteCode='
const apiAvatarPathPattern = /\/(?:avatar|storage)\/[^"'()\s,]*/i
const runtimeAvatarPathPattern = /\/(?:avatar|storage)\/[^"'()\s,]*/i

let avatarRepairObserver

const decodeSafely = (value) => {
	try {
		return decodeURIComponent(value)
	} catch (error) {
		return value
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
	return `${apiBaseUrl}/avatar/${identity.name}/120/${identity.id}`
}

const extractApiAvatarPath = (value) => {
	const match = value.match(apiAvatarPathPattern)
	return match ? match[0] : ''
}

export const normalizeAvatarUrl = (avatar, info = {}) => {
	let value = typeof avatar === 'string' ? avatar.trim() : ''

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

	return value ? `${apiBaseUrl}/${value}` : getDefaultAvatarUrl(info)
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
	const storageId = String(sourcePath).match(/\/storage\/image\/[^/]+\/([^/]+)\//i)
	return getDefaultAvatarUrl({ id: storageId ? storageId[1] : 0 })
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

const repairAvatarBackground = (element) => {
	if (!element?.style) return

	const background = element.style.backgroundImage || ''
	const pathMatch = background.match(runtimeAvatarPathPattern)
	if (!pathMatch) return

	const sourcePath = pathMatch[0]
	const primary = `${apiBaseUrl}${sourcePath}`
	const fallback = /^\/storage\//i.test(sourcePath) ? getRuntimeFallbackUrl(sourcePath) : ''
	const nextBackground = `url("${primary}")${fallback ? `, url("${fallback}")` : ''}`

	if (background !== nextBackground) {
		element.style.backgroundImage = nextBackground
	}
}

const repairAvatarImage = (image) => {
	if (!image?.getAttribute) return

	const source = image.getAttribute('src') || ''
	const pathMatch = source.match(runtimeAvatarPathPattern)
	if (!pathMatch) return

	const sourcePath = pathMatch[0]
	const primary = `${apiBaseUrl}${sourcePath}`
	const fallback = /^\/storage\//i.test(sourcePath) ? getRuntimeFallbackUrl(sourcePath) : ''

	if (source !== primary) {
		image.setAttribute('src', primary)
	}

	if (fallback && !image.getAttribute('data-avatar-fallback')) {
		image.setAttribute('data-avatar-fallback', fallback)
		image.addEventListener('error', () => {
			if (image.getAttribute('src') !== fallback) {
				image.setAttribute('src', fallback)
			}
		})
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

export const installAvatarRepair = () => {
	if (
		!apiBaseUrl ||
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
