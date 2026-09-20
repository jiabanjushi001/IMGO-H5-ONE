export function inviteShareUrl(code, serverUrl, location, packagedApp) {
	if (!/^[0-9]{6}$/.test(code)) return ''
	if (!packagedApp && /^https?:$/.test(location?.protocol || '')) {
		return `${location.origin}${location.pathname}#/pages/login/register?inviteCode=${code}`
	}
	try {
		const link = new URL(serverUrl)
		if (!['http:', 'https:'].includes(link.protocol)) return ''
		if (['localhost', '127.0.0.1', '0.0.0.0'].includes(link.hostname)) return ''
		const hashQuery = link.hash.split('?')[1] || ''
		const actualCode = new URLSearchParams(hashQuery || link.search).get('inviteCode')
		return actualCode === code ? link.href : ''
	} catch (error) {
		return ''
	}
}
