export function canUsePreparedVideo(userInfo) {
	if (!userInfo || typeof userInfo !== 'object') return false
	const userId = Number(userInfo.user_id || userInfo.id || 0)
	return userId === 1 || Number(userInfo.agent_mode || 0) === 1
}
