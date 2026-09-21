<template>
	<view>
		<view class="cu-custom" :style="[{height:CustomBar + 'px'}]">
			<view class="cu-bar fixed" :style="style" :class="[bgImage!=''?'none-bg text-white bg-img':'',bgColor, barClass]">
				<view class="action" @tap="BackPage" v-if="isBack">
					<text class="cuIcon-back"></text>
					<slot name="backText"></slot>
				</view>
				<view class="action" v-else>
					<slot name="backText"></slot>
				</view>
				<view class="content" :style="[{top:StatusBar + 'px'}]">
					<slot name="content"></slot>
				</view>
				<view class="right">
					<slot name="right"></slot>
				</view>
				
			</view>
		</view>
	</view>
</template>

<script>
	export default {
		data() {
			return {
				StatusBar: this.StatusBar,
				CustomBar: this.CustomBar
			};
		},
		name: 'custom',
		computed: {
			style() {
				var StatusBar= this.StatusBar;
				var CustomBar= this.CustomBar;
				var bgImage = this.bgImage;
				var style = `height:${CustomBar}px;padding-top:${StatusBar}px;`;
				if (this.bgImage) {
					style = `${style}background-image:url(${bgImage});`;
				}
				if (this.bgStyle) {
					style = `${style}${this.bgStyle}`;
				}
				return style
			}
		},
		props: {
			bgColor: {
				type: String,
				default: ''
			},
			barClass: {
				type: String,
				default: ''
			},
			/** 直接写到顶栏的内联样式，如 background:linear-gradient(...) */
			bgStyle: {
				type: String,
				default: ''
			},
			isBack: {
				type: [Boolean, String],
				default: false
			},
			fallbackToHome: {
				type: Boolean,
				default: false
			},
			bgImage: {
				type: String,
				default: ''
			},
		},
		methods: {
			/** 聊天页返回首页。H5 下 navigateBack 常无响应；刷新后能返回是因为栈只剩一页走了 switchTab */
			backFromChat() {
				uni.switchTab({
					url: '/pages/index/index',
					fail: () => {
						uni.reLaunch({ url: '/pages/index/index' });
					}
				});
			},
			BackPage() {
				const allroutes = getCurrentPages();
				const currentRoute = allroutes[allroutes.length - 1]?.route;
				if (currentRoute == 'pages/message/chat') {
					this.backFromChat();
					return;
				}
				if (allroutes.length < 2 && this.fallbackToHome) {
					uni.switchTab({ url: '/pages/index/index' });
					return;
				}
				if (allroutes.length < 2 && 'undefined' !== typeof __wxConfig) {
					let url = '/' + __wxConfig.pages[0]
					return uni.redirectTo({url})
				}
				uni.navigateBack({
					delta: 1,
					fail: () => {
						uni.switchTab({ url: '/pages/index/index' });
					}
				});
			}
		}
	}
</script>

<style>
/* 标题层绝对定位会盖住左右按钮；禁止标题及其子节点抢点击，保证返回可点 */
.cu-bar > .action,
.cu-bar > .right {
	position: relative;
	z-index: 10;
	pointer-events: auto;
}
.cu-bar > .content {
	pointer-events: none !important;
}
.cu-bar > .content * {
	pointer-events: none !important;
}
</style>
