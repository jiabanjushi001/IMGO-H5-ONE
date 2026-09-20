<template name="im-input">
	<view id="more-oprate">
		<view class="im-footer bg-gray" :style="[{paddingBottom:paddingB+'px',minHeight:footerHeight+'px',bottom:InputBottom+'px'}]">
			<view class="im-menus f-28" :class="[recShow ? 'cuIcon-keyboard' : 'cuIcon-sound']"  hover-class="tap"  @tap="showRec"></view>
			<view class="im-flex1 im-msgarea">
				<input class="solid-bottom im-flex1 bg-white im-input"  type="text" v-model="inputMsg" :adjust-position="false" :focus="isFocus" maxlength="300" cursor-spacing="10"
				 @focus="InputFocus" @blur="InputBlur" @confirm="sendTextMsg" v-if="recShow==false" confirm-type="send" :confirm-hold="true"/>
				 <view class="toolBox im-flex1" v-if="recShow==true">
					<view class="recorder" :class="{active:isUseRecorder}" @touchstart.prevent="startRecorder"
						@touchend.prevent="endRecorder" @touchmove.prevent="moveRecorder" @touchcancel="cancelRecorder">
						{{isUseRecorder ? '松开结束' : '按住说话'}}
					</view>
				</view>
				<view class='im-flex im-space-between message-quote radius-6 im-align-items-center' v-if="quote">
					<view class='text-overflow'>{{quote.content}}</view>
					<view class="cuIcon-close" @tap="closeQuote"></view>
				</view>
				
			</view>
			<mumu-recorder ref='recorderRef' @success='handlerSuccess' @error='handlerError' v-if="isH5"></mumu-recorder>
			 <view class="im-flex im-justify-content-start im-align-items-center">
				 <view class="im-menus cuIcon-emoji f-28" hover-class="tap" @tap="showAppBox(1)"></view>
				 <view class="im-menus cuIcon-roundadd f-28 mr-10" hover-class="tap" v-if="!inputMsg" @tap="showAppBox(2)"></view>
				 <view v-if="inputMsg">
					 <button class="cu-btn bg-green shadow  mr-10"  @touchend.prevent="sendTextMsg">发送</button>
				 </view>
			 </view>
			 <!-- 表情窗口 -->
			 <view class="im-flex im-columns im-emoji-box" :style="[{height:boxHeight+'px'}]" v-if="appBox==1">
				 <scroll-view scroll-x class="bg-gray nav im-emoji-header" scroll-with-animation :scroll-left="scrollLeft">
					<view class="cu-item" :class="index==TabCur?'text-green':''" v-for="(item,index) in emojiList" :key="index" @tap="tabSelect" :data-id="index">
						<view :class="[item.icon]" class="f-20"></view>
					</view>
				 </scroll-view>
				 <scroll-view scroll-y class="bg-white im-emoji-body">
					 <view class="im-flex im-wrap im-justify-content-start im-align-items-center pd-10">
					 	<view v-for="(item,index) in currentEmojiList" class="im-emoji-item">
							 <image :src="item.src" style="width:44rpx" mode="widthFix" @tap="chooseEmoji(item)"></image>
						</view>
					 </view>
				 </scroll-view>
			 </view>
			 
			 <!-- 工具栏窗口 -->
			 <view class="im-flex im-app-box im-flex im-justify-content-start im-wrap im-align-content-start pd-20" :style="[{height:boxHeight+'px'}]" v-if="appBox==2">
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" @tap="chooseImg">
						 <view class="bg-white cuIcon-album f-24 radius-10  im-app-item-icon"></view>
						 <view class="mt-5">照片</view>
					 </view>
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" @tap="chooseVideo">
						 <view class="bg-white cuIcon-video f-24 radius-10 im-app-item-icon"></view>
						 <view class="mt-5">视频</view>
					 </view>
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" @tap="chooseFile">
						 <view class="bg-white cuIcon-file f-24 radius-10 im-app-item-icon"></view>
						 <view class="mt-5">文件</view>
					 </view>
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" v-if='!contact.is_group && (isH5 || isApp) &&  parseInt(globalConfig.chatInfo.webrtc)' @tap="calling(0)">
						 <view class="bg-white cuIcon-dianhua f-24 radius-10 im-app-item-icon"></view>
						 <view class="mt-5">语音通话</view>
					 </view>
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" v-if='!contact.is_group && (isH5 || isApp) && parseInt(globalConfig.chatInfo.webrtc)' @tap="calling(1)">
						 <view class="bg-white cuIcon-record f-24 radius-10 im-app-item-icon"></view>
						 <view class="mt-5">视频通话</view>
					 </view>
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" @tap="chooseProject" v-if="globalConfig.demon_mode">
						 <view class="bg-white cuIcon-apps f-24 radius-10 im-app-item-icon"></view>
						 <view class="mt-5">项目</view>
					 </view>
					 <view class="im-flex im-columns im-align-items-center mt-10 im-app-item" @tap="chooseProject" v-if="globalConfig.demon_mode">
						 <view class="bg-white cuIcon-friend f-24 radius-10 im-app-item-icon"></view>
						 <view class="mt-5">客户</view>
					 </view>
			 </view>
		</view>
		<view class="voice-model c-white radius-10 im-flex im-columns im-align-items-center pd-10" v-show="isUseRecorder">
			<view class="cuIcon-voicefill mt-15 mb-5 f-28" :class="[isCancel ? 'c-red' : '']"></view>
			<view :class="[isCancel ? 'c-red' : '']">
				{{isCancel ? '松开取消' : '正在录音...'}}
			</view>
		</view>
		
	</view>
