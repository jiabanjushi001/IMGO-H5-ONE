<template>
		<cu-custom :isBack="true" style="color:white !important">
			<template #backText>关闭视频</template>
		</cu-custom>
		<view class="video-model im-flex im-align-items-center" >
			
			<video class="video-box" id="myVideo"  :src="url"  controls autoplay="autoplay" style="width:100%;height:100vh"></video>
			<view class="opt-model im-flex  im-align-items-center">
				<button class="cu-btn round mr-10" @tap="download">保存到本地</button>
				<button class="cu-btn round" @tap="closeModel">关闭</button>
			</view>
		</view>
</template>

<script>
	export default {
		data() {
			return {
				url:'',
				name:''
			}
		},
		onLoad(option){
			this.url=decodeURI(option.src);
			this.name=option.name;
		},
		mounted(){
		},
		methods: {
			closeModel(){
				uni.navigateBack();
			},
			download(){
				// #ifndef H5
				uni.downloadFile({
					url: this.url, 
					success: (res) => {
						if (res.statusCode === 200) {
							uni.saveVideoToPhotosAlbum({
								filePath: res.tempFilePath,
								success: function () {
									uni.showToast({
										title:"已保存到相册",
										icon:'none'
									})
								}
							});
							
						}
					}
				});
				// #endif
				// #ifdef H5
				const tempLink = document.createElement("a");
				tempLink.style.display = "none";
				tempLink.href = this.url;
				tempLink.setAttribute("download", this.name);
				tempLink.setAttribute("target", "_blank");
				document.body.appendChild(tempLink);
				tempLink.click();
				document.body.removeChild(tempLink);
				// #endif
			}
		}
	}
</script>

<style lang="scss">
	.video-model{
		background-color: #000;width: 100%;height: 100%;position: fixed;top:0;overflow:hidden;;
	}
	.opt-model{
		position: absolute;bottom:100rpx;right:20rpx;padding:4rpx 10rpx;text-align: center;
		.bm-btn{
			width:120rpx;
		}
	}
	.video-box{width:100%}
</style>
