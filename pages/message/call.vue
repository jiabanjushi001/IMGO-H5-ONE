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
			this.html='/hybrid/html/index.html?id='+userInfo.value.user_id+'&status='+option.status+'&calling='+calling+'&name='+option.name+'&target_id='+option.id+'&type='+option.type+'&platform='+platform+'&avatar='+option.avatar+'&stun='+stun;
			this.main_id=option.msg_id;
			this.type= option.type;
			this.status = option.status
			this.contact={
				id:option.id,
				displayName:option.name,
				avatar:option.avatar
			}
			// #ifdef H5
			setTimeout(() => {
			  this.Frames = document.getElementsByTagName('iframe');
			  this.mainFrame = this.Frames[0].contentWindow;
			  this.postMsg = this.h5sendH5
			}, 500)
			
			window.onmessage = (e) => {
				this.msgCallback(e)
			}
			//#endif
			
			uni.$off('webrtcConn');
			// 接收websocket传输过来的消息,并发送给webview
			uni.$on('webrtcConn',(e)=>{
				 if(e.fromUser.id==userInfo.value.user_id){
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
				
			})
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
			  this.mainFrame.postMessage(params, '*')
			},
			msgCallback(e){
				let iceCandidate = '';
				let msg_id='';
				if(this.wsData){
					msg_id=this.wsData.msg_id ?? '';
				}
				let api=true;
				switch(e.data['event']){
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
							title: '请检查音视频',
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
					}).then((res)=>{
						if(e.data['event']=='calling'){
							this.wsData=res.data;
						}
						if(res.data.extends.code==907){
							uni.showToast({
								title:'对方不在线',
								icon:'none'
							})
							this.closeCall();
						}
					});
				}
				
			},
			//关闭通话页面
			closeCall(){
				const innerAudioContext = uni.createInnerAudioContext();
				innerAudioContext.autoplay = true;
				innerAudioContext.src = config.apiUrl+'/static/voice/guaduan.mp3';
				innerAudioContext.onStop((res) => {
					this.$forceUpdate()
				})
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
