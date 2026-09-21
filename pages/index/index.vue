<template>
	<view class="home-page">
		<cu-custom bgColor="bg-white" class="home-header">
			<template #backText>
				<view v-if="PageCur=='message' || PageCur=='contacts'" class="f-20 ml-10 mr-10" @tap="search()">
					<text class="cuIcon-search" style="margin-left: -10px;"></text>
				</view>
			</template>
			<template #content>{{PageName}}</template>
			<template #right>
				<view v-if="PageCur=='message'" class="f-20 ml-10 mr-10" @tap="modelName='add'">
					<text class="cuIcon-add f-28"></text>
				</view>
				
			</template>
		</cu-custom>
		<view class="home-content">
			<message v-show="PageCur=='message'"></message>
			<contacts v-show="PageCur=='contacts'" :TabCur="TabCur"></contacts>
			<compass v-show="PageCur=='compass'"></compass>
			<mine v-show="PageCur=='mine'" :active="PageCur=='mine'" :refresh-key="mineRefreshKey"></mine>
		</view>
		<view class="cu-bar tabbar bg-white shadow foot home-tabbar">
			<view class="action" @click="NavChange(item)" v-for="(item,index) in navList" :key="index" data-cur="message">
				<view class='cuIcon-cu-image'>
					<image :src="$asset('static/image/tabbar/' + item.name + (PageCur==item.name?'-active':'') + '.svg')"></image>
				    <view class="cu-tag badge" v-if="item.notice>0">{{item.notice}}</view>
				</view>
				<view :class="PageCur==item.name?'is-active':'text-black'">{{item.title}}</view>
			</view>
		</view>
		<view class="cu-modal bottom-modal" :class="modelName=='add' ? 'show' : ''" @tap="modelName=''">
			<view class="cu-dialog">
				<view class="cu-list menu bg-white">
					<view class="cu-item" @tap="addFriend()" v-if="globalConfig.sysInfo && globalConfig.sysInfo.runMode==2">
						<view class="content padding-tb-sm">
							<text class="cuIcon-friendadd"></text>
							<text>添加朋友</text>
						</view>
					</view>
					<view class="cu-item" @tap="addGroup()">
						<view class="content padding-tb-sm">
							<text class=" cuIcon-friend"></text>
							<text>添加群聊</text>
						</view>
					</view>
					<view v-if="showScan" class="cu-item" @tap="scan()">
						<view class="content padding-tb-sm">
							<text class=" cuIcon-scan mr-10"></text>
							<text>扫 一 扫</text>
						</view>
					</view>
					<view class="parting-line-5"></view>
					<view class="cu-item" @tap="modelName=''">
						<view class="content padding-tb-sm">
							<text class="c-red">取消</text>
						</view>
					</view>
			
				</view>
			</view>
		</view>
	</view>
</template>

