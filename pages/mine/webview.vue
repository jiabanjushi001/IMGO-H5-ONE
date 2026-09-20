<template>
  <view class="">
    <web-view :src="html"></web-view>
  </view>
</template>

<script>
export default {
		data() {
			return {
				html: ''
			}
		},
		onLoad: function (option) {
			this.html=decodeURIComponent(option.src)
			if(option.title){
				uni.setNavigationBarTitle({    
					title: option.title
				});//设置标题

			}
		},
	}
</script>

<style>

</style>
