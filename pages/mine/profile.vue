<template>
	<view class="profile-page">
		<cu-custom bgColor="text-white" bgStyle="background:linear-gradient(126deg,#4c63e9 0%,#5e70f5 55%,#6b6eeb 100%);color:#fff;" :isBack="true">
			<template #backText></template>
			<template #content>个人信息</template>
		</cu-custom>
		<form>
			<view class="cu-form-group" style="height:140rpx">
				<view class="title">头像</view>
				<view class="im-flex im-align-items-center">
					<avatar
					       selWidth="240px" selHeight="480upx" expWidth="240px"  expHeight="240px" @upload="uploadAvatar" :avatarSrc="userInfo.avatar"
					       avatarStyle="width: 100rpx; height: 100rpx; border-radius: 100%;" :quality="1">
					   </avatar>
					<text class="cuIcon-right ml-10 f-18 text-grey"></text></view>
			</view>
			<view class="cu-form-group">
				<view class="title">账号</view>
				<view class="text-gray">{{userInfo.account}}</view>
			</view>
			<view class="cu-form-group">
				<view class="title">{{globalConfig.sysInfo.runMode==1 ? "姓名" : '昵称'}}</view>
				<view class="text-gray" v-if="globalConfig.sysInfo.runMode==1 && globalConfig.sysInfo.diyName!=1">{{userInfo.realname}}</view>
				<input class="uni-input" style="text-align: right;" v-if="globalConfig.sysInfo.runMode==2 || globalConfig.sysInfo.diyName==1" v-model="userInfo.realname" placeholder="请输入昵称" />
			</view>
			<view class="cu-form-group">
				<view class="title">e-mail</view>
				<input class="uni-input" style="text-align: right;" v-model="userInfo.email" placeholder="请输入email地址" />
			</view>
			<view class="cu-form-group">
				<view class="title">性别</view>
				<view>
					<radio-group @change="changeSex">
					<label class="radio mr-10" v-for="x in sexList"><radio name="sex" :value="x.id" :checked="userInfo.sex==x.id" color="#4c63e9" /> {{x.name}}</label>
					</radio-group>
				</view>
			</view>
			<view class="cu-form-group align-start">
				<view class="title">个性签名</view>
				<textarea maxlength="-1" v-model="userInfo.motto" placeholder="请输入个性签名"></textarea>
			</view>
		</form>
		<view class="padding flex flex-direction">
			<button class="cu-btn theme-btn lg" :class="{ 'is-disabled': saved }" :disabled="saved" @tap="saveInfo()" >保存</button>
		</view>
		<view class="padding flex flex-direction" style="position: fixed;bottom: 0;width:100%;align-items: center;">
			<text class="text-red" @tap="logout()">注销登录</text>
		</view>
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	import avatar from "@/components/yq-avatar/yq-avatar.vue";
	const loginStore = useloginStore(pinia)
	export default {
		components: {
			avatar
		},
		data() {
			return {
				loginStore:loginStore,
				globalConfig:loginStore.globalConfig,
				userInfo:{},
				sexList:[
					{
						id:'2',
						name:'未知'
					},
					{
						id:'1',
						name:'男'
					},
					{
						id:'0',
						name:'女'
					},
				],
				saved:false
			}
		},
		mounted() {
			this.userInfo=JSON.parse(JSON.stringify(loginStore.userInfo));
		}, 
		methods: {
			logout(){
				let client_id=uni.getStorageSync('client_id');
				this.$api.LoginApi.logout({client_id:client_id}).then(res => {
					if (res.code == 0) {
						loginStore.logout()
					}
				})
				
			},
			textareaBInput(e) {
				this.userInfo.motto = e.detail.value
			},
			changeSex(evt){
				this.userInfo.sex=parseInt(evt.detail.value);
			},
			saveInfo(){
				if(this.userInfo.realname == ''){
					uni.showToast({
						title:'请输入昵称',
						icon:'none'
					})
					return false
				  }
				  let pattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
				  if(this.userInfo.email!='' && !pattern.test(this.userInfo.email)){
					  uni.showToast({
					  	title:'请输入正确的邮箱',
					  	icon:'none'
					  })
					  return false
				  }
				  // 防止无限次点击
				  this.saved=true;
				  let params={
					realname:this.userInfo.realname,
					email:this.userInfo.email,
					sex:this.userInfo.sex,
					motto:this.userInfo.motto
				  }
				  this.$api.msgApi.updateUserInfo(params).then(res=>{
					if(res.code == 0){
					  uni.showToast({
					  	title:'保存成功',
					  	icon:'none'
					  })
					  let data=JSON.parse(JSON.stringify(this.userInfo))
					  loginStore.login(data)
					}
				  })
				  // 8秒后可以重新保存
				  setTimeout(()=>{
				  	this.saved=false;
				  },8000)
			},
			setAvatar(){
				uni.navigateTo({
					url:"/pages/mine/avatar"
				})
			},
			uploadAvatar(res){
				uni.showLoading({
					title:'上传中...'
				})
				uni.uploadFile({
					url: this.$api.msgApi.uploadAvatar,
					filePath: res.path,
					name: 'file',
					header: {
						'Authorization': uni.getStorageSync('authToken'),
					},
					formData: {
						ext: 'png'
					},
					success: (e) => {
						uni.hideLoading();
						let res=JSON.parse(e.data);
						if(res.code==0){
							uni.showToast({
								title:res.msg,
								icon:'none'
							})
							this.userInfo.avatar=res.data
							let data=JSON.parse(JSON.stringify(this.userInfo));
							loginStore.login(data);
						}
					},
					fail: (res) => {
						uni.hideLoading();
					}
				})
			}
		}
	}
</script>

<style>
.profile-page {
	min-height: 100vh;
	background: #f4f6fb;
}
.mine-theme-header .cu-bar {
	background: linear-gradient(126deg, #4c63e9 0%, #5e70f5 55%, #6b6eeb 100%) !important;
	color: #fff;
}
.mine-theme-header .action,
.mine-theme-header .content,
.mine-theme-header .cuIcon-back {
	color: #fff !important;
}
.theme-btn {
	background: linear-gradient(126deg, #4c63e9 0%, #5e70f5 100%) !important;
	color: #fff !important;
	border: 0 !important;
	box-shadow: 0 12rpx 28rpx rgba(76, 99, 233, .28);
}
.theme-btn.is-disabled,
.theme-btn[disabled] {
	opacity: .55;
	box-shadow: none;
	border: solid 1px #dbdada !important;
}
</style>
