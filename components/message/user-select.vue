<template>
	<view>
		<view class="cu-bar bg-white search">
			<view class="search-form round">
				<text class="cuIcon-search"></text>
				<input type="text" v-model="keywords" placeholder="输入搜索的关键词" confirm-type="search"/>
			</view>
		</view>

		<view class="gui-padding" style="padding-bottom: 120rpx;" v-if="!keywords">
			<view class="cu-list  menu">
				<view class="cu-item arrow"  v-if="isAuth" @tap="atAll">
					<view class='cu-avatar mr-15 group-bg sm'>
					</view>
					<view class="content">
						<text class="c-333">所有人</text>
					</view>
				</view>
			</view>
			<uni-data-checkbox :multiple="multiple" class="user-list-select" @change="chooseUser" v-model="changeUser" :localdata="lists"></uni-data-checkbox>
			<Empty v-if="!lists.length" noDatatext="无联系人" textcolor="#999" ></Empty>
		</view>
		<view class="gui-padding" style="padding-bottom: 120rpx;"  v-if="keywords">
			<uni-data-checkbox :multiple="multiple" class="user-list-select" @change="chooseSearchUser" v-model="searcheUser" :localdata="searchList"></uni-data-checkbox>
			<Empty v-if="!searchList.length" noDatatext="未搜索到数据" textcolor="#999" ></Empty>
		</view>
		<view class="cu-bar bg-white tabbar border shop footer-opt">
		    <scroll-view class="scroll-view_H" scroll-x="true"  :scroll-anchoring="true" :scroll-left="scrollLeft">
		    	<view class="user-list-avatar">
					<template v-for="(item,index) in selectUser" :key="index">
						<AuthImage class="user-avatar" :src="item.avatar" :info="item" avatar mode="aspectFill" @tap="removeUser(item.id,item.user_id)"></AuthImage>
					</template>
				</view>
				<view class="select-num pd-10">已选{{selectUser.length}}人</view>
		    </scroll-view>
		</view>
	</view>
</template>

<script>
	import { storeToRefs } from 'pinia';
	import pinia from '@/store/index'
	import { useloginStore } from '@/store/login';
	const userStore = useloginStore(pinia)
	export default {
		name  : "user-select",
		props : { 
			type:{type:Number, default:0},//1创建群聊，2邀请群成员，3：转发，4：提及某人，5：选择联系人
			contact_id:{type:String, default:''},
			multiple:{type:Boolean, default:true},
			user_ids:{type:Object, default:{}}
		},
		data() {
			return {
				lists: [],
				keywords:'',
				searchList:[],
				selectUser:[],
				userList: [],
				changeUser: [], //选中的数据
				scrollLeft:300,
				contacList:[],
				searcheUser:[], //搜索选中人员
				isAuth:false,
				userInfo:userStore.userInfo,
			}
		},
		watch: {
			keywords(val){
				this.search();
				// 如果关键词为空,搜索选中的人员为已经选中的人员
				if(val!=''){
					this.searcheUser=this.changeUser;
				}
			},
			searcheUser(val){
				const arr=this.changeUser.concat(val);
				this.changeUser=[...new Set(arr)];
			}
		},
		mounted() {
			 if (this.type == 4) {
				this.getGroupUser()
			} else {
				this.getAllUser()
			}
			let lists=JSON.parse(JSON.stringify(this.userList));
			this.lists=lists;

		},
		methods: {
			getAllUser() {
				const allContact=uni.getStorageSync('allContacts');
				let contact=[];
				if(this.type==3){
					contact = allContact.filter((item)=>{
						return item.id!=this.contact_id;
					})
					contact.forEach(item => {
						item.disable = false;
						let name=item.displayName;
						if(item.is_group){
							name+="（群聊）";
						}
						item.text=name;
						item.value=item.id;
					})
				}else{
					contact = allContact.filter((item)=>{
						return item.is_group==0;
					})
					this.contacList=JSON.parse(JSON.stringify(contact));
					contact.forEach(item => {
						item.disable = false;
						item.text=item.displayName;
						item.value=item.id;
						if (this.user_ids.length && this.type==2) {
							if (this.user_ids.includes(item.id)) {
								item.disable = true
							}
						}
					})
					if(this.type==1 && this.contact_id){
						this.changeUser.push(parseInt(this.contact_id));
						this.selectUser=contact.filter((item)=>{
							return item.id==this.contact_id;
						})
					}
				}
				this.userList = contact
				this.lists = contact
			},
			getGroupUser(){
				this.userList = []
				this.$api.msgApi.groupUserList({
					group_id: this.contact_id
				}).then(res => {
					const isAuth=res.data.filter(item => (item.role == 1 || item.role == 2)  && item.userInfo.id== this.userInfo.user_id)
					if(isAuth.length) this.isAuth=true;
					const allUser=res.data;
					allUser.forEach((item,index)=>{
						item.realname=item.userInfo.displayName;
						item.displayName=item.userInfo.displayName;
						item.avatar=item.userInfo.avatar;
						item.name_py=item.userInfo.name_py;
						item.disable = false;
						item.text=item.userInfo.displayName;
						item.value=item.userInfo.id;
					})
					const userList=res.data.filter(item => item.userInfo.id != this.userInfo.user_id)
					this.lists=userList;
					this.userList = userList;
				})
			},
			// 选择人员
			chooseUser(e){
				if(this.multiple){
					this.selectUser=e.detail.data;
				}else{
					this.selectUser=[e.detail.data];
				}
				setTimeout(()=>{
					this.scrollLeft+=300;
				},50)
			},
			// 提醒所有人
			atAll(){
				this.$emit('setData',{realname:'所有人',user_id:0});
			},
			// 选择搜索列表的成员
			chooseSearchUser(e){
				if(this.multiple){
					this.selectUser=this.$util.mergeArrays(e.detail.data,this.selectUser);
				}else{
					this.selectUser=[e.detail.data];
				}
				setTimeout(()=>{
					this.scrollLeft+=300;
				},50)
			},
			// 移除成员
			removeUser(id,user_id){
				this.selectUser=this.selectUser.filter((e)=>{
					return e.id!=id;
				})
				this.changeUser=this.changeUser.filter((e)=>{
					return e!=user_id;
				})
			},
			// 搜索成员
			search: function(e) {
				let contact=JSON.parse(JSON.stringify(this.lists));
				this.searchList=this.$util.searchObject(contact,['displayName','name_py'],this.keywords);
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

<style lang="scss">
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
			.radio__inner{
				border-color:#18bc37 !important;
				.radio__inner-icon{
					background-color: #18bc37 !important;
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
