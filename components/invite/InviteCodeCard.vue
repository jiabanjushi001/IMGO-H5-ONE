<script setup>
defineProps({
	code: { type: String, default: '' },
	loading: { type: Boolean, default: false },
	error: { type: Boolean, default: false }
})
const emit = defineEmits(['copy-code', 'retry'])
</script>

<template>
	<view class="invite-code-card">
		<view class="invite-code-heading"><text>我的专属邀请码</text><text class="cuIcon-ticket"></text></view>
		<view class="invite-code-body">
			<view class="invite-code-main">
				<view v-if="error" class="invite-code-error">暂时无法获取邀请码</view>
				<view v-else class="invite-code-value">{{ loading ? '······' : code }}</view>
				<text class="invite-code-hint">好友注册时填写，自动成为你的直属下级</text>
			</view>
			<view v-if="error" class="invite-code-copy" @tap="emit('retry')">重试</view>
			<view v-else class="invite-code-copy" @tap="emit('copy-code')">复制</view>
		</view>
	</view>
</template>

<style scoped>
.invite-code-card { padding:29rpx 32rpx 27rpx; border:1rpx solid #e5eaf6; border-radius:31rpx; background:#fff; box-shadow:0 12rpx 32rpx rgba(39,55,109,.06); }
.invite-code-heading { display:flex; align-items:center; justify-content:space-between; gap:15rpx; color:#303b57; font-size:25rpx; font-weight:700; }
.invite-code-heading .cuIcon-ticket { color:#7181ed; font-size:29rpx; }
.invite-code-body { display:flex; align-items:stretch; gap:24rpx; margin-top:22rpx; }
.invite-code-main { flex:1; min-width:0; display:flex; flex-direction:column; justify-content:center; }
.invite-code-value { color:#405de2; font-size:68rpx; line-height:1.1; font-weight:800; letter-spacing:10rpx; font-family:monospace; }
.invite-code-error { color:#c25860; font-size:25rpx; }
.invite-code-hint { margin-top:18rpx; color:#8b95a9; font-size:19rpx; line-height:1.45; }
.invite-code-copy {
	flex:none;
	display:flex;
	align-items:center;
	justify-content:center;
	align-self:stretch;
	width:148rpx;
	min-height:148rpx;
	border-radius:24rpx;
	background:linear-gradient(135deg,#4167e6 0%,#5e6df0 50%,#8264e8 100%);
	box-shadow:0 12rpx 28rpx rgba(65,103,230,.32);
	color:#fff;
	font-size:30rpx;
	font-weight:750;
}
</style>
