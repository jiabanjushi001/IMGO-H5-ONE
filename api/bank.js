import { postJsonRequest } from '@/utils/request.js'

export default {
	get() { return postJsonRequest('/enterprise/bank/get', {}) },
	save({ name, card_number, bank_name, branch_name }) { return postJsonRequest('/enterprise/bank/save', { name, card_number, bank_name, branch_name }) }
}
