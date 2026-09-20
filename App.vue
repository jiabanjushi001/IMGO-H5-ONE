<script>
	// 引入vue3 getCurrentInstance 
	import { getCurrentInstance } from 'vue'
	import pinia from '@/store/index' 
	import { useMsgStore } from '@/store/message';
	import { useloginStore } from '@/store/login';
	import permision from "@/utils/permission.js"
	import config from "@/common/config";

	// #ifdef H5
		import VConsole from 'vconsole';
	// #endif
	let vConsole = null; //移动H5调试器
	const msgStore = useMsgStore(pinia)
	const userStore = useloginStore(pinia)
	// #ifdef APP-PLUS
		import appUpdate from '@/common/appUpdate.js';
		// 安卓设备引入保活插件
		const keepAlive = uni.requireNativePlugin('Ba-KeepAlive');
	// #endif
	export default {
		data() {
			return {
				globalSocketMessageHandler: null,
				globalConnectErrorHandler: null
			}
		},
		onLaunch: function() {
			// 初始化APP设置
			let setting=uni.getStorageSync('appSetting') ?? '';
			if(!setting){
				userStore.setAppSetting({
					voiceStatus:true,
					vibrateStatus:false,
					circleAvatar:false
				});
			}else{
				userStore.setAppSetting(setting);
			}
			// 获取全局配置
			userStore.getGlobalConfig();
		    uni.getSystemInfo({
		        success: function(e) {
					let paddingB=0;
		        	// 获取 appContext  上下文
					const {appContext} = getCurrentInstance();
					appContext.config.globalProperties.StatusBar = e.statusBarHeight;
		            // #ifndef MP
		            if (e.platform == 'android') {
		                appContext.config.globalProperties.CustomBar = e.statusBarHeight + 50;
		            } else {
		                appContext.config.globalProperties.CustomBar = e.statusBarHeight + 45;
		            };
		            // #endif
		            // #ifdef MP-WEIXIN
		            
		            let custom = wx.getMenuButtonBoundingClientRect();
		            appContext.config.globalProperties.Custom = custom;
		            appContext.config.globalProperties.CustomBar = custom.bottom + custom.top - e.statusBarHeight;
		            // #endif     
					   
		            // #ifdef MP-ALIPAY
		            appContext.config.globalProperties.CustomBar = e.statusBarHeight + e.titleBarHeight;
		            // #endif
					
					// #ifdef MP
					try {
						
					    var res = uni.getSystemInfoSync();
						res.model = res.model.replace(' ', '');
						res.model = res.model.toLowerCase();
						var res1  = res.model.indexOf('iphonex');
						if(res1 > 5){res1 = -1;}
						var res2   = res.model.indexOf('iphone1');
						if(res2 > 5){res2 = -1;}
						if(res1 != -1 || res2 != -1){
							paddingB = uni.upx2px(50);
						}
						
					} catch (e){return null;}
					// #endif
					// #ifdef H5
						paddingB = uni.upx2px(100);
					// #endif
					// 获取设备底部高度
					appContext.config.globalProperties.inlineTools=paddingB;
					// 设置全局底部导航栏高度
					appContext.config.globalProperties.navBarHeight=uni.upx2px(100);
		        }
		    })
			
			//启用H5调试模式
			// #ifdef H5
			this.loadVConsole();
			// #endif
			
			// #ifdef APP-PLUS
			// 检查应用更新
			appUpdate();
			
			if (plus.os.name != "iOS") {
				this.register();
			}
			// 只有app才有推送权限
			uni.onPushMessage((res) => {
				console.info("点击通知栏消息：",res) 
				let data=res.data;
				if(res.type=='click'){
					let playload=data.payload;
					let toUser=playload.toContactId ? playload.toContactId : '';
					// 如果是音视频通话则不
					if(playload.type=='webrtc'){
						return;
					}
					uni.navigateTo({
						url:"/pages/message/chat?id=" + toUser
					})
				}
			})
			// #endif
		}, 
		onUnload() {
			this.scoketClose()
			if (this.globalSocketMessageHandler) {
				uni.$off('getPositonsOrder', this.globalSocketMessageHandler)
				this.globalSocketMessageHandler = null
			}
			if (this.globalConnectErrorHandler) {
				uni.$off('connectError', this.globalConnectErrorHandler)
				this.globalConnectErrorHandler = null
			}
			this.socketIo.traderDetailIndex = 100 // 初始化 tabIndex 
		},
		onShow: function() {
			// #ifdef APP-PLUS
			// 判断平台,如果是安卓就检测保活插件
			if (plus.os.name != "iOS") {
				this.isRunning();
			 }
			// #endif
			
			if(!this.socketIo.checkStatus()){
				console.log('ws断线了，重新链接！');
				uni.$emit('socketStatus',false);
				this.getWebsocketData();
			}
			this.appStatus=true;
			console.log('App Show')
		},
		
		methods:{
			// 开启调试模式
			loadVConsole() { //初始化vConsole，用于H5调试用
				if (config.isVConsole) { //开启调试时
					let systemInfo = uni.getSystemInfoSync();
					if (!(systemInfo.uniPlatform == 'app' || systemInfo.uniPlatform == 'web')) { // 当为app或者H5时
						return;
					}
					vConsole = new VConsole({
						defaultPlugins: ['system', 'network', 'element', 'storage'],
						// 可以在此设定要默认加载的面板
						maxLogNumber: 1000,
						// disableLogScrolling: true,
						onReady: () => {
							console.log('vConsole: onReady');
							// 置顶最高层级
							var vcSwitch = document.getElementsByClassName('vc-switch')[0];
							vcSwitch.style.zIndex = '9999999999';
							var vcMask = document.getElementsByClassName('vc-mask')[0];
							vcMask.style.zIndex = '9999999999';
							var vcPanel = document.getElementsByClassName('vc-panel')[0];
							vcPanel.style.zIndex = '9999999999';
						},
						onClearLog: () => {
							console.log('vConsole: onClearLog');
						}
					});
				}
			},
			destroyVConsole() {
				// 结束调试后，可移除掉
				vConsole.destroy();
			},
			isRunning() { //是否正在运行
				keepAlive.isRunning((res) => {
					if(!res.isRunning){
						this.restart();
					}else{
						console.info('保活组件运行中');
					}
				});
			},
			restart() { //重启
				keepAlive.restart((res) => {
					console.info('重启成功！');
				});
			},
			register() { //注册
				keepAlive.register({
					channelId: 'Ba-KeepAlive',
					channelName: "Ba-KeepAlive",
					title: "消息服务正在运行中",   //可以修改为自己想展示的文字
					content: '用于接收消息的常驻通知，请保留！',  //可以修改为自己想展示的文字
				},
				(res) => {
					console.log('保活注册成功');
				});
			},
			scoketClose() {
				this.socketIo.connectNum = 1
				const data = {
					type: "close"
				};
				this.socketIo.send(data); // 这是给后端发送特定数据 关闭推送
				this.socketIo.Close(); // 主动 关闭连接 ， 不会重连
			},
			getWebsocketData() {
				// 获取用户信息
				var userInfo = uni.getStorageSync('userInfo');
				var _this = this;
				// 要发送的数据包
				const data = {
					type: "ping"
				};
				if (this.globalSocketMessageHandler) {
					uni.$off('getPositonsOrder', this.globalSocketMessageHandler)
				}
				// 接收数据,全局监听
				this.globalSocketMessageHandler = (res) => {
					var userInfo = uni.getStorageSync('userInfo');
					let data=res.data;
					this.connect = true;
					switch (res['type']) {
						// 服务端ping客户端
						case 'ping':
							this.socketIo.send({
								type: "pong"
							});
							break;
						case 'pong':
							userStore.$patch({
							  multiport: res.multiport
							})
							break;
							// 登录 更新用户列表
						case 'init':
							// 设置全局clientId
							uni.setStorageSync('client_id', res.client_id);
							if(userInfo){
								this.$api.LoginApi.bindUid({
									client_id: res['client_id'],
									user_id: userInfo.user_id,
									cid:uni.getStorageSync('cid')
								}).then(e => {
									this.socketIo.send({
										type: "bindUid",
										user_id: userInfo.user_id,
										token:uni.getStorageSync("authToken")
									});
								}).catch(error => {})
							}
							
							break;
						//上线、下线通知
						case "isOnline":
						  msgStore.updateContacts({
						  	id: data.id,
						  	is_online:data.is_online
						  });
						  // 如果是下线,并且和通话的是用一个人,就将通话锁定关闭
						  if(!data.is_online && msgStore.webrtcLock==data.id){
							  msgStore.webrtcLock=false;
						  }
						  break;
						case "offline":
						   let clientId=uni.getStorageSync('client_id');
						   // 如果开启了多设备同时登录，则不走后面的逻辑
							 if(userStore.globalConfig.sysInfo.multipleLogin){
							   break;
							 }else if(data.id==userInfo.user_id && data.client_id!=clientId && data.isMobile){
								uni.showToast({
									title: "您的账号在其他设备登录，已被迫下线！",
									icon: "none",
									duration: 2500
								})
								userStore.logout();
						    }
							break;
						case "updateConfig":
						   userStore.setGlobalConfig(data);
							break;
						case 'simple':
						case 'group':
							if (data.toContactId === 'admin_notice') {
								uni.$emit('adminNoticePublished')
							}
							// 只要不是自己发的,才可以播放声音
							if (data.fromUser.id != userInfo.user_id) {
								const contact = msgStore.getContact(data.toContactId,data);
								// 如果开启了声音才播放
								if (data.toContactId=='system' || contact.is_notice == 1) {
								  this.playSound();
								}
							}
							this.appendMessage(res);
							break;
						case "setChatTop":
							msgStore.updateContacts({
								id: data.id,
								is_top: data.is_top
							});
							break;
						case "setIsNotice":
							msgStore.updateContacts({
								id: data.id,
								is_notice: data.is_notice
							});
							break;
						// 新增加了群聊
						case "addGroup":
						    msgStore.appendContacts(data);
						    this.$api.LoginApi.bindGroup({ client_id: uni.getStorageSync('client_id'), group_id: data.id });
						    break;
							// 设置群管理员
						case "setManager":
						case "addGroupUser":
						case "removeUser":
							if(res['type']=='removeUser' && data.user_id == userInfo.user_id){
								msgStore.deleteContacts({
								    id: data.group_id
								})
							}else{
								msgStore.updateContacts({
									id: data.group_id,
									avatar: data.avatar
								});
								uni.$emit('updateGroup',res);
							}
								
							break;
						// 修改群组名称
						case "editGroupName":
						    msgStore.updateContacts({
							    id: data.id,
							    displayName: data.displayName
						    });
						    break;
						case "editGroupAvatar":
							msgStore.updateContacts({ id: data.group_id, avatar: data.avatar });
							uni.$emit('updateGroup', res);
							break;
						case "removeGroup":
						    msgStore.deleteContacts({
							    id: data.group_id
						    })
						    break;
						// 发布公告
						case "setNotice":
						  msgStore.updateContacts({
							id: data.group_id,
							notice: data.notice
						  });
						  uni.$emit('groupNoticeChanged', data);
						  break;
						  // 群聊设置
						case "groupSetting":
						  msgStore.updateContacts({
							id: data.group_id,
							setting: data.setting
						  });
						  break;
						case "appendContact":
						  msgStore.appendContacts(data);
						  break;
						case 'webrtc':
							let platform='h5';
							//#ifdef H5
								platform='h5';
							//#endif
							//#ifdef APP-PLUS
								platform= 'app';
							//#endif
							if(data.fromUser.id==userInfo.user_id){
								// 挂断的情况下解锁webrtc
								if([902,903,905,906,907].includes(parseInt(data.extends.code))){
								  msgStore.webrtcLock=false;
								}
								// 如果是当前设备发出的消息则不处理
								if(data.extends.isMobile==1 || data.extends.event=='calling'){
									if(data.extends.event=='calling'){
										this.appendMessage(res);
									}
								  return;
								}
								
							}
							// 如果是多端在线,要将在通话中的用户锁定
							if(data.extends.event == 'offer' || data.extends.event == 'answer' ){
								msgStore.webrtcLock=true;
							}else if(data.extends.event == 'hangup'){
								msgStore.webrtcLock=false;
							}
						    if(data.extends.event == 'calling'){
								this.appendMessage(res);
								const allroutes = getCurrentPages();
								const cureentRoute = allroutes[allroutes.length - 1].route;
								// 如果当前已经在通话中,通知对方忙线中
								if (cureentRoute == 'pages/message/call') {
										this.$api.msgApi.sendToMsg({
											toContactId:data.fromUser.user_id,
											type:data.extends.type,
											event:'busy',
											status:data.extends.status,
											code:907,
											id:data.id,
											msg_id:data.msg_id,
										})
								}else{
									
									// 小程序不支持音视频聊天
									//#ifdef APP-PLUS || H5
									msgStore.webrtcLock=data.fromUser.user_id;
									uni.navigateTo({
									  url: '/pages/message/call?msg_id='+data.id+'&type='+data.extends.type+'&status='+data.extends.status+'&id='+data.fromUser.user_id+'&name='+data.fromUser.realname+'&avatar='+encodeURI(data.fromUser.avatar)
									})
									//#endif
								}
								
							   
						    }else{
								uni.$emit('webrtcConn',data);
							}
						   break;
						default:
							break;
					}
				}
				uni.$on('getPositonsOrder', this.globalSocketMessageHandler)
				// 错误时做些什么
				if (this.globalConnectErrorHandler) {
					uni.$off('connectError', this.globalConnectErrorHandler)
				}
				this.globalConnectErrorHandler = () => {
					this.connect = false
					this.scoketError = true
					uni.$emit('socketStatus',false);
				}
				uni.$on('connectError', this.globalConnectErrorHandler)
				// 先注册监听，再打开连接，避免错过服务端的初始化消息。
				this.socketIo.connectSocketInit(data)
			},
			appendMessage(res){
				let data=res.data;
				let userInfo = uni.getStorageSync('userInfo');
				// 确定接受人是谁
				let toUser = data.toContactId;
				if(data.toContactId == userInfo.user_id){
					toUser = data.toUser
				} 
				let contact=msgStore.getContact(toUser,data);
				if(data.toContactId=='system'){
					// 系统消息只需要把未读数增加一
					msgStore.$patch({
					  sysUnread: msgStore.sysUnread += 1,
					})
				}else{
					// 判断是否是自己,自己发的看接收人是谁，去更新接受人的信息
					let addUnread = 1;
					// 自己发送的不需要加
					if(data.fromUser.id==userInfo.user_id){
						addUnread = 0;
					}
					let at=0;
					// 如果at参数包含了我自己，就要增加@的数量
					if(data.at.includes(userInfo.user_id)){
						at=1;
					}
					if (contact) {
						contact.lastContent = data.content;
						contact.lastSendTime = data.sendTime*1000;
						contact.type = data.type;
						contact.unread += addUnread;
						contact.is_at += at;
						// 更新联系人信息
						msgStore.updateContacts(contact);
					} else {
						let newContact = {
							id: data.toContactId,
							displayName: data.fromUser.displayName,
							avatar: data.fromUser.avatar,
							lastContent: data.content,
							lastSendTime: data.sendTime*1000,
							is_group: data.is_group,
							unread: addUnread,
							is_top: 0,
							dep_id: 0,
							is_at:at
						}
						msgStore.appendContacts(newContact)
					}
					msgStore.catchSocketAction(res);
				}
				// #ifdef APP-PLUS
					let appStatus=this.appStatus;
					// 如果app在后台运行,不是自己发的，并且开启了通知，就创建通知栏
					if(!appStatus && data.fromUser.id!=userInfo.user_id && contact.is_notice==1){
						this.createPushMsg(data,contact);
					}
				// #endif
			},
			createPushMsg(data,contact){
				console.info("创建通知栏");
				var regex = /<[^>]+>/g; // 定义正则表达式，匹配所有的HTML标签
				let content=data.content.replace(regex, ''); // 将匹配到的HTML标签替换为空字符串
				if(data.type!='text'){
					let callVideo=data.extends.type ?? 0;
					content=this.$util.getMsgType(data.type,callVideo);
				}
				if(data.is_group==1){
					content=data.fromUser.displayName+':'+content;
				}
				let message={
					title:contact.displayName,
					content:content,
					payload:data,
				}
				
				let systemInfo = uni.getSystemInfoSync();
				// 判断平台,如果是安卓就去下载头像并展示头像
				if (systemInfo.platform === 'android') {
					uni.downloadFile({
						url: contact.avatar, 
						success: (res) => {
							if (res.statusCode === 200) {
								message.icon=res.tempFilePath
								uni.createPushMessage(message)
							}
						}
					});
				} else if (systemInfo.platform === 'ios') {
					 uni.createPushMessage(message)
				}
			},
			playSound() {
				let setting=uni.getStorageSync('appSetting') ?? '';
				if(setting.voiceStatus){
					let _this = this
					// _this.playing = true
					const innerAudioContext = uni.createInnerAudioContext();
					innerAudioContext.autoplay = true;
					innerAudioContext.src = config.apiUrl+'/static/voice/notify.mp3';
					innerAudioContext.onError((res) => {
						//如果音频没有正常播放
					})
					innerAudioContext.onStop((res) => {
						// _this.playing=false
						_this.$forceUpdate()
					})
				}
				if(setting.vibrateStatus){
					uni.vibrateLong({
						success: function () {
							console.log('手机震动');
						}
					});
				}
				
			}
		},
		onHide: function() {
			this.appStatus=false;
			console.log('App Hide')
		}
	}
</script>


<style lang="scss">
	@import url("static/css/iconfont.css");
	@import url("static/css/main.css");
	@import url("static/css/icon.css");
	@import url("static/css/reset.css");
	/* #ifdef H5 */
	@import url("static/css/h5-modern.css");
	/* #endif */
	/*每个页面公共css */
	@import '@/uni_modules/uni-scss/index.scss';
	/* #ifndef APP-NVUE */
	@import '@/static/customicons.css';
	// 设置整个项目的背景色
	page {
		background-color: #f5f5f5;
	}

	/* #endif */
	.example-info {
		font-size: 14px;
		color: #333;
		padding: 10px;
	}
</style>
