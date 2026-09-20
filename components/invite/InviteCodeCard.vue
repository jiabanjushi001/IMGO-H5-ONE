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
		<view v-if="error" class="invite-code-error">暂时无法获取邀请码</view>
		<view v-else class="invite-code-value">{{ loading ? '······' : code }}</view>
		<view class="invite-code-bottom">
			<text>好友注册时填写，自动成为你的直属下级</text>
			<view v-if="error" class="invite-code-copy" @tap="emit('retry')">重试</view>
			<view v-else class="invite-code-copy" @tap="emit('copy-code')">复制</view>
		</view>
	</view>
</template>

<style scoped>
.invite-code-card { padding:29rpx 32rpx 27rpx; border:1rpx solid #e5eaf6; border-radius:31rpx; background:#fff; box-shadow:0 12rpx 32rpx rgba(39,55,109,.06); }
.invite-code-heading,.invite-code-bottom { display:flex; align-items:center; justify-content:space-between; gap:15rpx; }
.invite-code-heading { color:#303b57; font-size:25rpx; font-weight:700; }
.invite-code-heading .cuIcon-ticket { color:#7181ed; font-size:29rpx; }
.invite-code-value { margin:26rpx 0 21rpx; color:#405de2; font-size:68rpx; line-height:1.1; font-weight:800; letter-spacing:10rpx; font-family:monospace; }
.invite-code-error { margin:30rpx 0; color:#c25860; font-size:25rpx; }
.invite-code-bottom { color:#8b95a9; font-size:19rpx; }
.invite-code-bottom text { flex:1; }
.invite-code-copy { flex:none; padding:10rpx 20rpx; border-radius:16rpx; background:#eef1ff; color:#566ce7; font-size:21rpx; font-weight:700; }
</style>
