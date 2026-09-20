import { postJsonRequest } from '@/utils/request.js'

export default {
	status() { return postJsonRequest('/enterprise/checkin/status', {}) },
	submit() { return postJsonRequest('/enterprise/checkin/submit', {}) }
}
