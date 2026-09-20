<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>{{title}}</template>
		</cu-custom>
		<view class="cu-bar bg-white search fixed" :style="[{top:CustomBar + 'px'}]">
			<view class="search-form round">
				<text class="cuIcon-search"></text>
				<input type="text" v-model="keywords" placeholder="输入搜索的关键词" confirm-type="search"/>
			</view>
		</view>
		
		<view style="margin-top:120rpx">
			<view v-if="type<3 && contacts.length>0">
				<view class="padding">联系人</view>
				<view class="cu-list menu-avatar no-padding">
					<view class="cu-item" v-for="(items,sub) in contacts" :key="sub" @tap='openDetails(items)'>
						<view class='cu-avatar lg radius mr-15' :style="[{backgroundImage:'url('+items.avatar+')'}]">
						</view>
						<view class="content">
							<view class="c-333">{{items.displayName}}</view>
						</view>
					</view>
				</view>
			</view>
			<view v-if="groupList.length>0">
				<view class="padding">群聊</view>
				<view class="cu-list menu-avatar no-padding">
					<view class="cu-item" v-for="(items,sub) in groupList" :key="sub" @tap='openDetails(items)'>
						<view class='cu-avatar lg radius mr-15' :style="[{backgroundImage:'url('+items.avatar+')'}]">
						</view>
						<view class="content">
							<view class="c-333">{{items.displayName}}</view>
						</view>
					</view>
				</view>
			</view>
			<view v-if="!groupList.length && !contacts.length">
				<Empty noDatatext="暂无数据" textcolor="#999" ></Empty>
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
				title:'搜索聊天',
				keywords:'',
				allContacts:[],
				contacts:[],
				groupList:[],
				type:1
			};
		},
		watch:{
			keywords(val){
				this.search();
			}
		},
		onLoad(options){
			this.type=options.type;
			if(this.type==2){
				this.title="搜索联系人";
			}else if(this.type==3){
				this.title="搜索群聊";
			}else{
				this.title="搜索聊天";
			}
		},
		mounted(){
			this.allContacts=uni.getStorageSync('allContacts');
		},
		methods: {
			search(){
				const arr=this.$util.searchObject(this.allContacts,['displayName','name_py','account'],this.keywords);
				const contacts=[];
				const groupList=[];
				arr.forEach((item)=>{
					if(item.is_group==1){
						groupList.push(item);
					}else{
						contacts.push(item);
					}
				})
				this.groupList=groupList;
				this.contacts=contacts;
			},
			// 打开聊天
			openDetails(items){
				if(this.type==2 && items.is_gourp==0){
					uni.navigateTo({
						url:"/pages/contacts/detail?id="+items.id
					})
				}
				uni.navigateTo({
					url:"/pages/message/chat?id="+items.id
				})
			}
		}
	}
</script>

<style scoped>

</style>