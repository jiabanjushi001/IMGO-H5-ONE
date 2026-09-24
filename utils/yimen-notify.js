/**
 * 一门 APP 本地通知封装（ym-jsbridge）。
 * 浏览器 / 普通 H5：全部 no-op，不影响 release:h5。
 * 仅在 jsBridge.inApp 为 true 时调用原生能力。
 * 点通知只激活 App，不跳转特定页面。
 */
import jsBridge from 'ym-jsbridge'

/** P1 真机冒烟已通过，默认关闭。需要再验 bridge 时可临时改为 true。 */
export const ENABLE_YIMEN_NOTIFY_P1_SMOKE = false

const NOTIFY_THROTTLE_MS = 1500

let authGranted = null
let lastNotifyAt = 0

function getBridge() {
	if (jsBridge && typeof jsBridge === 'object') return jsBridge
	if (typeof window !== 'undefined' && window.jsBridge) return window.jsBridge
	return null
}

export function isYimenApp() {
	const bridge = getBridge()
	return !!(bridge && bridge.inApp)
}

function whenReady() {
	return new Promise((resolve) => {
		const bridge = getBridge()
		if (!bridge || !bridge.inApp) {
			resolve(null)
			return
		}
		if (typeof bridge.isReady === 'function' && bridge.isReady()) {
			resolve(bridge)
			return
		}
		if (typeof bridge.ready === 'function') {
			bridge.ready(() => resolve(bridge))
			return
		}
		resolve(bridge)
	})
}

/** 检查 / 申请通知权限。granted === false 时可引导去设置。 */
export async function requestNotifyAuth() {
	const bridge = await whenReady()
	if (!bridge?.notification?.requestAuth) {
		return { ok: false, reason: 'not-in-app' }
	}
	return new Promise((resolve) => {
		try {
			bridge.notification.requestAuth((granted) => {
				const ok = !!granted
				authGranted = ok
				resolve({ ok: true, granted: ok })
			})
		} catch (error) {
			resolve({ ok: false, reason: 'error', error })
		}
	})
}

/** 进程内只申请一次；已拒绝则后续直接跳过。 */
export async function ensureNotifyAuth() {
	if (!isYimenApp()) return false
	if (authGranted === true) return true
	if (authGranted === false) return false
	const result = await requestNotifyAuth()
	return !!(result.ok && result.granted)
}

/** 清除权限缓存，便于从设置页返回后重新检测。 */
export function resetNotifyAuthCache() {
	authGranted = null
}

/** 重新检测通知权限（不弹业务引导框）。 */
export async function refreshNotifyAuth() {
	if (!isYimenApp()) return false
	authGranted = null
	return ensureNotifyAuth()
}

/** 仅在之前未授权时重新检测（从设置返回后生效，避免已授权时反复打扰）。 */
export async function refreshNotifyAuthIfDenied() {
	if (!isYimenApp()) return false
	if (authGranted === true) return true
	authGranted = null
	return ensureNotifyAuth()
}

/**
 * 未授权时弹出引导，确认后跳转系统通知设置。
 * @returns {Promise<boolean>} 当前是否已授权
 */
export async function ensureNotifyAuthWithSettingsPrompt() {
	if (!isYimenApp()) return false
	const granted = await ensureNotifyAuth()
	if (granted) return true
	return new Promise((resolve) => {
		uni.showModal({
			title: '开启消息通知',
			content: '通知权限未开启，无法及时提醒新消息。请在系统设置中打开本应用的通知。',
			confirmText: '去设置',
			cancelText: '暂不',
			success: (res) => {
				if (res.confirm) openNotifySettings()
				resolve(false)
			},
			fail: () => resolve(false)
		})
	})
}

/**
 * 发送本地通知。
 * @param {{ title: string, content: string, interval?: number }} options
 */
export async function notifyLocal(options = {}) {
	const bridge = await whenReady()
	if (!bridge?.notification?.notify) {
		return { ok: false, reason: 'not-in-app' }
	}
	const title = String(options.title || '').trim()
	const content = String(options.content || '').trim()
	if (!title || !content) {
		return { ok: false, reason: 'invalid-payload' }
	}
	const payload = {
		title,
		content,
		interval: Number.isFinite(options.interval) ? options.interval : 0
	}
	// 不传 url / badge：点通知只回到 App；避免误弹未读汇总通知
	return new Promise((resolve) => {
		try {
			bridge.notification.notify(payload, (succ, data) => {
				resolve({ ok: !!succ, data })
			})
		} catch (error) {
			resolve({ ok: false, reason: 'error', error })
		}
	})
}

export async function openNotifySettings() {
	const bridge = await whenReady()
	if (!bridge) return { ok: false, reason: 'not-in-app' }
	try {
		if (typeof bridge.openSetting === 'function') {
			bridge.openSetting(6)
			return { ok: true }
		}
		if (typeof bridge.appSettings === 'function') {
			bridge.appSettings()
			return { ok: true }
		}
		return { ok: false, reason: 'unsupported' }
	} catch (error) {
		return { ok: false, reason: 'error', error }
	}
}

/**
 * 后台新消息本地通知（带权限检查与节流）。
 * @param {{ title: string, content: string, force?: boolean }} options
 */
export async function notifyIncomingChat(options = {}) {
	if (!isYimenApp()) return { ok: false, reason: 'not-in-app' }
	const granted = await ensureNotifyAuth()
	if (!granted) return { ok: false, reason: 'denied' }

	const now = Date.now()
	if (!options.force && now - lastNotifyAt < NOTIFY_THROTTLE_MS) {
		return { ok: false, reason: 'throttled' }
	}
	lastNotifyAt = now

	return notifyLocal({
		title: options.title,
		content: options.content,
		interval: 0
	})
}

/**
 * P1 冒烟：申请权限 → 立即发一条测试本地通知。
 */
export async function runYimenNotifyP1Smoke() {
	if (!isYimenApp()) {
		return { ok: false, reason: 'not-in-app' }
	}
	const auth = await requestNotifyAuth()
	if (!auth.ok) {
		return { ok: false, reason: auth.reason || 'auth-failed', auth }
	}
	if (auth.granted === false) {
		return { ok: false, reason: 'denied', auth }
	}
	const notify = await notifyLocal({
		title: '本地通知测试',
		content: '一门本地通知对接成功。若看到本条，说明 ym-jsbridge 可用。',
		interval: 0
	})
	return notify.ok ? { ok: true, auth, notify } : { ok: false, reason: 'notify-failed', auth, notify }
}
