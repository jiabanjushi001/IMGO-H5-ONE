<template>
	<view>
		<view class="cu-custom" :style="[{height:CustomBar + 'px'}]">
			<view class="cu-bar fixed" :style="style" :class="[bgImage!=''?'none-bg text-white bg-img':'',bgColor]">
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
				return style
			}
		},
		props: {
			bgColor: {
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
			BackPage() {
				const allroutes = getCurrentPages();
				const currentRoute = allroutes[allroutes.length - 1]?.route;
				// 如果当前在聊天页面，自动返回到首页，避免层级过深
				if (currentRoute == 'pages/message/chat') {
					uni.switchTab({
						url: '/pages/index/index'
					});
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
					delta: 1
				});
			}
		}
	}
</script>
