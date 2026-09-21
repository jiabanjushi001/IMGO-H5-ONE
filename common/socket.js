import api from '@/common/config.js' // 接口Api，图片地址等等配置，可根据自身情况引入，也可以直接在下面url填入你的 webSocket连接地址
class socketIO {
	constructor(data, time, url) {
		this.socketTask = null
		this.is_open_socket = false //避免重复连接
		this.isConnecting = false
		this.manualClose = false
		this.url = url ? url : api.wssUrl //连接地址
		this.data = data ? data : null
		this.connectNum = 1 // 重连次数
		this.traderDetailIndex = 100 // traderDetailIndex ==2 重连
		this.accountStateIndex = 100 // accountStateIndex ==1 重连
		this.followFlake = false // followFlake == true 重连
		this.init=false;
		//心跳检测
		this.timeout = time ? time : 20000 //多少秒执行检测
		this.heartbeatInterval = null //检测服务器端是否还活着
		this.reconnectTimeOut = null //重连之后多久再次重连
		this.networkStatus=true
	}
	
	CALLBACK = (res) => {
	    if(res.isConnected && !this.manualClose){
	    	this.traderDetailIndex=2;
	    	this.connectSocketInit(this.data || {type:'ping'})
	    }
	}

	clearReconnectTimer() {
		if (this.reconnectTimeOut !== null) {
			clearTimeout(this.reconnectTimeOut)
			this.reconnectTimeOut = null
		}
	}

	handleSocketDisconnect(task) {
		// onError 和 onClose 可能为同一次断线先后触发，只处理当前连接一次。
		if (task !== this.socketTask) return
		this.socketTask = null
		this.isConnecting = false
		this.is_open_socket = false
		clearInterval(this.heartbeatInterval)
		this.heartbeatInterval = null
		this.clearReconnectTimer()
		if (this.manualClose) return

		if (this.connectNum < 5) {
			this.connectNum += 1
			this.traderDetailIndex = 2
			this.reconnect()
		} else {
			uni.$emit('connectError')
			this.networkStatus = false
			uni.offNetworkStatusChange(this.CALLBACK)
			uni.onNetworkStatusChange(this.CALLBACK)
			this.connectNum = 1
		}
	}

