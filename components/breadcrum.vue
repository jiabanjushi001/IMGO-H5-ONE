<template>
	<view>
		<scroll-view class="folder-wap" :scroll-x="true" :scroll-left="99999999">
			<view  class="im-flex im-justify-content-start im-align-items-center">
				<view class="tab-item  im-flex im-justify-content-start im-align-items-center" v-for="(item, index) in tree" @tap="open(item)" :key='index'>
				<view class="mar10 lz-tree-name font-color-999" :class="{ 'font-color-333': index == tree.length - 1 }">{{ item.name }}</view>
					<slot name="icon">
						<text class="font-color-999" :class="[icon ? icon :'cuIcon-right']" v-if="index < tree.length - 1"></text>
					</slot>
				</view>
			</view>
		</scroll-view>
	</view>
</template>

<script>
	export default {
		name: "breadcrum",
		props: {
			tree:{type: Array,default:()=>{return {}}},
			icon:{type: String,default:()=>{return 'cuIcon-right'}}
		},
		data() {
			return {}
		},
		methods:{
			open(item){
				this.$emit('openBread',item);
			}
		}
	}
</script>

<style lang="scss">
	.folder-wap {
		box-sizing: border-box;
		width: 100%;
		white-space: nowrap;
		padding:20rpx;
		.tab-item {
			padding-bottom: 10rpx;
			box-sizing: border-box;
			.lz-tree-name{
				margin:0 20rpx;
			}
			.iconfont{
				font-size:24rpx !important;
			}
			&:last-child {
				margin-right: 32rpx;
				.lz-tree-name{
					margin-right:80rpx;
				}
			}
		}
	}
	.font-color-333{
		color:#303133 !important;
	}
	.font-color-999{
		color:#909399
	}
</style>
