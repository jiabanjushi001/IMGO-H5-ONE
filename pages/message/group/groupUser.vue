<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>群成员</template>
		</cu-custom>
		<view class="cu-bar bg-white search fixed" :style="[{top:CustomBar + 'px'}]">
			<view class="search-form round">
				<text class="cuIcon-search"></text>
				<input type="text" v-model="keywords" placeholder="输入搜索的关键词" confirm-type="search"/>
			</view>
		</view>
		<view style="margin-top: 104rpx;">
			<view class="cu-list menu-avatar">
				<view class="cu-item" v-for="item in userList">
					<view class="cu-avatar round lg" :style="'background-image:url('+item.userInfo.avatar+');'" @tap="openChatDetail(item)"></view>
					<view class="content">
						<view class="text-grey">
							<view class="text-cut">{{item.userInfo.displayName}}</view>
							<view v-if="item.role<3" class="cu-tag round sm" :class="item.role==1 ? 'bg-red' : 'bg-orange'">{{item.role ==1?"群主":item.role==2?'管理员':''}}</view>
							<view v-if="item.user_id==userInfo.user_id" class="cu-tag round sm">我</view>
							
						</view>
						<view v-if="noSpeakExp(item.no_speak_time)" class="text-red text-xs">禁言至：{{noSpeakExp(item.no_speak_time)}}</view>
					</view>
					<view class="action" @tap="openModel(item)">
						<view class="text-grey text-sm"> <text class="cuIcon-more f-24" v-if="item.role>1 && isAuth" ></text></view>
					</view>
				</view>
				<Empty v-if="!userList.length" noDatatext="未搜索到数据" textcolor="#999" ></Empty>
			</view>
		</view>
		<view class="cu-modal bottom-modal" :class="modelName=='userOpt'?'show':''">
			<view class="cu-dialog">
				<view class="manage-content">
					<view class="cu-list menu bg-white">
						<view class="cu-item" v-if="curUser">
							<view class="content im-flex im-justify-content-center im-align-items-center">
								<view class="cu-avatar round sm" :style="'background-image:url('+(curUser.avatar)+');'"></view>
								<view class="text-cut ml-5">{{curUser.realname}}</view>
								<view v-if="curUser.role==2" class="cu-tag round sm bg-orange">管理员</view>
							</view>
						</view>
						<view class="cu-item"  @tap="changeOwner()" v-if="isAdmin">
							<view class="content padding-tb-sm">
								<text class="c-orange">转让管理权限</text>
							</view>
						</view>
						<view class="cu-item"  @tap="setManage()" v-if="isAdmin">
							<view class="content padding-tb-sm">
								<text>{{curUser.role==2 ? '取消管理员' : '设为管理员'}}</text>
							</view>
						</view>
						<view class="cu-item"  @tap="showNoSpeak()" v-if="isAdmin">
							<view class="content padding-tb-sm">
								<text>设置禁言</text>
							</view>
						</view>
						<view class="cu-item" @tap="removeUser()">
							<view class="content padding-tb-sm">
								<text>移出群聊</text>
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
		<uni-popup ref="nospeak" type="share" safeArea backgroundColor="#fff">
			<uni-list>
				<uni-list-item clickable v-for="(item,index) in noSpeakList" :title="item.name" @click="setTime(item.id)" >
					<template v-slot:footer>
						<text class="cuIcon-check c-green" v-if="noSpeakTimer==item.id"></text>
					</template>
				</uni-list-item>
				<uni-list-item title="自定义" clickable @click="setTime(0)" >
					<template v-slot:footer>
						<uni-number-box :min="1" :max="365" :value="noSpeakDay" />&nbsp;天
					</template>
				</uni-list-item>
			</uni-list>
			<view class="padding flex flex-direction mt-10">
				<button class="cu-btn bg-green lg" @tap="setNoSpeak()">确定</button>
			</view>
		</uni-popup>
	</view>
