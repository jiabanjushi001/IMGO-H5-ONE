<template>
	<view class="auth-page">
		<view class="auth-orb auth-orb-one"></view>
		<view class="auth-orb auth-orb-two"></view>
		<view class="auth-shell">
			<view class="auth-brand">
				<view class="auth-logo-wrap">
					<image class="login-logo" :src="globalConfig.sysInfo.logo || './static/image/rocket.png'" mode="aspectFit"></image>
				</view>
				<view class="auth-brand-name">{{globalConfig.sysInfo.name ?? packData.name}}</view>
				<view class="auth-brand-copy">让每一次沟通，都简单而可靠</view>
			</view>

			<view class="auth-card">
				<view class="auth-heading">欢迎回来</view>
				<view class="auth-caption">登录后继续你的即时沟通</view>
				<form class="auth-form">
					<view class="cu-form-group auth-field">
						<view class="auth-field-icon cuIcon-people"></view>
						<view class="auth-field-body">
							<view class="title">账号</view>
							<input placeholder="请输入账号" maxlength="32" name="account" v-model="loginForm.account" />
						</view>
					</view>
					<view class="cu-form-group auth-field" v-if="!forget">
						<view class="auth-field-icon cuIcon-lock"></view>
						<view class="auth-field-body">
							<view class="title">密码</view>
							<input placeholder="请输入密码" maxlength="32" type="password" name="password" v-model="loginForm.password" />
						</view>
					</view>
					<view class="cu-form-group auth-field" v-else>
						<view class="auth-field-icon cuIcon-message"></view>
						<view class="auth-field-body">
							<view class="title">验证码</view>
							<input placeholder="请输入验证码" maxlength="6" name="code" v-model="loginForm.code" />
						</view>
						<button class="auth-code-btn" @tap="sendCode">获取验证码</button>
					</view>
				</form>

				<view class="forget auth-options">
					<view class="auth-remember"><switch class="switch" color="#526dff" :checked="loginForm.rememberMe" :class="loginForm.rememberMe?'checked':''" @change="switchChange" />记住我</view>
					<view class="auth-link" @tap="forget=!forget">{{forget ? '密码登录' : '忘记密码？'}}</view>
				</view>
				<button class="auth-primary-btn" @tap="login()">登录</button>
				<button v-if="canRegister" class="auth-secondary-btn" @tap="register()">注册账号</button>
			</view>

			<view class="auth-demo" v-if="globalConfig && globalConfig.demon_mode">
				<view class="auth-demo-title cuIcon-info"> 演示账号</view>
				<view>账号：13800000002～13800000020</view>
				<view>密码：123456</view>
			</view>
			<view class="footer-version">{{globalConfig.sysInfo.name ?? packData.name}} · v{{packData.version}}</view>
		</view>
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	import packageData from "../../package.json"
	const loginStore = useloginStore(pinia)
	export default {
		data() {
			return {
				loginForm:{
					account:'',
					password:'',
					code:'',
					client_id:'',
					rememberMe:false
				},
				forget:false,
				packData:packageData,
				globalConfig:loginStore.globalConfig
			}
		},
		computed: {
			canRegister() {
				return [1, 2].includes(Number(this.globalConfig?.sysInfo?.regtype))
			}
		},
		watch:{
			forget(val){
			  if(val){
				this.loginForm.password='123456';
			  }
			}
		},
		onLoad(options){
			// 检查是否有token,如果有就自动登录
			const token=options.token;
			if(token){
				this.doLogin({token:token});
			}
		},
		mounted() {
			if(this.globalConfig && this.globalConfig.demon_mode){
				const random = Math.floor(Math.random() * 19 + 2)
				this.loginForm.account=13800000000+random;
				this.loginForm.password='123456';
			}
			let LoginAccount=uni.getStorageSync('LoginAccount');
			if(LoginAccount){
				this.loginForm=LoginAccount;
			}
		},
		methods: {
			switchChange(e) {
				this.loginForm.rememberMe=e.detail.value;
			},
			sendCode(){
			  if(!this.loginForm.account){
				uni.showToast({
					title: '请输入账号！',
					icon: "none"
				});
				return false;
			  }
			  let data={
				account:this.loginForm.account,
				type:1
			  }
			  this.$api.LoginApi.sendCode(data).then((res)=>{
				  uni.showToast({
				  	title: res.msg,
				  	icon: "none"
				  });
			  })
			},
			register(){
				uni.navigateTo({
					url:"/pages/login/register"
				})
			},
			login(){
				if(this.loginForm.rememberMe){
					uni.setStorageSync('LoginAccount',this.loginForm);
				}else{
					uni.removeStorageSync('LoginAccount');
				}
				if(this.loginForm.account==""){
					uni.showToast({
						title: '请输入账号！',
						icon: "none"
					});
					return false;
				}
				if(this.loginForm.password==""){
					uni.showToast({
						title: '请输入密码！',
						icon: "none"
					});
					return false;
				}
				this.doLogin(this.loginForm);
				
			},
			doLogin(data){
				// WebSocket IDs are ephemeral and must not decide whether credentials can log in.
				const requestData = { ...data };
				delete requestData.client_id;
				this.$api.LoginApi.login(requestData).then(res => {
					if (res.code == 0) {
						uni.setStorageSync('authToken', res.data.authToken)
						let userInfo=res.data.userInfo;
						// 登录成功后绑定wss
						this.socketIo.send({
							type: "bindUid",
							user_id: userInfo.user_id,
							token:res.data.authToken
						});
						loginStore.login(userInfo);
						uni.reLaunch({
							url: '/pages/index/index'
						})
					}
				})
			}
		}
	}
</script>

<style scoped>
	.login-logo {
		width: 112rpx;
		height: 112rpx;
		border-radius: 28rpx;
	}
	.remark-title{
		font-weight: 600;
	}
</style>
