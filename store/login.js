import { defineStore } from 'pinia'
import LoginApi from '@/api/login.js'
import { normalizeAvatarData } from '@/utils/avatar.js'

const createDefaultGlobalConfig = () => ({
	demon_mode: false,
	sysInfo: {
		logo: '',
		name: null,
		regtype: 0,
		regauth: 0,
		runMode: 1,
		multipleLogin: false,
		ipregion: 0,
		diyName: 0,
		noticePopup: '1',
		showScan: '1',
		showGroupQr: '1'
	},
	chatInfo: {
		online: 0,
		webrtc: 0,
		simpleChat: 0,
		stun: '',
		stunUser: '',
		stunPass: '',
		redoTime: 120,
		dbDelMsg: 0
	},
	compass: {
		status: 0,
		mode: 1,
		list: []
	},
	fileUpload: {
		size: 10
	}
})

const isConfigObject = (value) => value && typeof value === 'object' && !Array.isArray(value)

export const normalizeGlobalConfig = (value) => {
	const defaults = createDefaultGlobalConfig()
	const config = isConfigObject(value) ? value : {}

	return {
		...defaults,
		...config,
		sysInfo: {
			...defaults.sysInfo,
			...(isConfigObject(config.sysInfo) ? config.sysInfo : {})
		},
		chatInfo: {
			...defaults.chatInfo,
			...(isConfigObject(config.chatInfo) ? config.chatInfo : {})
		},
		compass: {
			...defaults.compass,
			...(isConfigObject(config.compass) ? config.compass : {})
		},
		fileUpload: {
			...defaults.fileUpload,
			...(isConfigObject(config.fileUpload) ? config.fileUpload : {})
		}
	}
}

export const useloginStore = defineStore({
  id: 'login', // id必填，且需要唯一
  state: () => {
    return {
	  userInfo:normalizeAvatarData(uni.getStorageSync('userInfo') ? uni.getStorageSync('userInfo') : {}),
	  globalConfig:normalizeGlobalConfig(uni.getStorageSync('globalConfig')),
	  appSetting:uni.getStorageSync('appSetting') ? uni.getStorageSync('appSetting') : [],
	  multiport:false
    }
  },
  // actions 用来修改 state
    actions: {
      login(userInfo) {
		  normalizeAvatarData(userInfo)
		  // 登陆后本地保存登录信息
      	uni.setStorageSync('userInfo', userInfo)
      	this.userInfo = userInfo;
		// 登陆后需要触发一次更新联系人
		uni.$emit('socketStatus',true);
		
      },
      logout() {
		 uni.removeStorageSync('userInfo');
		 uni.removeStorageSync("authToken");
		 uni.removeStorageSync('client_id');
		 this.userInfo = {}; 
		 uni.reLaunch({
		 	url: "/pages/login/index"
		 })
      },
	  getGlobalConfig(){
		  return LoginApi.getSystemInfo().then((res)=>{
			  if (res.code !== 0 || !res.data) return null
			  this.setGlobalConfig(res.data)
			  return this.globalConfig
		  })
	  },
	  setGlobalConfig(config){
		  const isSectionUpdate = isConfigObject(config) && typeof config.name === 'string' && isConfigObject(config.value)
		  const nextConfig = isSectionUpdate
			  ? { ...this.globalConfig, [config.name]: { ...this.globalConfig[config.name], ...config.value } }
			  : config
		  const normalizedConfig = normalizeGlobalConfig(nextConfig)
		  Object.assign(this.globalConfig, normalizedConfig)
		  uni.setStorageSync('globalConfig', normalizedConfig)
	  },
	  setAppSetting(setting){
		  uni.setStorageSync('appSetting', setting)
		  this.appSetting = setting
	  }
    }
})