	// 进入这个页面的时候创建websocket连接【整个页面随时使用】
	connectSocketInit(data) {
		if (data !== undefined) this.data = data
		// 每次连接前刷新地址，避免误用已失效的本地备用站
		const latestUrl = (api.getWssUrl && api.getWssUrl()) || api.wssUrl
		if (latestUrl) this.url = latestUrl
		if (this.socketTask && [2, 3].includes(this.socketTask.readyState)) {
			this.handleSocketDisconnect(this.socketTask)
		}
		if (this.socketTask || this.isConnecting) return this.socketTask
		this.clearReconnectTimer()
		this.manualClose = false
		this.isConnecting = true
		console.info('[IMGO] WebSocket 连接中:', this.url)
		// #ifdef H5
		try {
			if (String(location.protocol || '').startsWith('file') || location.origin === 'null') {
				console.warn('[IMGO] 当前为 file:// / Origin:null。若服务端校验 Origin，wss 握手可能返回 403；一门离线包需服务端放行该 Origin，或用 http(s) 打开页面。')
			}
		} catch (e) {}
		// #endif
		let task
		try {
			task = uni.connectSocket({
				url: this.url,
				success: () => console.info("正准备建立websocket中...")
			})
		} catch (error) {
			console.error('WebSocket连接创建失败', error)
			this.handleSocketDisconnect(null)
			return null
		}
		if (!task) {
			this.handleSocketDisconnect(null)
			return null
		}
		this.socketTask = task
		task.onOpen(() => {
			if (this.socketTask !== task || this.manualClose) return
			this.isConnecting = false
			this.is_open_socket = true
			this.clearReconnectTimer()
			clearInterval(this.heartbeatInterval)
			this.connectNum = 1
			if(!this.networkStatus){
				// 连接成功后取消网络监听
				uni.offNetworkStatusChange(this.CALLBACK);
			}
			this.networkStatus=true;
			console.info("WebSocket连接正常！", this.url);
			uni.$emit('socketStatus',true);
			this.send(this.data)
			this.start();
			// 注：只有连接正常打开中 ，才能正常收到消息
			task.onMessage((e) => {
				if (this.socketTask !== task) return
				// 字符串转json
				let res = JSON.parse(e.data);
				if (res) {
					uni.$emit('getPositonsOrder', res);
				}
			});
		})
		if(!this.init){
			// 全局错误监听只注册一次；断线统一由 handleSocketDisconnect 去重。
			uni.onSocketError((res) => {
				console.info(res,'WebSocket连接打开失败，请检查！', this.url);
				if (this.socketTask) this.handleSocketDisconnect(this.socketTask)
			});
			this.init=true;
		}
		// 这里仅是事件监听【如果socket关闭了会执行】
		task.onClose((res) => {
			console.info("已经被关闭了-------", this.url, res || '')
			this.handleSocketDisconnect(task)
		})
		return task
	}
    // 主动关闭socket连接
	Close() {
		this.manualClose = true
		this.clearReconnectTimer()
		uni.offNetworkStatusChange(this.CALLBACK)
		this.networkStatus = true
		clearInterval(this.heartbeatInterval)
		this.heartbeatInterval = null
		const task = this.socketTask
		this.socketTask = null
		this.isConnecting = false
		this.is_open_socket = false
		if (!task) return
		task.close({
			success() {
				uni.showToast({
					title: 'SocketTask 关闭成功',
					icon: "none"
				});
			}
		});
	}
	//发送消息
	send(data) {
		// 注：只有连接正常打开中 ，才能正常成功发送消息
		if (this.socketTask && this.is_open_socket) {
			this.socketTask.send({
				data: JSON.stringify(data),
				async success() {
				},
			});
		}
	}
	
	// 检测状态
	checkStatus(){
		console.info("检查状态")
		if (this.socketTask && ![2,3].includes(this.socketTask.readyState)) return true
		if (this.socketTask) this.handleSocketDisconnect(this.socketTask)
		this.clearReconnectTimer()
		console.info("未链接！")
		return false
	}
	
	//开启心跳检测
	start() {
		this.heartbeatInterval = setInterval(() => {
			this.send({
				"type": "ping"
			});
		}, this.timeout)
	}
	//重新连接
	reconnect() {
		//停止发送心跳
		console.info('检查是否手动断开，并重新连接')
		clearInterval(this.heartbeatInterval)
		this.heartbeatInterval = null
		//如果不是人为关闭的话，进行重连
		if (!this.manualClose && !this.socketTask && !this.isConnecting && this.reconnectTimeOut === null &&
			(this.traderDetailIndex == 2 || this.accountStateIndex == 0 || this
			.followFlake)) {
			console.info("5秒后重新连接...")
			this.reconnectTimeOut = setTimeout(() => {
				this.reconnectTimeOut = null
				this.connectSocketInit(this.data);
			}, 5000)
		}
	}
	/**
	 * @description 将 scoket 数据进行过滤 
	 * @param {array} array
	 * @param {string} type 区分 弹窗 openposition 分为跟随和我的
	 */
	arrayFilter(array, type = 'normal', signalId = 0) {
		let arr1 = []
		let arr2 = []
		let obj = {
			arr1: [],
			arr2: []
		}
		arr1 = array.filter(v => v.flwsig == true)
		arr2 = array.filter(v => v.flwsig == false)
		if (type == 'normal') {
			if (signalId) {
				arr1 = array.filter(v => v.flwsig == true && v.sigtraderid == signalId)
				return arr1
			} else {
				return arr1.concat(arr2)
			}
		} else {
			if (signalId > 0) {
				arr1 = array.filter(v => v.flwsig == true && v.sigtraderid == signalId)
				obj.arr1 = arr1
			} else {
				obj.arr1 = arr1
			}
			obj.arr2 = arr2
			
			return obj
		}
	}
}
export default socketIO
