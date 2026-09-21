<template>
		<cu-custom :isBack="true" style="color:white !important">
			<template #backText>关闭视频</template>
		</cu-custom>
		<view class="video-model im-flex im-align-items-center" >
			
			<video class="video-box" id="myVideo"  :src="playUrl"  controls autoplay="autoplay" style="width:100%;height:100vh"></video>
			<view class="opt-model im-flex  im-align-items-center">
				<button class="cu-btn round mr-10" @tap="download">保存到本地</button>
				<button class="cu-btn round" @tap="closeModel">关闭</button>
			</view>
		</view>
</template>

<script>
	import { resolveMediaDisplayUrl, triggerAuthedDownload, downloadAuthedFile } from '@/utils/avatar.js'
	export default {
		data() {
			return {
				url:'',
				playUrl:'',
				name:''
			}
		},
		onLoad(option){
			this.url=decodeURIComponent(option.src || '');
			this.name=option.name || 'video';
			this.playUrl = ''
			resolveMediaDisplayUrl(this.url).then((displayUrl) => {
				this.playUrl = displayUrl || this.url
			}).catch(() => {
				this.playUrl = this.url
			})
		},
		mounted(){
		},
		methods: {
			closeModel(){
				uni.navigateBack();
			},
			download(){
				uni.showLoading({ title: '保存中' })
				// #ifndef H5
				downloadAuthedFile(this.url).then(({ tempFilePath }) => {
					uni.hideLoading()
					uni.saveVideoToPhotosAlbum({
						filePath: tempFilePath,
						success: function () {
							uni.showToast({
								title:"已保存到相册",
								icon:'none'
							})
						},
						fail: () => {
							uni.showToast({ title: '保存失败', icon: 'none' })
						}
					});
				}).catch(() => {
					uni.hideLoading()
					uni.showToast({ title: '下载失败', icon: 'none' })
				})
				// #endif
				// #ifdef H5
				triggerAuthedDownload(this.url, this.name).then(() => {
					uni.hideLoading()
					uni.showToast({ title: '已开始下载', icon: 'none' })
				}).catch(() => {
					uni.hideLoading()
					uni.showToast({ title: '下载失败', icon: 'none' })
				})
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
