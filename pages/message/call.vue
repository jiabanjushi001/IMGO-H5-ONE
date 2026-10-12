<template>
  <view class="">
    <web-view @message="message" :src="html"></web-view>
  </view>
</template>

<script>
	import permision from "@/utils/permission.js"
	import { storeToRefs } from 'pinia';
	import { useloginStore } from '@/store/login';
	import pinia from '@/store/index'
	import config from "@/common/config";
	import { assetUrl } from '@/utils/asset-url.js'
	const userStore = useloginStore(pinia)
	const {userInfo} = storeToRefs(userStore);

	export default {
		data() {
			return {
				html: '',
				postMsg:null,
				webview:null,
				Frames: null,
				mainFrame: null,
				contact:null,
				type:false,
				wsData:null,
				main_id:null,
				videoMode:'',
				videoFileId:0,
				videoUrl:'',
				isClosing:false,
				frameReadyTimer:null,
				webrtcHandler:null,
				windowMessageHandler:null,
				hangupAudio:null,
				globalConfig:userStore.globalConfig
			}
		},
		onReady() {
			// #ifdef APP-PLUS
				var currentWebview = this.$scope.$getAppWebview() //获取当前页面的webview对象
				var that=this;
				setTimeout(function() {
					that.webview = currentWebview.children()[0]
				}, 600); //如果是页面初始化调用时，需要延时一下
				this.postMsg = this.appsendH5
			// #endif
		},
		onLoad: function (option) {
			this.videoMode = option.type == 1 && option.video_mode === 'recorded' ? 'recorded' : (option.type == 1 ? 'live' : '')
			this.videoFileId = Number(option.video_file_id || 0)
			this.videoUrl = option.video_url || ''
			let platform='h5';
			let calling=0;
			//#ifdef H5
			platform='h5';
			calling=1;
			//#endif
			//#ifdef APP-PLUS
			platform= 'app';
			// 如果是app需要用户授权
			this.checkAuth(option).then((e)=>{
				// 如果是拨打方拿到权限之后发起通话
				if(option.status==1){
					setTimeout(() => {
						this.postMsg({event:'calling'});
					}, 800)
				}
			});
			//#endif
			let stun=encodeURIComponent(JSON.stringify({
				stun:this.globalConfig.chatInfo.stun ?? '',
				stunUser:this.globalConfig.chatInfo.stunUser ?? '',
				stunPass:this.globalConfig.chatInfo.stunPass ?? '',
			}));
			this.html=assetUrl('hybrid/html/index.html')+'?id='+userInfo.value.user_id+'&status='+option.status+'&calling='+calling+'&name='+encodeURIComponent(option.name || '')+'&target_id='+option.id+'&type='+option.type+'&platform='+platform+'&avatar='+encodeURIComponent(option.avatar || '')+'&token='+encodeURIComponent(uni.getStorageSync('authToken') || '')+'&video_mode='+this.videoMode+'&video_file_id='+this.videoFileId+'&video_url='+encodeURIComponent(this.videoUrl)+'&stun='+stun;
			this.main_id=option.msg_id;
			this.type= option.type;
			this.status = option.status
			this.contact={
				id:option.id,
				displayName:option.name,
				avatar:option.avatar
			}
			// #ifdef H5
			this.frameReadyTimer = setTimeout(() => {
			  this.Frames = document.getElementsByTagName('iframe');
			  this.mainFrame = this.Frames[0] && this.Frames[0].contentWindow;
			  if (this.mainFrame) this.postMsg = this.h5sendH5
			}, 500)
			
			this.windowMessageHandler = (e) => {
				const frame = Array.from(document.getElementsByTagName('iframe')).find(item => item.contentWindow === e.source)
				if (!frame) return
				this.mainFrame = e.source
				this.postMsg = this.h5sendH5
				this.msgCallback(e)
			}
			window.addEventListener('message', this.windowMessageHandler)
			//#endif
			
			uni.$off('webrtcConn');
			// 接收websocket传输过来的消息,并发送给webview
			this.webrtcHandler = (e)=>{
				 if(e.extends && e.extends.video_mode){
					this.videoMode=e.extends.video_mode;
					this.videoFileId=Number(e.extends.video_file_id || this.videoFileId || 0);
				 }
				 if(e.fromUser.id==userInfo.value.user_id){
					// The server echoes RTC control events back to the sending account so
					// another logged-in device can stop its call UI.  Events marked as
					// isMobile=1 originated from this H5 client and must not close the
					// page that just accepted the call.
					if(Number(e.extends.isMobile)===1){
						return;
					}
					if(e.extends.event=="otherOpt"){
						this.wsData=null;
						this.main_id=null;
						this.postMsg({event:'hangup'},'*');
					}
					return;
				  }
				// 如果msgID不相同,表示在忙线中
				if(this.main_id && this.main_id!=e.id){
					this.$api.msgApi.sendToMsg({
						toContactId:e.fromUser.user_id,
						type:e.extends.type,
						event:'busy',
						status:e.extends.status,
						code:907,
						id:e.id,
						msg_id:e.msg_id,
					})
					return;
				}else{
					this.wsData=e;
					if(this.main_id && this.main_id==e.id){
						this.postMsg(JSON.parse(JSON.stringify(e.extends)),'*');
					}
				}
				
			}
			uni.$on('webrtcConn', this.webrtcHandler)
		},
			onUnload() {
			this.teardownCallPage()
		},
		methods: {
			async checkAuth(option){
				// #ifdef APP-PLUS
					let record=await this.requestPermission('record');
					if(!record){
						return false;
					}
					if(option.type==1){
						let camera=await this.requestPermission('camera');
						if(!camera){
							return false;
						}
					}
				// #endif
			},
			//音视频权限获取
			async requestPermission(auth) {
				let isIos=false;
			    // #ifdef APP-PLUS
					isIos = (plus.os.name == "iOS")
			    // #endif
				let andriodAuth='';
				if(auth=='record'){
					andriodAuth='android.permission.RECORD_AUDIO';
				}else if(auth=='camera'){
					andriodAuth='android.permission.CAMERA';
				}
				if(isIos){
					// let iosRes = await permision.judgeIosPermission(auth);
					return true;
				}else{
					let andRes = await permision.requestAndroidPermission(andriodAuth);
					return andRes==1 ? true : false;
				}
			},
			// 接收webview传输过来的消息
			message(event) {
				 const msg = {
				   data: event.detail.data[0]
				 }
				 this.msgCallback(msg)
			},
			// app端向webview传输消息
			appsendH5(params) {
			if (params.iceCandidate && params.iceCandidate.length>0){
				params.iceCandidate = JSON.parse(params.iceCandidate)
				}
			  this.webview.evalJS("getUniAppMessage('" + JSON.stringify(params) + "')")
			},
			// h5端向webview传输消息
			h5sendH5(params) {
			  if (this.mainFrame && !this.mainFrame.closed) {
				this.mainFrame.postMessage(params, '*')
			  }
			},
			stopEmbeddedCall() {
				if (typeof this.postMsg !== 'function') return
				try {
					this.postMsg({ event: 'forceClose' }, '*')
				} catch (error) {
					console.warn('stop embedded call failed', error)
				}
			},
			teardownCallPage() {
				if (this.frameReadyTimer) {
					clearTimeout(this.frameReadyTimer)
					this.frameReadyTimer = null
				}
				this.stopEmbeddedCall()
				if (this.webrtcHandler) {
					uni.$off('webrtcConn', this.webrtcHandler)
					this.webrtcHandler = null
				}
				// #ifdef H5
				if (this.windowMessageHandler) {
					window.removeEventListener('message', this.windowMessageHandler)
					this.windowMessageHandler = null
				}
				// #endif
				this.mainFrame = null
				this.Frames = null
			},
			msgCallback(e){
				const data = e && e.data
				const embeddedEvents = ['calling', 'hangup', 'iceCandidate', 'mediaDevices', 'acceptRtc', 'offer', 'answer']
				if (!data || typeof data !== 'object' || !embeddedEvents.includes(data.event)) return
				let iceCandidate = '';
				let msg_id='';
				if(this.wsData){
					msg_id=this.wsData.msg_id ?? '';
				}
				let api=true;
				switch(data.event){
					case 'hangup':
						this.closeCall();
						if(e.data['code']==907){
							uni.showToast({
								title: '对方忙~~',
								icon:'none'
							})
						}
						if(!e.data.isbtn){
							api=false;
						}
						break;
					case 'iceCandidate':
						console.log('监听同步ice')
						let niceCandidate = {}
						niceCandidate['candidate'] = e.data['iceCandidate']['candidate']
						niceCandidate['sdpMLineIndex'] = e.data['iceCandidate']['sdpMLineIndex']
						niceCandidate['sdpMid'] = e.data['iceCandidate']['sdpMid']
						iceCandidate=JSON.stringify(niceCandidate)
						break;
						case "mediaDevices":
							api=false;
							uni.showToast({
								title: data.message || '无法打开摄像头或麦克风',
								icon:'none'
						})
						this.closeCall();
				}
				if(api){
					this.$api.msgApi.sendToMsg({
						id:this.main_id,
						msg_id:msg_id,
						toContactId:this.contact.id,
						type:this.type,
						event:e.data['event'],
						status:e.data['status'] ?? '',
						code:e.data['code'] ?? '',
						callTime:e.data['callTime'] ?? '',
						sdp:e.data['sdp'] ?? '',
						iceCandidate:iceCandidate,
							video_mode:this.videoMode,
							video_file_id:this.videoFileId,
							isMobile:1,
						}).then((res)=>{
						if (!res || Number(res.code) !== 0 || !res.data || Array.isArray(res.data)) {
							if (e.data['event'] === 'calling') this.closeCall()
							return
						}
						if(e.data['event']=='calling'){
							this.wsData=res.data;
						}
						if(res.data.extends && res.data.extends.code==907){
							this.stopEmbeddedCall();
							uni.showToast({
								title:'对方不在线',
								icon:'none'
							})
							this.closeCall();
						}
					});
				}
				
			},
			playHangupSound(){
				if (this.hangupAudio) return
				const audio = uni.createInnerAudioContext();
				this.hangupAudio = audio
				audio.autoplay = true;
				audio.loop = false;
				audio.src = config.apiUrl+'/static/voice/guaduan.mp3';
				const release = () => {
					if (this.hangupAudio !== audio) return
					this.hangupAudio = null
					if (typeof audio.destroy === 'function') audio.destroy()
				}
				if (typeof audio.onEnded === 'function') audio.onEnded(release)
				if (typeof audio.onError === 'function') audio.onError(release)
			},
			//关闭通话页面
			closeCall(){
				if (this.isClosing) return
				this.isClosing = true
				this.stopEmbeddedCall()
				this.playHangupSound()
				let pages = getCurrentPages();
				if (pages.length > 1) {
				  uni.navigateBack();
				} else {
				  uni.reLaunch({
				    url: '/pages/index/index'
				  })
				}
			}

		}
	}
</script>

<style>
	.container {
		padding: 20px;
		font-size: 14px;
		line-height: 24px;
	}
</style>