<script>
	import message from '@/pages/message';
	import contacts from '@/pages/contacts';
	import compass from '@/pages/compass';
	import mine from '@/pages/mine';
	import { storeToRefs } from 'pinia';
	import { useMsgStore } from '@/store/message';
	import { useloginStore } from '@/store/login';
	import pinia from '@/store/index'
	import scan from '@/common/scan.js'
	const msgStore = useMsgStore(pinia)
	const loginStore = useloginStore(pinia)
	const { unread,sysUnread } = storeToRefs(msgStore);
	export default {
		components: {
			message,
			contacts,
			compass,
			mine
		},
		data() {
			let navList=[
				{
					name:'message',
					title:'消息',
					notice:unread
				},
				{
					name:'contacts',
					title:'通讯录',
					notice:sysUnread
				}
			]
			let compass={
				name:'compass',
				title:'探索',
				notice:0
			};
			if(loginStore.globalConfig && loginStore.globalConfig.compass){
				if(loginStore.globalConfig.compass.status==1){
					navList.push(compass);
				}
			}
			let mine={
				name:'mine',
				title:'我的',
				notice:0
			}
			navList.push(mine);
		return {
				globalConfig:loginStore.globalConfig,
				PageCur: 'message',
				PageName: '消息',
				TabCur:0,
				modelName:false,
				socketStatusHandler:null,
				initContactsHandler:null,
				mineRefreshKey: 0,
				navList:navList,
				/** 离开页面前保存的滚动位置，返回后恢复 */
				savedScrollTop: 0,
				pendingRestoreScroll: false,
				pageScrollTop: 0,
			}
		},
		computed: {
			showScan() {
				const value = this.globalConfig?.sysInfo?.showScan
				return value !== '0' && value !== 0 && value !== false
			}
		},
		mounted(){
			// #ifndef MP
				uni.hideTabBar();
			// #endif	
			// 监听ws状态,如果重新连接了,要更新联系人
			this.socketStatusHandler = (e)=>{
				if(e){
					console.log('触发了一次');
					this.initContacts();
				}
			}
			uni.$on('socketStatus', this.socketStatusHandler)
			this.initContactsHandler = ()=>{
				this.initContacts();
			}
			uni.$on('initContacts', this.initContactsHandler)
		},
		onUnload() {
			this.cleanupSocketListeners()
		},
		beforeUnmount() {
			this.cleanupSocketListeners()
		},
		onPageScroll(e) {
			this.pageScrollTop = e.scrollTop || 0
		},
		onHide() {
			this.savePageScroll()
		},
		onShow() {
			try { uni.hideLoading(); } catch (e) {}
			this.pendingRestoreScroll = this.savedScrollTop > 0
			this.mineRefreshKey++
			this.initContacts()
			uni.$emit('systemNoticeRefresh')
		},
		methods: {
			cleanupSocketListeners() {
				if (this.socketStatusHandler) {
					uni.$off('socketStatus', this.socketStatusHandler)
					this.socketStatusHandler = null
				}
				if (this.initContactsHandler) {
					uni.$off('initContacts', this.initContactsHandler)
					this.initContactsHandler = null
				}
			},
			/** 读取当前页面滚动容器位置（H5 桌面端可能是 uni-page-wrapper） */
			getPageScrollTop() {
				// #ifdef H5
				try {
					const wrapper = document.querySelector('#app uni-page-wrapper') || document.querySelector('uni-page-wrapper');
					if (wrapper && wrapper.scrollHeight > wrapper.clientHeight + 1) {
						return wrapper.scrollTop || 0;
					}
					return window.pageYOffset || document.documentElement.scrollTop || document.body.scrollTop || 0;
				} catch (e) {
					return this.pageScrollTop || 0;
				}
				// #endif
				// #ifndef H5
				return this.pageScrollTop || 0;
				// #endif
			},
			setPageScrollTop(top) {
				const y = Math.max(0, Number(top) || 0);
				// #ifdef H5
				try {
					const wrapper = document.querySelector('#app uni-page-wrapper') || document.querySelector('uni-page-wrapper');
					if (wrapper) wrapper.scrollTop = y;
					window.scrollTo(0, y);
					if (document.documentElement) document.documentElement.scrollTop = y;
					if (document.body) document.body.scrollTop = y;
				} catch (e) {}
				// #endif
				// #ifndef H5
				uni.pageScrollTo({ scrollTop: y, duration: 0 });
				// #endif
			},
			savePageScroll() {
				this.savedScrollTop = this.getPageScrollTop();
			},
			restorePageScroll() {
				if (!this.pendingRestoreScroll) return;
				const top = this.savedScrollTop;
				this.pendingRestoreScroll = false;
				if (!(top > 0)) return;
				const apply = () => this.setPageScrollTop(top);
				this.$nextTick(() => {
					apply();
					setTimeout(apply, 50);
					setTimeout(apply, 180);
				});
			},
			closeModel(){
				this.modelName=false;
			},
			scan(){
				scan.scanQr();
			},
			NavChange: function(item) {
				this.PageCur = item.name
				this.PageName = item.title
			},
			showContacts(){
				this.TabCur==1 ? this.TabCur=0 :this.TabCur=1
			},
			initContacts(){
				this.modelName='';
				this.$api.msgApi.initContacts().then(res => {
					if (res.code !== 0 || !Array.isArray(res.data)) {
						this.restorePageScroll();
						return;
					}
					// 设置消息未读数和系统消息未读数
					msgStore.sysUnread=res.count;
					const avatarVersion = Date.now();
					const contacts = res.data.map(item => {
						if (Number(item.is_group) !== 1 || !/\/avatar\/group-\d+\//.test(item.avatar || '')) return item;
						return { ...item, avatar: `${item.avatar.split('?')[0]}?v=${avatarVersion}` };
					});
					msgStore.initContacts(contacts);
					this.restorePageScroll();
				}).catch(() => {
					this.restorePageScroll();
				})
			},
			addGroup(){
				uni.navigateTo({
					url: '/pages/index/userSelection?type=1'
				})
			},
			addFriend(){
				uni.navigateTo({
					url: '/pages/contacts/search'
				})
			},
			search(){
				const type = this.PageCur=="message" ? 1 : 2;
				uni.navigateTo({
					url: '/pages/index/search?type='+type
				})
			}
		}
	}
</script>

<style>
/* 首页顶栏高度；占位与固定栏同步；标题/图标垂直居中 */
.home-header :deep(.cu-custom),
.home-page .home-header .cu-custom {
	height: 44px !important;
}
.home-header :deep(.cu-bar),
.home-page .home-header .cu-bar {
	height: 44px !important;
	min-height: 44px !important;
	padding-top: 0 !important;
	display: flex;
	align-items: center;
}
.home-header :deep(.content),
.home-page .home-header .content {
	top: 0 !important;
	bottom: 0 !important;
	height: auto !important;
	margin: auto;
	display: flex !important;
	align-items: center;
	justify-content: center;
	font-size: 34rpx;
	font-weight: 600;
	line-height: 1.2;
}
.home-header :deep(.action),
.home-header :deep(.right),
.home-page .home-header .action,
.home-page .home-header .right {
	height: 100%;
	display: flex;
	align-items: center;
}
</style>
