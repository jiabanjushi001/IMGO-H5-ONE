/**
 * 本地静态资源路径（static / hybrid 等）。
 * 兼容：Nginx 站点根路径、子目录、一门 APP（file:// / fs://www/）离线包。
 */
export function assetUrl(path) {
	if (path == null || path === '') return path
	const value = String(path)
	if (/^(https?:|data:|blob:|fs:|file:)/i.test(value)) return value

	const relative = value.replace(/^\.\//, '').replace(/^\//, '')

	// #ifdef H5
	if (typeof window !== 'undefined') {
		const base = window.__IMGO_BASE__ || new URL('./', document.baseURI).href
		try {
			return new URL(relative, base).href
		} catch (error) {
			return relative
		}
	}
	// #endif

	return '/' + relative
}

export default assetUrl
