<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>{{title}}</template>
			<template #right>
				<view class="mr-10 f-16" @tap="save">{{type==3 ? '转发' : '完成'}}</view>
			</template>
		</cu-custom>
		<user-select :type="type" :contact_id="contact_id" :user_ids="user_ids" ref="userSelect"></user-select>
	</view>
</template>

<script>
	import userSelect from '@/components/message/user-select.vue';
	import { useMsgStore } from '@/store/message';
	import { storeToRefs } from 'pinia';
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
	const {appendContacts} = storeToRefs(msgStore);
	export default {
		components: {
			userSelect
		},
		data() {
			return {
				title:'发起群聊',
				contact_id:'',
				selectUser:[],
				userList: [],
				changeUser: [], //选中的数据
				user_ids: [], //
				type: 1,
				relayState: false,
				scrollLeft:300,
				curMsg:{}
			}
		},
		watch: {
			relayState(val) {
				if (val == true) {
					uni.showToast({
						icon: 'success',
						title: '转发成功'
					})
					setTimeout(() => {
						uni.navigateBack()
					}, 2000)
				}

			}
		},
		onLoad(options) {
			var pages = getCurrentPages(); //当前页
			var beforePage = pages[pages.length - 2]; //上个页面
			this.type = options.type ? options.type : 1;
			this.contact_id = options.contact_id ? options.contact_id : '';
			if (options.type == 2) {
				this.title="添加成员";
				// 调用上一页的方法刷新 
				// #ifdef H5
				this.user_ids = beforePage.user_ids
				// #endif
				// #ifndef H5
				this.user_ids = beforePage.$vm.user_ids
				// #endif
			}else if (options.type == 3) {
				this.title="转发聊天";
				// 调用上一页的方法刷新 
				// #ifdef H5
				this.curMsg = beforePage.curMsg
				// #endif
				// #ifndef H5
				this.curMsg = beforePage.$vm.curMsg
				// #endif
			} else  if (options.type == 4) {
				this.title="选择提醒的人";
			} else {
				this.title="发起群聊";
			}


		},
		methods: {
			// 转发
			relay() {
				let user_ids = this.changeUser.map(it => {
					return it.id
				})
				if (!user_ids.length) {
					uni.showToast({
						title: "请选择至少一名人员",
						icon: "none"
					})
				} else if (user_ids.length > 5) {
					uni.showToast({
						title: "转发的人数不能超过5人！",
						icon: "none"
					})
				} else {
					var pages = getCurrentPages(); //当前页
					var beforePage = pages[pages.length - 2]; //上个页面
					let toContactId = ''
					let fromUser = ''
					// 调用上一页的方法刷新
					// #ifdef H5
					fromUser = beforePage.fromUser
					toContactId = beforePage.contact.id
					// #endif
					// #ifndef H5
					fromUser = beforePage.$vm.fromUser
					// #endif
					let selectedItem = ''
					if (this.type == 'relayCrm' || this.type == 'relayProject') {
						// #ifdef H5
						selectedItem = beforePage.message
						// #endif
						// #ifndef H5
						selectedItem = beforePage.$vm.message
						// #endif
					} else {
						selectedItem = uni.getStorageSync('selectedItem')
					}
					user_ids.forEach(it => {
						let msg = {
							id: this.$util.getUuid(),
							is_group: 0,
							fromUser,
							extends: selectedItem.extends ? selectedItem.extends : '',
							type: selectedItem.type,
							toContactId: it,
							content: selectedItem.content,
							sendTime: new Date().getTime()
						}

						this.$api.msgApi.sendMessage(msg)
							.then((res) => {
								if (res.code !== 200) return
								this.relayState = true
							})
					})
				}
			},
			// 添加群成员
			addGroupUser(user_ids) {
				var pages = getCurrentPages(); //当前页
				var beforePage = pages[pages.length - 2]; //上个页面
				// 调用上一页的方法刷新 
				// #ifdef H5
				beforePage.getsimpleMessage = false
				let group_id = beforePage.group_id
				// #endif
				// #ifndef H5
				beforePage.$vm.getsimpleMessage = false
				let group_id = beforePage.$vm.group_id
				// #endif
				this.$api.msgApi.addGroupUser({
					user_ids,
					id: group_id
				}).then(res => {
					if (res.code == 200) {
						uni.navigateBack()
					}
				})
			},
			// 添加群聊
			addGroup(user_ids) {
				this.$api.msgApi.addGroup({
					user_ids
				}).then(res => {
					if (res.code == 200) {
						setTimeout(() => {
							uni.navigateBack()
						}, 2000)



					}
				})
			},
			save(){
				uni.showLoading({
					title: '保存中...',
					mask:true
				});
				this.changeUser=this.$refs.userSelect.changeUser;
				this.selectUser=this.$refs.userSelect.selectUser;
				console.log(this.changeUser);
				if(!this.changeUser.length){
					return uni.showToast({
						title:'请选择人员',
						icon:'none'
					})
				}
				try{
					if(this.type==1){
						this.$api.msgApi.addGroup({user_ids:this.changeUser}).then(res =>{
							uni.hideLoading();
							const data = res.data;
							msgStore.appendContacts(data);
							uni.navigateTo({
								url:'/pages/message/chat?id='+data.id
							})
						})
					}else if(this.type==2){
						this.$api.msgApi.addGroupUser({user_ids:this.changeUser,id:this.contact_id}).then(res =>{
							this.closePage();
						})
					}else if(this.type==3){
						if(this.changeUser.length>5){
							return uni.showToast({
								title:'单次转发不能超过5人',
								icon:'none'
							})
						}
						this.$api.msgApi.forwardMessage({user_ids:this.changeUser,msg_id:this.curMsg.msg_id}).then(res =>{
							this.closePage();
						})
					}else if(this.type==4){
						const eventChannel = this.getOpenerEventChannel();
						eventChannel.emit('getAtList',this.selectUser);
						this.closePage();
					}else{
						this.$api.msgApi.removeUser({user_ids:this.changeUser,id:this.contact_id}).then(res =>{
							this.closePage();
						})
					}
				}catch(e){
					console.info(e);
					this.closePage();
				}
				
			},
			// 关闭加载动画返回上一个页面
			closePage(){
				uni.hideLoading();
				uni.navigateBack();
			},
			// 监听提交
			confirm: function(e) {
				let arr = []
				if (e) { //这个值为输入框输入的值
					var brr = this.userList.filter(value => {
						//遍历数组，返回值为true保留并复制到新数组，false则过滤掉
						let data = value.realname ? value.realname : value.userInfo.displayName
						if (data.includes(e.trim())) {
							arr.push(value)
						}
						return data.includes(e.trim());
					});
					this.lists = arr
				}

			},
		}
	}