</template>
<script>
	import { useMsgStore } from '@/store/message';
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
	import { useloginStore } from '@/store/login';
	const userStore = useloginStore(pinia)
	export default {
		data() {
			return {
				keywords:'',
				group_id: '',
				modelName:'',
				userList: [],
				allUser:[],
				isAdmin:false,
				isManage:false,
				isAuth:false,
				noSpeakTimer:0,
				noSpeakDay:1,
				userInfo:userStore.userInfo,
				curUser:{},  //选中的用户
				noSpeakList:[{
					name:'10分钟',
					id:1
				},{
					name:'1小时',
					id:2
				},{
					name:'3小时',
					id:3
				},{
					name:'1天',
					id:4
				}]
			}
		},
		watch:{
			keywords(val){
				if(val==''){
					this.userList=this.allUser;
				}else{
					this.search();
				}
			},
		},
		onLoad(options) {
			this.group_id = options.group_id?options.group_id:''
			this.getGroupuserlist()

		},
		methods: {
			getGroupuserlist() {
				this.userList = []
				this.$api.msgApi.groupUserList({
					group_id: this.group_id
				}).then(res => {
					const admin=res.data.filter(item => item.role == 1 && item.userInfo.id== this.userInfo.user_id)
					if(admin.length) this.isAdmin=true;
					const manage=res.data.filter(item => item.role == 2 && item.userInfo.id== this.userInfo.user_id)
					if(manage.length) this.manage=true;
					if(admin.length || manage.length) this.isAuth=true;
					const allUser=res.data;
					allUser.forEach((item)=>{
						item.realname=item.userInfo.displayName;
						item.name_py=item.userInfo.name_py;
					})
					this.allUser=allUser;
					this.userList = res.data;
				})
			},
			openModel(item){
				item.realname=item.userInfo.displayName;
				item.avatar=item.userInfo.avatar;
				this.curUser=item;
				this.modelName='userOpt';
			},
			// 设置取消管理员
			setManage(){
				const role=this.curUser.role==2 ? 3 : 2;
				this.$api.msgApi.setManager({
					id:this.group_id,
					user_id:this.curUser.user_id,
					role:role
				}).then((res)=>{
					if(res.code==0){
						this.getGroupuserlist();
					}
					this.modelName='';
				})
			},
			// 移出成员
			removeUser(){
				this.modelName='';
				uni.showModal({
					title: '确定要删除该成员吗?',
					success: e => {
						if (e.confirm) {
							this.$api.msgApi.removeUser({
								id:this.group_id,
								user_id:this.curUser.user_id,
							}).then((res)=>{
								if(res.code==0){
									this.getGroupuserlist();
								}
							})
						}
					}
				});
			},
			changeOwner(){
				this.modelName='';
				uni.showModal({
					title: '确定将管理权限转移给该成员吗?',
					success: e => {
						if (e.confirm) {
							this.$api.msgApi.changeOwner({
								id:this.group_id,
								user_id:this.curUser.user_id,
							}).then((res)=>{
								if(res.code==0){
									uni.reLaunch({
										url:'/pages/index/index'
									})
								}
							})
						}
					}
				});
			},
			search(){
				const allUser=JSON.parse(JSON.stringify(this.allUser));
				this.userList=this.$util.searchObject(allUser,['realname','name_py'],this.keywords);
			},
			openChatDetail(item){
				if(this.userInfo.user_id==item.user_id) return;
				let friend=msgStore.getContact(item.user_id);
				let curContact=msgStore.getContact(this.group_id);
				if(curContact.role<3 || curContact.setting.profile=='1' || friend){
					uni.navigateTo({
						url:"/pages/contacts/detail?id="+item.user_id
					})
				}else{
					uni.showToast({
						title:'已开启用户隐私！',
						icon:'none'
					})
					return false;
				}
			},
			showNoSpeak(){
				this.modelName=''
				this.$refs.nospeak.open()
			},
			setTime(val){
				console.log(val);
				this.noSpeakTimer=val;
			},
			setNoSpeak(){
				this.$refs.nospeak.close();
				this.$api.msgApi.setNoSpeak({
					id:this.group_id,
					user_id:this.curUser.user_id,
					noSpeakDay:this.noSpeakDay,
					noSpeakTimer:this.noSpeakTimer
				}).then((res)=>{
					this.noSpeakTimer=0;
					this.noSpeakDay=1;
					if(res.code==0){
						this.getGroupuserlist();
					}
				})
			},
			noSpeakExp(time){
				if(time * 1000>new Date().getTime()){
					return this.$util.date('m-d H:i',time);
				}else{
					return false;
				}
			}
		}
	}
</script>
<style scoped>
	.list-image {
		width: 80rpx;
		height: 80rpx;
		font-size: 0;
	}
	.share {
			/* #ifndef APP-NVUE */
			display: flex;
			/* #endif */
			flex-direction: column;
		}
		.uni-popup{
			z-index:9999 !important;
		}
</style>
