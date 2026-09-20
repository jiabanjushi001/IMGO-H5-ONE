<template>
	<view class="lz-empty-box">
		<view v-if="showImage" class="im-flex im-align-items-center im-justify-content-center">
			<!-- 请根据您的项目要求制作并更换为空图片 -->
			<image class="empty-img" :src="src"></image>
		</view>
		<view class="text-center">
			<text :class="showImage ? ' mt-20' : '' " :style='{color:textcolor}'>{{noDatatext}}</text>
		</view>
		
	</view>
</template>

<script>
export default {
		name: 'Empty',
		props: {
			src: {
				type: String,
				default: '/static/image/empty.png'
			}, //  图片路径
			showImage: {
				type: Boolean,
				default: true
			},
			noDatatext: {
				type: String,
				default: () => {
					return '暂无数据'
				}
			},
			textcolor: {
				type: String,
				default: "#333333"
			}
		},
		data() {
			return {};
		}

	};
</script>

<style>
	.lz-empty-box {
		margin: 40rpx auto;
		align:center
	}

	.empty-img {
		margin-top: 40rpx;
	}
</style>
