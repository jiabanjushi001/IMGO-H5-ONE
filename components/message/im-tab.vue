<template name="im-tab">
	<view class="tab-main im-flex im-justify-content-start im-align-items-center bg-gray" :style="{height:height+'rpx',borderRadius:height/2+'rpx'}">
		<view class="tab-item" :class="active==index ? 'active' : ''" v-for="(item,index) in values" @click="changeItem(item,index)"  :style="{height:itemHeight+'rpx',borderRadius:itemHeight/2+'rpx',lineHeight:itemHeight-8+'rpx'}" :key="index">
			{{item.name}} <text v-if="item.count>0">{{item.count>99 ? '99+' : item.count}}</text>	
		</view>
	</view>
</template>
<script>
export default {
	
	name  : "im-tab",
	components: {
	},
	props : {
		values:{type:Array, default:function(){return [];}},
		height:{type:Number,default:72}
	},
	data() {
		return {
			active:0,
			itemHeight:48
		}
	},
	created : function(){
		this.itemHeight=this.height-16;
	},
	methods:{
		changeItem(item,index){
			this.active=index;
			this.$emit('change',item,index)
		}
	}
}
</script>
<style lang="scss" scoped>
	.tab-main{
		padding:8rpx;
		.tab-item{
			padding: 4rpx 14rpx;
			min-width:150rpx;
			text-align: center;
			vertical-align: middle;
			font-weight: 500;
			color:#737d91;
			&.active{
				box-shadow: 0 6rpx 18rpx rgba(51, 66, 112, .10);
				background-color: #fff;
				font-weight: 700;
				color:#526dff;
			}
		}
	}
</style>
