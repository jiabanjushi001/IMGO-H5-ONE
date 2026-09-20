<template>
	<view class="auth-page auth-register-page">
		<view class="auth-orb auth-orb-one"></view>
		<view class="auth-orb auth-orb-two"></view>
		<cu-custom bgColor="auth-nav" :isBack="true" class="auth-register-nav">
			<template #backText></template>
			<template #content>创建账号</template>
		</cu-custom>
		<view class="auth-shell auth-register-shell">
			<view class="auth-brand auth-brand-compact">
				<view class="auth-logo-wrap">
					<image class="login-logo" :src="globalConfig.sysInfo.logo || './static/image/rocket.png'" mode="aspectFit"></image>
				</view>
				<view class="auth-brand-name">加入 {{globalConfig.sysInfo.name ?? packData.name}}</view>
				<view class="auth-brand-copy">创建账号，开启你的即时沟通</view>
			</view>

			<view class="auth-card auth-register-card">
				<form class="auth-form">
					<view class="cu-form-group auth-field">
						<view class="auth-field-icon cuIcon-profile"></view>
						<view class="auth-field-body">
							<view class="title">账号</view>
							<input :placeholder="placeholder" maxlength="32" name="account" v-model="regForm.account" @input="handleInput" />
						</view>
					</view>
					<view class="cu-form-group auth-field">
						<view class="auth-field-icon cuIcon-people"></view>
						<view class="auth-field-body">
							<view class="title">昵称</view>
							<input placeholder="请输入用户名或昵称" maxlength="32" name="realname" v-model="regForm.realname" />
						</view>
					</view>
					<view class="cu-form-group auth-field">
						<view class="auth-field-icon cuIcon-ticket"></view>
						<view class="auth-field-body">
							<view class="title">邀请码{{ requiresInvitation ? '（必填）' : '（选填）' }}</view>
							<input placeholder="请输入 6 位数字" type="tel" maxlength="6" name="inviteCode" :value="regForm.inviteCode" @input="handleInviteInput" />
						</view>
					</view>
					<view v-if="legacyInviteToken" class="auth-invite-hint">已识别专属邀请链接，也可以填写新的邀请码</view>
					<view class="cu-form-group auth-field" v-if="requiresVerification">
						<view class="auth-field-icon cuIcon-message"></view>
						<view class="auth-field-body">
							<view class="title">验证码</view>
							<input placeholder="请输入验证码" maxlength="6" name="code" v-model="regForm.code" />
						</view>
						<button class="auth-code-btn" @tap="sendCode">获取验证码</button>
					</view>
					<view class="cu-form-group auth-field">
						<view class="auth-field-icon cuIcon-lock"></view>
						<view class="auth-field-body">
							<view class="title">密码</view>
							<input placeholder="请输入 6～30 位密码" maxlength="30" type="password" name="password" v-model="regForm.password" />
						</view>
					</view>
					<view class="cu-form-group auth-field">
						<view class="auth-field-icon cuIcon-roundcheck"></view>
						<view class="auth-field-body">
							<view class="title">确认密码</view>
							<input placeholder="请再次输入密码" maxlength="30" type="password" name="repass" v-model="regForm.repass" />
						</view>
					</view>
				</form>
				<button class="auth-primary-btn" :loading="submitting" :disabled="submitting" @tap="login()">注册并登录</button>
				<view class="auth-login-tip" @tap="goLogin">已有账号？<text>返回登录</text></view>
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
				regForm:{
					account:'',
					realname:'',
					password:'',
					repass:'',
					code:'',
					inviteCode:''
				},
				placeholder:'请输入账号：4-32个字符',
				forget:false,
				submitting:false,
				legacyInviteToken:'',
				packData:packageData,
				globalConfig:loginStore.globalConfig
			}
		},
		computed:{
			requiresVerification(){
				return Boolean(parseInt(this.globalConfig.sysInfo.regauth))
			},
			requiresInvitation(){
				return Number(this.globalConfig.sysInfo.regtype) === 2
			}
		},
		onLoad(options){
			const incoming = typeof options?.inviteCode === 'string' ? options.inviteCode.trim() : ''
			if (/^[0-9]{6}$/.test(incoming)) this.regForm.inviteCode = incoming
			else this.legacyInviteToken = incoming
		},
		watch:{
			forget(val){
			  if(val){
				this.regForm.password='123456';
			  }
			}
		},
		mounted() {
			let regauth=this.globalConfig.sysInfo.regauth ?? 0;
		    if(regauth==1){
				this.placeholder='请输入手机号';
		    }else if(regauth==2){
				this.placeholder='请输入邮箱账号';
		    }else if(regauth==3){
				this.placeholder='请输入手机号/邮箱';
		    }
		},
		methods: {
			goLogin(){
				uni.navigateBack({ delta: 1 });
			},
			handleInput(event) {
			  let value = event.detail.value;
			  let filteredValue = value.replace(/[\u4e00-\u9fa5]/g, '');
			  this.regForm.account = filteredValue;
			},
			handleInviteInput(event) {
				this.regForm.inviteCode = String(event.detail.value || '').replace(/[^0-9]/g, '').slice(0, 6)
				return this.regForm.inviteCode
			},
			sendCode(){
			  if(!this.regForm.account){
				uni.showToast({
					title: '请输入账号！',
					icon: "none"
				});
				return false;
			  }
			  let data={
				account:this.regForm.account,
				type:2
			  }
			  this.$api.LoginApi.sendCode(data).then((res)=>{
				  uni.showToast({
				  	title: res.msg,
				  	icon: "none"
				  });
			  })
			},
			async login(){
				if(this.submitting) return false
				if(this.regForm.account==""){
					uni.showToast({
						title: '请输入账号！',
						icon: "none"
					});
					return false;
				}
				if(this.regForm.realname==""){
					uni.showToast({
						title: '请输入用户名或者昵称！',
						icon: "none"
					});
					return false;
				}
				if(this.regForm.password.length<6 || this.regForm.password.length>30){
					uni.showToast({
						title: '请输入6-30位密码！',
						icon: "none"
					});
					return false;
				}
				if(this.regForm.password!=this.regForm.repass){
					uni.showToast({
						title: '两次密码输入不相同！',
						icon: "none"
					});
					return false;
				}
				const inviteCode = this.regForm.inviteCode.trim()
				if(inviteCode && !/^[0-9]{6}$/.test(inviteCode)){
					uni.showToast({ title: '邀请码必须是 6 位数字', icon: 'none' })
					return false
				}
				if(this.requiresInvitation && !inviteCode && !this.legacyInviteToken){
					uni.showToast({ title: '请输入邀请码', icon: 'none' })
					return false
				}
				const payload={
					...this.regForm,
					code:this.requiresVerification ? this.regForm.code : '',
					inviteCode:inviteCode || this.legacyInviteToken
				}
				this.submitting = true
				let registered = false
				try {
					const registration = await this.$api.LoginApi.register(payload)
					if(registration?.code != 0) return
					registered = true
					const res = await this.$api.LoginApi.login({
						account: this.regForm.account,
						password: this.regForm.password
					})
					if(res?.code != 0 || !res.data?.authToken || !res.data?.userInfo){
						throw new Error('自动登录失败')
					}
					uni.setStorageSync('authToken', res.data.authToken)
					uni.removeStorageSync('LoginAccount')
					const userInfo = res.data.userInfo
					this.socketIo.send({
						type: 'bindUid',
						user_id: userInfo.user_id,
						token: res.data.authToken
					})
					loginStore.login(userInfo)
					uni.reLaunch({ url: '/pages/index/index' })
				} catch (error) {
					if(registered){
						uni.showModal({
							title: '注册已成功',
							content: '自动登录失败，请手动登录',
							showCancel: false,
							success: () => uni.reLaunch({ url: '/pages/login/index' })
						})
					}
				} finally {
					this.submitting = false
				}
			},
		}
	}
</script>

<style scoped>
	.login-logo {
		width: 96rpx;
		height: 96rpx;
		border-radius: 24rpx;
	}
	.auth-invite-hint { margin:-1rpx 12rpx 13rpx; color:#7180a8; font-size:20rpx; }
</style>
