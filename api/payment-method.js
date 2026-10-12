import { apiUrl, postJsonRequest } from '@/utils/request.js'

export const paymentQrUploadUrl = `${apiUrl}/common/upload/uploadFile`

export default {
	get() { return postJsonRequest('/enterprise/paymentMethod/get', {}) },
	save({ payment_type, receipt_name, receipt_account = '', qr_file_id = 0 }) {
		return postJsonRequest('/enterprise/paymentMethod/save', { payment_type, receipt_name, receipt_account, qr_file_id })
	}
}
