<template>
	<view>
		<cu-custom bgColor="bg-gradual-blue" :isBack="true">
			<template #backText></template>
			<template #content>新邀请</template>
			<template #right>
				<view class="f-20 ml-10 mr-10" @tap="searchFriend()">
					<text class="cuIcon-add"></text>
				</view>
			</template>
		</cu-custom>
		<uni-segmented-control :current="current" :values="items" @clickItem="onClickItem" styleType="text"></uni-segmented-control>
		<view class="cu-list menu">
			<view class="cu-item" v-if="!params.is_mine" v-for="(x,index) in list" :key="index">
				<view class='cu-avatar md radius mr-15' :style="[{backgroundImage:'url('+x.create_user_info.avatar+')'}]">
				</view>
				<view class="content padding-tb-sm" @tap='openDetails(x.create_user_info.user_id)'>
					<view class="text-grey"  v-if="!params.is_mine">
						<text class="text-blue">{{x.create_user_info.realname}} </text> 申请添加您为好友
					</view>
					<view class="text-gray text-sm lh-15x">{{x.remark}}</view>
				</view>
				<view class="action ml-10">
					<text class="text-red" v-if="x.status==0">已拒绝</text>
					<text class="text-blue" v-if="x.status==1" @tap="sendMsg(x.create_user_info.user_id)">发消息</text>
					<button class="cu-btn round sm bg-green"  v-if="x.status==2" @tap="optApply(x)">
						操作
					</button>
				</view>
			</view>
			<view class="cu-item"  v-if="params.is_mine" v-for="(x,index) in list" :key="index">
				<view class='cu-avatar md radius mr-15' :style="[{backgroundImage:'url('+x.user_id_info.avatar+')'}]">
				</view>
				<view class="content" @tap='openDetails(x.user_id_info.user_id)'>
					<view class="text-grey">
						请求添加<text class="text-blue"> {{x.user_id_info.realname}} </text> 为好友
					</view>
				</view>
				<view class="action ml-10">
					<text class="text-red" v-if="x.status==0">已拒绝</text>
					<text class="text-blue" v-if="x.status==1"  @tap="sendMsg(x.user_id_info.user_id)">发消息</text>
					<text  class="text-orange" v-if="x.status==2">待同意</text>
				</view>
			</view>
			<view class="m-10" v-if="list.length">
				<uni-pagination :current="params.page" :total="total" :pageSize="params.limit"  @change="changePage"/>
			</view>
			
			<Empty v-if="!list.length" noDatatext="暂无群聊" textcolor="#999" ></Empty>
		</view>

	</view>
</template>

<script>
	import { storeToRefs } from 'pinia';
	import { useMsgStore } from '@/store/message';
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
	const {contacts} = storeToRefs(msgStore);
	/**
	 * 初始的引导页
	 */
	export default {
		name  : "group",
		data() {
			return {
				items: ['我收到的', '我发起的'],
				current: 0,
				list: [],
				total:0,
				params: {
					page: 1,
					limit: 10,
					is_mine:0
				}
			};
		},
		created() {
		},
		mounted(){
			this.getList();
		},
		methods: {
			getList(){
				this.$api.friendApi.applyList(this.params).then((res)=>{
					if(res.code==0){
						this.list = res.data;
						this.total = res.count;
					}
				})
			},
			changePage(e){
				this.params.page=e.current;
				this.getList();
			},
			onClickItem(e) {
				this.params.is_mine = e.currentIndex;
				this.current = e.currentIndex;
				this.params.page = 1;
				this.getList();
			},
			sendMsg(id){
				uni.navigateTo({
					url:"/pages/message/chat?id="+id
				})
			},
			openDetails(id){
				uni.navigateTo({
					url:"/pages/contacts/detail?id="+id
				})
			},
			searchFriend(){
				uni.navigateTo({
					url:"/pages/contacts/search"
				})
			},
			optApply(x){
				uni.showModal({
					title: '提示',
					content:"你确定同意该好友的请求吗",
					cancelText:"拒绝",
					cancelColor:'#e54d42',
					confirmText:"接受",
					success: (res)=>{
						let status=0;
						if (res.confirm) {
							status=1
						}
						this.$api.friendApi.acceptApply({friend_id:x.friend_id,status:status}).then((e)=>{
							if(e.code==0){
								uni.showToast({
									title:e.msg,
									icon:'none'
								})
								msgStore.sysUnread--;
								this.getList();
							}
						})
					}
				})
			}
			
		}
	}
</script>

<style scoped>

</style>