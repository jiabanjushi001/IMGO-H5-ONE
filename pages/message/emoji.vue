<template>
		<cu-custom :isBack="true"  bgColor="bg-white">
			<template #content>表情管理（{{emojiList.length}}）</template>
			<template #right>
				<view class="ml-10 mr-10" @tap="manage()">
					{{manageName}}
				</view>
			</template>
		</cu-custom>
		<view>
			<scroll-view scroll-y class="bg-white" style="padding-bottom:100rpx">
				<uni-grid :column="5" :highlight="true" @change="change" style="padding:10rpx">
					<uni-grid-item>
						<view class="grid-item-box" :index="0">
							<view class="upload-emoji" @tap="uploadEmoji"><text class="cuIcon-add c-999" style="vertical-align: sub;"></text></view>
						</view>
					</uni-grid-item>
					<uni-grid-item v-for="(item, index) in emojiList" :index="index+1" :key="index">
						<view class="grid-item-box">
							<AuthImage :src="item.displaySrc || item.src" style="width:100rpx" mode="widthFix"></AuthImage>
							<view class="emoji-check-box" v-if="isManage" :class="item.isCheck ? 'text-green cuIcon-roundcheckfill' : 'cuIcon-round'"></view>
						</view>
					</uni-grid-item>
				</uni-grid>
			</scroll-view>
			<view class="btn-opt bg-white im-flex im-align-items-center im-space-between pd-10" v-if="checkList.length>0">
				<button class="cu-btn bg-green" @tap="moveEmoji()">移动到最前面</button>
				<button class="cu-btn bg-red" @tap="delEmoji()">删除({{checkList.length}})</button>
			</view>
		</view>
</template>

<script>
	export default {
		data() {
			return {
				emojiList:[],
				isManage:false,
				manageName:'整理',
				checkList:[]
			}
		},
		onLoad(option){
			this.url=decodeURI(option.src);
			this.name=option.name;
		},
		mounted(){
			this.getEmojiList();
		},
		methods: {
			getEmojiList(){
				this.$api.emojiApi.emojiList({}).then(async (res)=>{
					if(res.code==0){
						const { mapAuthDisplayField } = await import('@/utils/avatar.js')
						await mapAuthDisplayField(res.data, 'src', 'displaySrc')
						this.emojiList=res.data;
					}
				})
			},
			// 上传表情
			uploadEmoji(){
				uni.chooseImage({
					count      : 9,
					sizeType   : ['compressed'],
					sourceType : ['album', 'camera'],
					success    : (res)=>{
						const tempFiles = res.tempFiles;
						tempFiles.forEach((item) => {
							if(item.size>2*1024*1024){
								return uni.showToast({
									title: '表情大小不能超过2MB',
									icon:'error'
								})
							}
							uni.uploadFile({
								url: this.$api.emojiApi.uploadEmoji,
								filePath: item.path,
								name: 'file',
								header: {
									'Authorization': uni.getStorageSync('authToken'),
								},
								success: (e) => {
									let res=JSON.parse(e.data);
									if(res.code==0){
										this.updateEmoji();
									}
								},
								fail: (res) => {
								}
							})
					    })
					}
				});
			},
			manage(){
				this.isManage =!this.isManage;
				this.manageName = this.isManage ? '取消' : '整理';
				if(!this.isManage){
					this.checkList=[];
					this.emojiList.forEach((item,index)=>{
						this.emojiList[index].isCheck=false;
					})
				}
			},
			change(e) {
				let { index } = e.detail;
				if(!this.isManage || index==0) return;
				let isCheck=this.emojiList[index-1].isCheck;
				let id=this.emojiList[index-1].id;
				if(!isCheck){
					this.checkList.push(id);
				}else{
					this.checkList = this.checkList.filter(item => item !== id);
				}
				this.emojiList[index-1].isCheck=!isCheck;
			},
			// 移动表情
			moveEmoji(){
				if(this.checkList.length<=0){
					return;
				}
				this.$api.emojiApi.moveEmoji({ids:this.checkList}).then((res)=>{
					this.updateEmoji()
				})
			},
			// 删除表情
			delEmoji(){
				if(this.checkList.length<=0){
					return;
				}
				uni.showModal({
					title: '确定要删除选中的表情吗？',
					success: (res)=>{
						if (res.confirm) {
							this.$api.emojiApi.delEmoji({ids:this.checkList}).then((res)=>{
								this.updateEmoji()
							})
						}
					},
				})
				
			},
			updateEmoji(){
				this.getEmojiList();
				this.checkList=[];
				uni.$emit('updateEmoji',true)
			}
		}
	}
</script>

<style lang="scss">
	page{
		background-color: #fff;
	}
	.upload-emoji{
		height:70rpx;
		font-size:50rpx;
		text-align: center;
	}
	.grid-dynamic-box {
		margin-bottom: 15px;
	}
	
	.btn-opt{
		position: fixed;
		bottom: 0;
		width:100%;
		height:100rpx;
		border-top: solid 1px #D2D2D2;
		z-index:1000;
	}

	.grid-item-box {
		flex: 1;
		// position: relative;
		/* #ifndef APP-NVUE */
		display: flex;
		/* #endif */
		flex-direction: column;
		align-items: center;
		justify-content: center;
	}

	.grid-item-box-row {
		flex: 1;
		// position: relative;
		/* #ifndef APP-NVUE */
		display: flex;
		/* #endif */
		flex-direction: row;
		align-items: center;
		justify-content: center;
		padding: 15px 0;
	}
	
	.grid-dot {
		position: absolute;
		top: 5px;
		right: 15px;
	}

	.swiper {
		height: 420px;
	}
	
	.emoji-check-box{
		position:absolute;
		right:5rpx;
		top:5rpx;
	}

	/* #ifdef H5 */
	@media screen and (min-width: 768px) and (max-width: 1425px) {
		.swiper {
			height: 630px;
		}
	}

	@media screen and (min-width: 1425px) {
		.swiper {
			height: 830px;
		}
	}

	/* #endif */
</style>
