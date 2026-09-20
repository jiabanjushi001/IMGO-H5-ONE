<template>
	<view  :style="{paddingBottom: paddingB+'px'}">
		<view class="cu-list menu" :class="['sm-border','card-menu margin-top']" v-if="compass.mode==1">
			<view class="cu-item" :class="'arrow'" v-for="(item,index) in compass.list" :key="index"  @tap="openApp(item)">
				<view class="content">
					<image :src="item.icon" class="png" mode="aspectFit"></image>
					<text class="text-grey">{{item.name}}</text>
				</view>
			</view>
		</view>
		<view class="cu-list grid" :class="['col-3',' margin-top']" v-if="compass.mode==2">
			<view class="cu-item" v-for="(item,index) in compass.list" :key="index" @tap="openApp(item)">
				<view >
					<image :src="item.icon" style="height:100rpx;width: 100rpx;"></image>
					<view class="cu-tag badge" v-if="item.badge!=0">
						<block v-if="item.badge!=1">{{item.badge>99?'99+':item.badge}}</block>
					</view>
				</view>
				<text>{{item.name}}</text>
			</view>
		</view>
		
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	const loginStore = useloginStore(pinia)
	export default {
		data() {
			return {
				isCard: true,
				userInfo:loginStore.userInfo,
				paddingB:0
			};
		},
		computed:{
			compass:()=>{
				return loginStore.globalConfig.compass
			},
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
			openApp(item) {
				let url=item.url;
				if(item.type==2){
					url='/pages/mine/webview?title='+item.name+'&src='+encodeURIComponent(item.url)
				}
				uni.navigateTo({
					url:url
				})
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