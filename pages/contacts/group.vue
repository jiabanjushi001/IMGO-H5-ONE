<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>群聊列表</template>
			<template #right>
				<view class="f-20 ml-10 mr-10" @tap="search()">
					<text class="cuIcon-search"></text>
				</view>
			</template>
		</cu-custom>
		<view class="cu-list menu-avatar no-padding">
			<view class="cu-item" v-for="(items,sub) in groupList" :key="sub" @tap='openDetails(items)'>
				<view class='cu-avatar lg radius mr-15' :style="[{backgroundImage:'url('+items.avatar+')'}]">
				</view>
				<view class="content">
					<view class="c-333">{{items.displayName}}</view>
				</view>
				<view class="action">
					 <text class="c-999 cuIcon-peoplefill" v-if="items.owner_id==userInfo.user_id"></text>
				</view>
				
			</view>
			<Empty v-if="!groupList.length" noDatatext="暂无群聊" textcolor="#999" ></Empty>
		</view>

	</view>
</template>

<script>
	import { useMsgStore } from '@/store/message';
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	const userStore = useloginStore(pinia);
	const msgStore = useMsgStore(pinia)
	/**
	 * 初始的引导页
	 */
	export default {
		name  : "group",
		data() {
			return {
				userInfo:userStore.userInfo
			};
		},
		computed: {
			groupList() {
				return msgStore.contacts.filter(item => Number(item.is_group) === 1).sort((a, b) => {
					if (a.index === '#') return 1;
					if (b.index === '#') return -1;
					return String(a.index || '').localeCompare(String(b.index || ''), 'zh');
				});
			}
		},
		methods: {
			// 打开聊天
			openDetails(items){
				uni.navigateTo({
					url:"/pages/message/chat?id="+items.id
				})
			},
			search(){
				uni.navigateTo({
					url:"/pages/index/search?type=3"
				})
			}
		}
	}
</script>

<style scoped>

</style>
