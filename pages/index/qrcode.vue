<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true" :fallbackToHome="true">
			<template #backText></template>
			<template #content>{{title}}</template>
		</cu-custom>
		<view class="mt-20 im-flex im-align-items-center">
			<image :src="canvasUrl" mode="widthFix" style="margin:0 auto"></image>
		</view>
		<view v-if="errorMessage" class="padding text-center text-grey">{{ errorMessage }}</view>
		<view class="padding flex flex-direction mt-10" v-if="canvasUrl">
			<button class="cu-btn bg-green lg"  @tap="saveHeadImgFile">
				 <!-- #ifdef H5 -->
					下载图片
				 <!-- #endif -->
				 <!-- #ifndef H5 -->
					保存到相册
				 <!-- #endif -->
				</button>
		</view>
		<mosowe-canvas-image ref="mosoweCanvasComponents"  :lists="lists" :height="height" :width="width" @canvasImage="canvasImage" ></mosowe-canvas-image>
	</view>
</template>
<script>
	import mosoweCanvasImage from '@/components/mosowe-canvas-image/mosowe-canvas-image.vue';
	import pinia from '@/store/index'
	import { useloginStore } from '@/store/login';
	const userStore = useloginStore(pinia);
	
	export default {
		data() {
			return {
				canvasUrl: '',
				errorMessage: '',
				group_id:0,
				lists:[],
				width:500,
				height:700,
				title:'我的二维码',
				contact:{}
			}
		},
		components:{
			mosoweCanvasImage
		},
		onLoad(options) {
			this.group_id = options.group_id?options.group_id:''
			// 如果有团队ID就生成群聊二维码,没有就生成个人二维码
			if(this.group_id){
				this.title="群二维码";
				this.getGroupInfo()
			}else{
				this.createUserQr();
			}
			
		},
		methods: {
			createUserQr(){
				let userInfo=userStore.userInfo;
				let qrUrl=userInfo.qrUrl ?? '';
				if(!qrUrl){
					uni.showToast({
						title:'请重新登陆后再生成二维码',
						icon:'none'
					})
					uni.navigateBack();
					return;
				}
				this.lists=[
					{
					    type: 'rect',
					    color: '#ffffff', 
					    width: this.width, 
					    height: this.height, 
					    x: 0, 
					    y: 0
					},
				    {
				        type: 'image',
				        content: userInfo.avatar, 
				        width: 100, 
				        height: 100, 
				        x: 70, 
				        y: 60
				    },
				    {
				        type: 'text',
				        content: userInfo.displayName, 
				        x: 190,
				        y: 96, 
						maxWidth:400,
				        color: '#000',
				        size: 28, 
				    },
					{
					    type: 'text',
					    content: userInfo.account, 
					    x: 190,
					    y: 135, 
						maxWidth:400,
					    color: '#999',
					    size: 20, 
					},
				    {
				        type: 'qr',
				        content: qrUrl, 
				        width: 360, 
				        height: 360, 
				        x: 70, 
				        y: 200, 
				    },
					{
					    type: 'text',
					    content: '扫一扫上面的二维码图案，加我为好友',
					    x: 105, 
					    y: 620, 
					    color: '#999',
						maxWidth:400,
					    size: 18, 
					},
				];
				setTimeout(()=>{
					this.$refs.mosoweCanvasComponents.createCanvas();
				},100)
			},
			getGroupInfo() {
				this.$api.msgApi.groupInfo({
					group_id: this.group_id
				}).then((res) => {
					const data = res?.code === 0 && res.data && !Array.isArray(res.data) ? res.data : null;
					if (!data || !data.qrUrl) {
						this.errorMessage = res?.msg || '群二维码暂不可用';
						return;
					}
					this.errorMessage = '';
					this.contact=data;
					let groupName=String(data.name || data.displayName || '群聊');
					let length=groupName.length;
					if(length>12){
						groupName=groupName.substring(0,12)+"...";
					}
					this.lists=[
						{
						    type: 'rect',
						    color: '#ffffff', 
						    width: this.width, 
						    height: this.height, 
						    x: 0, 
						    y: 0
						},
                        {
                            type: 'image',
							content: './static/image/group.png',
                            width: 100, 
                            height: 100, 
                            x: 200, 
                            y: 40
                        },
                        {
                            type: 'text',
                            content: '群聊：'+groupName, 
                            x: this.width/2, 
							align:'center',
                            y: 180, 
							maxWidth:400,
                            color: '#000',
                            size: 24, 
                        },
                        {
                            type: 'qr',
                            content: data.qrUrl, 
                            width: 360, 
                            height: 360, 
                            x: 70, 
                            y: 220, 
                        },
						{
						    type: 'text',
						    content: data.qrExpire ? '该二维码7天内（'+data.qrExpire+'前）有效' : '扫码加入群聊',
						    x: 105, 
						    y: 640, 
						    color: '#999',
							maxWidth:400,
						    size: 18, 
						},
                    ];
					this.$nextTick(() => {
						this.$refs.mosoweCanvasComponents?.createCanvas();
					});
				}).catch(() => {
					this.errorMessage = '群二维码获取失败，请稍后重试';
				});
			},
			canvasImage (e) {
				this.canvasUrl = e;
			},
			// 将base64 图片保存到本地系统相册
			saveHeadImgFile() {
				// #ifdef H5
					const tempLink = document.createElement("a");
					tempLink.style.display = "none";
					tempLink.href = this.canvasUrl;
					tempLink.setAttribute("download", this.contact.name+".jpg");
					tempLink.setAttribute("target", "_blank");
					document.body.appendChild(tempLink);
					tempLink.click();
					document.body.removeChild(tempLink);
				// #endif
				// #ifndef H5
					uni.saveImageToPhotosAlbum({
						filePath: this.canvasUrl,
						success: () => {
							uni.showToast({title:'图片保存成功',icon:'none'})
						}
					})
				// #endif
				
			}
		}
	}
</script>
<style scoped>
	.list-image {
		width: 80rpx;
		height: 80rpx;
		font-size: 0;
	}
</style>
