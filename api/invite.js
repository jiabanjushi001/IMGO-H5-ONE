import { postJsonRequest } from '@/utils/request.js'

export default {
	status() { return postJsonRequest('/enterprise/invite/status', {}) }
}
