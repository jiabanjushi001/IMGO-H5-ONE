import { postJsonRequest } from '@/utils/request.js'

export default {
	status() { return postJsonRequest('/enterprise/wallet/status', {}) },
	withdraw({ amount, request_id }) { return postJsonRequest('/enterprise/wallet/withdraw', { amount, request_id }) },
	history({ page = 1, limit = 20 } = {}) { return postJsonRequest('/enterprise/wallet/history', { page, limit }) },
	entries({ page = 1, limit = 20 } = {}) { return postJsonRequest('/enterprise/wallet/entries', { page, limit }) }
}
