<template>
	<cu-custom bgColor="bg-white" :isBack="true">
		<template #backText></template>
		<template #content>朋友圈</template>
	</cu-custom>
	<view  :style="{paddingBottom: paddingB+'px'}">
		<view class="im-friend-header">
			<view class="im-friend-bg">
				<image class="im-friend-image" src="./static/image/user-card-bg.jpg" mode="widthFix"></image>
			</view>
			<view class="im-user im-flex im-justify-content-start align-center">
				<text class="text-white mr-5">{{userInfo.realname}}</text>
				<image class="radius-10" style="width:120rpx;height:120rpx" :src="userInfo.avatar" mode="widthFix"></image>
			</view>
			
		</view>
		
		<view class="m-10 text-center">此页面是静态模板！</view>
		
		<view class="cu-card dynamic no-card">
			<view class="cu-item shadow">
				<view class="cu-list menu-avatar">
					<view class="cu-item">
						<view class="cu-avatar round lg" style="background-image:url(https://api.multiavatar.com/raingad3.png?apikey=zdvXV3W4MjwhP9);"></view>
						<view class="content flex-sub">
							<view>凯尔</view>
							<view class="text-gray text-sm flex justify-between">
								2019年12月3日
							</view>
						</view>
					</view>
				</view>
				<view class="text-content">
					折磨生出苦难，苦难又会加剧折磨，凡间这无穷的循环，将有我来终结！
				</view>
				<view class="grid flex-sub padding-lr" :class="isCard?'col-3 grid-square':'col-1'">
					<view class="bg-img" :class="isCard?'':'only-img'" style="background-image:url(https://ossweb-img.qq.com/images/lol/web201310/skin/big10006.jpg);"
					 v-for="(item,index) in isCard?9:1" :key="index">
					</view>
				</view>
				<view class="text-gray text-sm text-right padding">
					<text class="cuIcon-attentionfill margin-lr-xs"></text> 10
					<text class="cuIcon-appreciatefill margin-lr-xs"></text> 20
					<text class="cuIcon-messagefill margin-lr-xs"></text> 30
				</view>

				<view class="cu-list menu-avatar comment solids-top">
					<view class="cu-item">
						<view class="cu-avatar round" style="background-image:url(https://api.multiavatar.com/raingad5.png?apikey=zdvXV3W4MjwhP9);"></view>
						<view class="content">
							<view class="text-grey">莫甘娜</view>
							<view class="text-gray text-content text-df">
								凯尔，你被自己的光芒变的盲目。
							</view>
							<view class="bg-grey padding-sm radius margin-top-sm  text-sm">
								<view class="flex">
									<view>凯尔：</view>
									<view class="flex-sub">妹妹，你在帮他们给黑暗找借口吗?</view>
								</view>
							</view>
							<view class="margin-top-sm flex justify-between">
								<view class="text-gray text-df">2018年12月4日</view>
								<view>
									<text class="cuIcon-appreciatefill text-red"></text>
									<text class="cuIcon-messagefill text-gray margin-left-sm"></text>
								</view>
							</view>
						</view>
					</view>

					<view class="cu-item">
						<view class="cu-avatar round" style="background-image:url(https://api.multiavatar.com/raingad2.png?apikey=zdvXV3W4MjwhP9);"></view>
						<view class="content">
							<view class="text-grey">凯尔</view>
							<view class="text-gray text-content text-df">
								妹妹，如果不是为了飞翔，我们要这翅膀有什么用?
							</view>
							<view class="bg-grey padding-sm radius margin-top-sm  text-sm">
								<view class="flex">
									<view>莫甘娜：</view>
									<view class="flex-sub">如果不能立足于大地，要这双脚又有何用?</view>
								</view>
							</view>
							<view class="margin-top-sm flex justify-between">
								<view class="text-gray text-df">2018年12月4日</view>
								<view>
									<text class="cuIcon-appreciate text-gray"></text>
									<text class="cuIcon-messagefill text-gray margin-left-sm"></text>
								</view>
							</view>
						</view>
					</view>
				</view>
			</view>
		</view>
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	const loginStore = useloginStore()
	export default {
		data() {
			return {
				isCard: true,
				userInfo:loginStore.userInfo,
				paddingB:0
			};
		},
		created:function(){
			// #ifdef H5
			this.paddingB=this.inlineTools;
			// #endif
			
			// #ifndef H5
			this.paddingB=this.navBarHeight+this.inlineTools;
			// #endif
		},
		methods: {
			IsCard(e) {
				this.isCard = e.detail.value
			},
		}
	}
</script>

<style lang="scss">
.im-friend-header{
	width:100%;height:400rpx;position: relative;
	.im-friend-bg{
		width:100%;height:300rpx;overflow: hidden;
		.im-friend-image{
			width:100%;height:300rpx;
		}
	}
}
.im-user{
	position: absolute;right:60rpx;top:210rpx;overflow: hidden;
}
</style>
