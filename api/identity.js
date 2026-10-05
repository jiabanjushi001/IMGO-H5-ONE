import { apiUrl, postJsonRequest } from '@/utils/request.js'

export const identityUploadUrl = `${apiUrl}/common/upload/uploadFile`

export default {
	get() { return postJsonRequest('/enterprise/identity/get', {}) },
	save({ real_name, id_number, front_file_id, back_file_id }) {
		return postJsonRequest('/enterprise/identity/save', { real_name, id_number, front_file_id, back_file_id })
	}
}
