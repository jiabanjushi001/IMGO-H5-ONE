<template name="im-item">
	<view v-if="item.type=='diy'" :item="item" :index="index" :isSelf="isSelf">
		自定义
	</view>
	<view v-else>
		暂不支持该消息类型
	</view>
</template>
<script>
// import imVoice from '@/components/message/im-card-voice.vue';
export default {
	
	name  : "im-item",
	components: {
		// imVoice
	},
	props : {
		item:{type:Object, default:function(){return {};}},
		index:{type:Number, default:0},
		isSelf:{type:Boolean, default:false},
	},
	data() {
		return {
		}
	},
	created : function(){
	},
	methods:{
		
	}
}
</script>
<style scoped>

</style>