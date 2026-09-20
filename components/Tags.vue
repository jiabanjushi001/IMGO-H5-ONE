<template>
	<view class="mr-5 radius-4 im-tags lh-15x" :class="[
	size=='mini' ? 'im-tags-mini' : '', 
	size=='small' ? 'im-tags-samll' : '', 
	size=='medium' ? 'im-tags-medium' : '', 
	type == 'info' ? 'im-tags-info' : '' ,
	type == 'success' ? 'im-tags-success' : '' ,
	type == 'warning' ? 'im-tags-warning' : '', 
	type == 'danger' ? 'im-tags-danger' : '',]"
	>{{text}}</view>
</template>

<script>
	export default {
		name: "tags",
		props: {
			size:{type: String,default:'mini'}, //  图片路径
			type:{type: String,default:'primary'},
			text:{type: String, default:()=>{return '暂无'}}
		},
		data() {
			return {}
		}
	}
</script>

<style>
.im-tags{
	color: #175CFF;
	background-color: #D4E1FF;
	padding:10rpx 20rpx;
	font-size: 24rpx;
	border: 1px solid #b5cbff;
}

.im-tags-success{
	background-color: #f0f9eb;
	border-color: #e1f3d8;
	color: #67c23a;
}

.im-tags-info {
    background-color: #f4f4f5;
	border-color: #e9e9eb;
    color: #909399;
}

.im-tags-warning {
    background-color: #fdf6ec;
	border-color: #faecd8;
    color: #e6a23c;
}

.im-tags-danger {
    background-color: #fef0f0;
	border-color: #fde2e2;
    color: #f56c6c;
}


.im-tags-mini{
	padding:0 8rpx;
	font-size: 22rpx;
}

.im-tags-small{
	padding:6rpx 12rpx;
	font-size: 22rpx;
}
.im-tags-medium{
	padding:8rpx 16rpx;
	font-size: 24rpx;
}

</style>