</template>
<script>
import MumuRecorder from '@/uni_modules/mumu-recorder/components/mumu-recorder/mumu-recorder.vue'
import emoji from '@/utils/emoji.js'
import { useMsgStore } from '@/store/message';
import { useloginStore } from '@/store/login';
import pinia from '@/store/index'
const msgStore = useMsgStore(pinia);
const userStore = useloginStore(pinia);
export default {
	name  : "im-input",
	components: { MumuRecorder },
	props : { 
		boxStatus:{type:Number, default:0},
		contact:{type:Object, default:{}}
	},
	data() {
		return {
			InputBottom     :0,
			paddingB        : 0,
			footerHeight    : 50,
			boxHeight       :300,
			uploading       : false,
			recShow         : false,
			inputMsg        : "",
			recorderManager : null,
			recing          : false,
			recLength       : 1,
			recTimer        : null,
			tmpVoice        : '',
			isUseRecorder: false,
			playItemIndex: -1,
			currentAudio: '',
			mainHeight:0,
			isCancel:false,
			isH5:false,
			isApp:false,
			appBox:0,
			TabCur: 0,
			scrollLeft: 0,
			emojiList:[],
			currentEmojiList:[],
			isFocus:false,
			globalConfig:userStore.globalConfig,
			quote:''
		}
	},
	watch:{
		boxStatus(val){
			this.appBox=0;
			this.InputBottom=0;
		},
		appBox(val){
			// 如果没有打开应用盒子,输入框的高度设置为0;
			if(val==0 && !this.isFocus){
				this.InputBottom=0;
			}
		},
		InputBottom(val){
			this.$emit('setPad',val);
		}
	},
	created : function(){
		// 监听键盘高度
		//#ifdef APP-PLUS || MP-WEIXIN || MP-ALIPAY
		
		uni.onKeyboardHeightChange(res => {
			if(this.appBox==0 || res.height>0){
				this.InputBottom=res.height;
			}
		})
		//#endif
		this.emojiList=emoji;
		this.currentEmojiList=emoji[0]['children'];
		uni.getSystemInfo({
			success: res => {
				let windowHeight = res.windowHeight;
				this.mainHeight=windowHeight;
			}
		});
		// #ifdef H5
		this.isH5=true;
		// #endif
		// #ifdef APP-PLUS
		this.isApp=true;
		// #endif
		// #ifndef H5
		this.isH5=false;
		this.recorderManager = uni.getRecorderManager();
		this.recorderManager.onStop((res) => {
			this.tmpVoice    = res.tempFilePath;
			this.recing      = false;
			if(this.recLengt<1){
				// 检测录音是否大于1秒
				return this.checkRecorder(this.recLength);
			}else{
				// 发送语音消息
				this.sendVoiceMsg();
			}
		});
		this.recorderManager.onError(() => {
			uni.showToast({ title: '录音失败', icon: 'none' });
			this.recing = false;
		});
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
				let paddingB = uni.upx2px(50);
				this.paddingB =paddingB;
				this.footerHeight=55+paddingB;
			}
		} catch (e){return null;}
		// #endif
	},
	methods:{
		msgItem(){
			return {
				id:this.$util.getUuid(),
				sendTime:new Date().getTime(),
				status: "going",
				type: "text",
				content: "",
				is_read: 0,
				is_group: 0,
				file_id: 0,
				file_cate: 0,
				fileName: "",
				fileSize: 0,
				extends: null
			}
		},
		// 选择表情
		chooseEmoji(item){
			let tag='[!'+item.name+']'
			this.inputMsg+=tag;
		},
		tabSelect(e) {
			this.TabCur = e.currentTarget.dataset.id;
			this.scrollLeft = (e.currentTarget.dataset.id - 1) * 60;
			this.currentEmojiList=this.emojiList[this.TabCur]['children'];
		},
		showAppBox(val){
			if(this.appBox==val){
				this.appBox=0;
				this.InputBottom=0;
			}else{
				this.appBox=val;
				this.InputBottom=this.boxHeight;
				this.recShow  = false;
			}
		},
		// 展示录音界面
		showRec : function(){
			this.InputBottom=0;
			this.appBox=0;
			this.recShow  == true ? this.recShow  = false : this.recShow  = true
		},
		// 发送录音
		sendVoiceMsg: function(){
			if (this.tmpVoice == '') {
				uni.showToast({ title: "录制已取消", icon: "none" });
				return;
			}
			let res={
				localUrl:this.tmpVoice,
				duration:this.recLength
			}
			this.handlerSuccess(res);
			this.tmpVoice  = '';
			this.recLength = 0;
		},
		// 发送文本消息
		sendTextMsg: function () {
			if(this.appBox!=1){
				this.isFocus=true;
			}
			if (this.inputMsg == '') {return false;}
			let message={
				type:'text',
				content:this.inputMsg
			}
			
			// 如果有qute就是引用消息
			if(this.quote.msg_id){
				message.pid=this.quote.msg_id;
				message.extends=this.quote;
			}
			this.closeQuote();
			this.inputMsg = '';
			this.$emit('send',Object.assign(this.msgItem(), message),'');
		},
		// 选择图片
		chooseImg(){
			let message={
				type:'image',
				status:'going'
			};
			uni.chooseImage({
				count      : 9,
				sizeType   : ['compressed'],
				sourceType : ['album', 'camera'],
				success    : (res)=>{
					const tempFiles = res.tempFiles;
					tempFiles.forEach((item) => {
					    message.content=item.path;
					    message.fileName=item.name;
					    message.fileSize=item.size;
						this.$emit('send',Object.assign(this.msgItem(), message),item.path);
				    })
					
				}
			});
		},
		// 选择视频
		chooseVideo: function () {
			let message={
				type:'video',
				status:'going'
			};
			uni.chooseVideo({
				sourceType: ['camera', 'album'],
				success: (res) => {
					if(res.duration>60){
						return uni.showToast({
							title: '视频长度不能超过60秒',
							icon:'error'
						})
					}
					const tempFilePaths = res.tempFilePath;
					let arr={
						duration:res.duration,
						width:res.width,
						height:res.height
					};
					message.fileName=res.name;
					message.fileSize=res.size;
					message.extends=arr;
					message.content=tempFilePaths;
					this.$emit('send',Object.assign(this.msgItem(), message),tempFilePaths);
				}
			});
		},
		// 选择文件
		chooseFile:function(){
			let self=this;
			// #ifdef H5
				uni.chooseFile({
				    count: 5, //默认100
				    success: function (res) {
						self.appendFile(res);
				    }
				});
			// #endif
			// #ifdef MP
				wx.chooseMessageFile({
				    count: 5, //默认100
					success: function (res) {
						self.appendFile(res);
					}
				});
			// #endif
			
			// #ifdef APP-PLUS
				const lemonjkFileSelect = uni.requireNativePlugin('lemonjk-FileSelect');
				lemonjkFileSelect.showPicker({
				    pathScope: "/Download",  // 各属性配置见下方【showPicker可配置参数说明】
				    mimeType: "*/*",
				    utisType:"public.data",
				    multi:'yes',
				    }, result => {
				    // 未授权文件读取权限,可以提示用户未打开读取文件权限（仅安卓）
				    if(result.code==1001){
				        uni.showModal({
				            title:"需要文件访问权限",
				            content:"您还未授权本应用读取文件。为保证您可以正常上传文件，请在权限设置页面打开文件访问权限（不同手机厂商表述可能略有差异）请根据自己手机品牌设置",
				            confirmText:"去授权",
				            cancelText:"算了",
				            success(e) {
				                if(e.confirm){
				                    // 跳转到应用设置页
				                    lemonjkFileSelect.gotoSetting();        
				                }
				            }
				        })
				    }
					let type='file';
					let imageExts=['jpg','jpeg','png','bmp','gif'];
					let videoExts=['mp4','3gp','avi','m2v','mkv','mov'];
					result.files.forEach((item)=>{
						if(imageExts.includes(item.fileExtension)){
							type='image';
						}else if(videoExts.includes(item.fileExtension)){
							type='video';
						}else{
							type='file';
						}
						let filePath='file://'+item.filePath;
						const message={
							type:type,
							status:'going',
							fileName:item.FileName,
							fileSize:item.fileSize,
							content:filePath
						};
						this.$emit('send',Object.assign(this.msgItem(), message),filePath);
					})
				})
			// #endif
			
		},
		// 写入文件
		appendFile(res){
			const tempFiles=res.tempFiles;
			tempFiles.forEach((item) => {
				let path=item.path;
				// #ifdef APP-PLUS
					path='file://'+ item.path;
				// #endif
				let message={
					type:'file',
					status:'going',
					fileName:item.name,
					fileSize:item.size,
					content:path
				};
				// #ifdef H5
				let type=item.type;
				if(type.indexOf("image/")!=-1){
					message.type="image";
				}
				if(type.indexOf("video/")!=-1){
					message.type="video";
				}
				// #endif
				this.$emit('send',Object.assign(this.msgItem(), message),path);
			})
			
		},
		chooseProject(){
			return uni.showToast({
				title: '自己扩展呗~~'
			})
		},
		calling(is_video){
			// #ifdef MP
				return uni.showToast({
					title:'小程序暂不支持',
					icon:'none'
				})
			// #endif
			if(!parseInt(this.globalConfig.chatInfo.webrtc)){
				return uni.showToast({
					title:'未开启音视频通话',
					icon:'none'
				})
			}
			if(msgStore.webrtcLock){
				return uni.showToast({
					title:'其他终端正在通话中',
					icon:'none'
				})
			}
			let msg_id=this.$util.getUuid();
			uni.navigateTo({
			  url: '/pages/message/call?msg_id='+msg_id+'&type='+is_video+'&status=1&id='+this.contact.id+'&name='+this.contact.displayName+'&avatar='+encodeURI(this.contact.avatar)
			})
			
		},
		InputFocus(e) {
			this.isFocus=true;
			if(this.appBox>0){
				// 关闭表情盒子和更多盒子
				this.appBox=0;
			}
			// #ifdef H5
				this.InputBottom = 0;
			// #endif
			
		},
		InputBlur(e) {
			if(!this.appBox && !this.isFocus){
				this.InputBottom = 0;
			}
			this.isFocus=false;
		},
		// 开始录音
		startRecorder() {
			console.log('录音开始...')
			// #ifdef H5
			this.$refs.recorderRef.start()
			// #endif
			
			// #ifndef H5
			this.recorderManager.start({duration:60000, format:'mp3' });
			this.recLength  = 0;
			this.recTimer   = setInterval(()=>{this.recLength++;}, 1000);
			// #endif
			
			this.isUseRecorder = true
		},
		// 结束录音
		endRecorder() {
			console.log('录音结束')
			// #ifdef H5
			this.$refs.recorderRef.stop()
			// #endif
			
			// #ifndef H5
			this.recorderManager.stop();
			clearInterval(this.recTimer);
			// #endif
			
			this.isUseRecorder = false
		},
		//中断录音
		cancelRecorder(){
			this.endRecorder();
			this.isCancel=true;
		},
		// 移除录音按钮
		moveRecorder(e){
			let touch=e.touches[0];
			if(touch.clientY<this.mainHeight-80){
				this.isCancel=true;
			}else{
				this.isCancel=false;
			}
		},
		handlerSuccess(res) {
			this.checkRecorder(res.duration);
			if(this.isCancel){
				this.isCancel=false;
				return console.log('录音已取消');
			}
			let message={
				type:'voice',
				content:res.localUrl,
				fileName:this.$util.getUuid()+'.mp3',
				extends:{
					duration: res.duration
				}
			}
			this.$emit('send',Object.assign(this.msgItem(), message));
		},
		// 检测录音是否合格
		checkRecorder(duration){
			if(duration<1 || isNaN(duration) || !duration){
				this.recLength  = 0;
				this.tmpVoice  = '';
				this.recing  = false;
				this.isCancel=true;
				return uni.showToast({
					title: '录音时间太短',
					icon: 'error'
				})
			}
		},
		handlerError(code) {
			switch (code) {
				case '101':
					uni.showModal({
						content: '当前浏览器版本较低，请更换浏览器使用，推荐在微信中打开。'
					})
					break;
				case '201':
					uni.showModal({
						content: '麦克风权限被拒绝，请刷新页面后授权麦克风权限。'
					})
					break
				default:
					console.log('录音功能受限，请知晓！')
					break
			}
		},
		closeQuote(){
			this.quote='';
		},
		// 引用消息
		quoteMessage(quote){
			this.quote=quote;
		}
	}
}
</script>
<style lang="scss" scoped>
.im-footer{padding:0; width:100%; position:fixed; left:0; bottom:0;display:flex; flex-wrap:nowrap; overflow:hidden; box-shadow:1px 1px 6px #999999; align-items:flex-end;z-index:101}
.im-footer .items{width:auto; line-height:88rpx; flex-shrink:0; font-size:28rpx; color:#2B2E3D;}
.im-msgarea{padding:10rpx 10rpx 10rpx 0;}
.im-menus{width:80rpx; height:80rpx; flex-shrink:0; line-height:80rpx; text-align:center;}
.im-input{padding:10rpx 16rpx; border-radius:10rpx;margin:0 8rpx !important;height:72rpx}
.im-input input{width:100%; background:#F4F5F6; color:#2B2E3D; height:72rpx; line-height:72rpx; font-size:28rpx; margin-top:10rpx;}
.im-record{width:100%; position:fixed; left:0; bottom:0; background:#FFFFFF; padding:30px 0; padding-bottom:100rpx; z-index:11; box-shadow:1px 1px 6px #999999;}
.im-record-close{width:100rpx; height:100rpx; position:absolute; top:0px; right:0px; z-index:100; text-align:center; line-height:100rpx; color:#888888; font-size:38rpx !important;}
.im-record-txt{text-align:center; font-size:26rpx; line-height:30px; padding-bottom:10px; color:#CCCCCC;}
.im-record-btn{width:60px; height:60px; margin:0 auto; border:5px solid #F1F2F3; border-radius:100%; background:#00B26A;}
.im-recording{background:#FF0000; animation:fade linear 2s infinite;}
@keyframes fade{from{opacity:0.1;} 50%{opacity:1;} to{opacity:0.1;}}
.im-record-txt text{color:#00B26A; padding:0 12px;}
.im-send-voice{margin-top:12px; font-size:28rpx; color:#00BA62; text-align:center;}
.im-send-voice text{margin:0 15px; color:#00BA62;}
.toolBox {
	height:72rpx;
	.recorder{
	display: flex;
	align-items: center;
	justify-content: center;
	background-color: #fff;
	padding: 14rpx;
	border-radius: 10rpx;
	font-size: 28rpx;
	box-shadow: 2rpx 2rpx 6rpx rgba(0, 0, 0, 0.2);
	position: relative;
	margin: 0 6rpx !important;
	height:100%;
	&.active {
		background-color: #67C23A;;
		color:#fff;
	}
}
}
.voice-model{
	width:240rpx;
	height:180rpx;
	position: fixed;
	top: 50%;
	z-index: 2;
	transform: translate(-50%, -50%);
	left: 50%;
	padding: 0 4rpx 0 6rpx;
	background-color: #363636b3;
}
.im-emoji-box{
	position:fixed;bottom:0;width:100%;background-color: #F1F2F3;z-index:3;
	.im-emoji-header{
		height:90rpx;
	}
	.im-emoji-body{
		height:100%;padding-bottom: 100rpx;
		.im-emoji-item{
			padding:22rpx;
		}
	}
}
.im-app-box{
	position:fixed;bottom:0;width:100%;background-color: #F1F2F3;z-index:3;
	.im-app-item{
		width:160rpx;height:160rpx;
		.im-app-item-icon{
			padding:20rpx 25rpx;
		}
	}
}

.message-quote{
	padding:8rpx;
	font-size:24rpx;
	margin:16rpx 10rpx 0 10rpx;
	background-color: #e3e3e3;
	.text-overflow{
		overflow: hidden !important;
		text-overflow: ellipsis;
		white-space: nowrap !important;
		width:380rpx;
	}
}
</style>