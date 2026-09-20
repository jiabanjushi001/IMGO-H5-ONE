<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>{{title}}</template>
		</cu-custom>
		<view class="cu-bar bg-white search fixed" :style="[{top:CustomBar + 'px'}]">
			<view class="search-form round">
				<text class="cuIcon-search"></text>
				<input type="text" v-model="keywords" placeholder="请输入用户账号进行搜索" confirm-type="search"/>
				
			</view>
			<view class="action">
				<button class="cu-btn round bg-green" @tap="search()">搜索</button>
			</view>
		</view>
		
		<view style="margin-top:120rpx">
			<view>
				<view class="cu-list menu no-padding">
					<view class="cu-item" v-for="(items,sub) in contacts" :key="sub" @tap='openDetails(items)'>
						<view class='cu-avatar radius mr-15' :style="[{backgroundImage:'url('+items.avatar+')'}]">
						</view>
						<view class="content">
							<view class="c-333">{{items.realname}}</view>
						</view>
						<view class="action ml-10">
							<view class="text-blue" v-if="items.friend" @tap.stop="sendMsg(items.user_id)">发消息</view>
							<view class="text-blue" v-if="!items.friend">查看</view>
						</view>
					</view>
				</view>
			</view>
			<view v-if="!contacts.length">
				<Empty :noDatatext="noText" textcolor="#999" ></Empty>
			</view>
		</view>
	</view>
</template>

<script>
	/**
	 * 初始的引导页
	 */
	export default {
		name  : "search",
		data() {
			return {
				title:'搜索朋友',
				keywords:'',
				contacts:[],
				type:1,
				noText:'暂无数据'
			};
		},
		methods: {
			search(){
				if(this.keywords==''){
					return uni.showToast({
						title:"请输入用户账号进行搜索",
						icon:'none'
					})
				}
				this.noText="未搜索到数据";
				this.$api.msgApi.searchUser({keywords:this.keywords}).then((res)=>{
					if(res.code==0){
						this.contacts=res.data;
					}
				})
			},
			// 打开用户详情
			openDetails(items){
				uni.navigateTo({
					url:"/pages/contacts/detail?id="+items.user_id
				})
			},
			sendMsg(id){
				uni.navigateTo({
					url:"/pages/message/chat?id="+id
				})
			}
		}
	}
</script>

<style scoped>

</style>