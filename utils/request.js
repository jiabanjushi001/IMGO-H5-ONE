import ajax from '@/uni_modules/u-ajax/js_sdk'
import config from '@/common/config.js'
import { normalizeAvatarData } from '@/utils/avatar.js'
// 创建请求实例
const axios = ajax.create({
	// 默认配置
	baseURL: config.apiUrl,
	timeout: 10000,
	// #ifdef H5
		withCredentials: false
	// #endif
})
// 注册请求拦截器
axios.interceptors.request.use(
	config => {
		// 在发送请求前做些什么
		return config
	},
	error => {
		uni.showToast({
			title: "网络超时！",
			icon: "none"
		})
		return error
	}
)

// 注册响应拦截器
axios.interceptors.response.use(
	response => {
		const res = normalizeAvatarData(response.data)
		let msg = res.msg
		if (msg) {
			if ([400, 401,402, 403, 404, 405, 502, 500].includes(res.code)) {
				uni.showToast({
					title: msg,
					icon: "none",
					duration: 2500
				})
			} else if (res.code == 0) {
				uni.showToast({
					title: msg,
					icon: "none",
					duration: 2500
				})
			} else if (res.code == -1) {
				uni.showToast({
					title: msg,
					icon: "none",
					duration: 2500,
					success() {
						setTimeout(() => {
							uni.reLaunch({
								url: "/pages/login/index"
							})
						}, 1000)
					}
				})

			}
		}
		return res
	},
	error => {
		// 对响应错误做点什么 
		let code = error.statusCode
		switch (code) {
			case 401:
				uni.showModal({
					title: '提示',
					content: '授权信息已经失效,请重新登录！',
					success: (res) => {
						if (res.confirm) {
							uni.reLaunch({
								url: "/pages/login/index"
							})
						}
					},
					fail: (res) => {
						console.log('fail')
					}
				});
				break
			case 500:
				uni.showToast({
					title: `服务器返回错误，请检查！`,
					icon: "none",
					duration: 2500
				})
				break;
			default:
				uni.showToast({
					title: error.errMsg,
					icon: "none",
					duration: 2500
				});
				break;
		}
		return error
	}
)

//****** 带 Json 的函数 请求头统一是application/json else formData
export const apiUrl = config.apiUrl;
export const getRequest = (url, params) => {
	let Authorization = uni.getStorageSync("authToken");
	return axios({
		method: 'get',
		url: `${url}`,
		params: params,
		header: {
			'Authorization': Authorization
		}
	});
};
export const postRequest = (url, params) => {
	return axios({
		method: 'post',
		url: `${url}`,
		header: {
			'Content-Type': 'application/x-www-form-urlencoded', //formdata 格式
			'Authorization':uni.getStorageSync("authToken"),
			"clientId": uni.getStorageSync('client_id'),
			"cid": uni.getStorageSync('cid')
		},
		data: params,
	});
};
export const postJsonRequest = (url, params) => {
	return axios({
		method: 'post',
		url: `${url}`,
		data: params,
		header: {
			'Content-Type': 'application/json', //json 格式
			'Authorization': uni.getStorageSync("authToken"),
			"clientId": uni.getStorageSync('client_id'),
			"cid": uni.getStorageSync('cid')
		}
	});
};
export default axios