</script>

<style lang="scss" scoped>
.group-bg{
	background-image: url(@/static/image/group.png);
}
.search-warp {
	width: 750rpx;
	padding: 15rpx 50rpx;
}
::v-deep .checklist-group{
	display: grid !important;
	.checklist-box{
		padding:20rpx;
		.checkbox__inner{
			width:40rpx !important;
			height:40rpx !important;
			overflow:hidden;
			.checkbox__inner-icon{
				position: absolute;
				top: -8px !important;
				left: -4px !important;
				height: 20px !important;
				width: 20px !important;
				border-right-width: 2px !important;
				border-bottom-width: 2px !important;
			}
		}
		.checklist-content{
			margin-left:20rpx;
			.checklist-text{
				font-size:36rpx !important;
			}
			
		}
	}
	.is-checked{
		.checkbox__inner{
			background-color: #18bc37 !important;
			border-color: #18bc37 !important;
		}
		.checklist-content{
			.checklist-text{
				color: #18bc37 !important;
			}
			
		}
	}
}
.footer-opt{
	position: fixed;
	bottom:0;
	left:0;
	width:100%;
}

.scroll-view_H {
	white-space: nowrap;
	width: 100%;
}

.user-list-avatar{
	float: left;
	margin-top:10rpx;
	.user-avatar{
		width:70rpx;
		height:70rpx;
		flex: 0 0 auto;
		border-radius: 8rpx;
		margin-left: 15rpx;
		display: inline-block;
		&:last-child{
			margin-right: 15rpx;
		}
	}
	.select-num{
		padding:10rpx;
	}
	
}
</style>
