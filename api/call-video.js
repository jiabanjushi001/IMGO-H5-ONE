import { postJsonRequest } from '@/utils/request.js'

export function getCallVideos(params = {}) {
	return postJsonRequest('/enterprise/callVideo/index', { page: 1, limit: 100, ...params })
}
