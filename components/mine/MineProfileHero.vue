<script setup>
import { computed, ref, watch } from 'vue'
import { resolveAvatarDisplayUrl, normalizeAvatarUrl } from '@/utils/avatar.js'

const props = defineProps({
	user: { type: Object, default: () => ({}) },
	circleAvatar: { type: Boolean, default: false }
})
const emit = defineEmits(['edit'])
const displayName = computed(() => props.user.realname || props.user.account || '我的账号')
const account = computed(() => props.user.account || '—')
const displayAvatar = ref('')
const initial = computed(() => displayName.value.slice(0, 1))

const refreshAvatar = (user) => {
	const normalized = normalizeAvatarUrl(user && user.avatar, user || {})
	displayAvatar.value = normalized
	resolveAvatarDisplayUrl(user && user.avatar, user || {}).then((url) => {
		const current = normalizeAvatarUrl(props.user && props.user.avatar, props.user || {})
		if (current === normalized) {
			displayAvatar.value = url
		}
	}).catch(() => {})
}

watch(() => props.user, (value) => refreshAvatar(value), { immediate: true, deep: true })
</script>

<template>
	<view class="mine-hero" @tap="emit('edit')">
		<view class="mine-hero-main">
			<view class="mine-hero-avatar" :class="{ 'is-round': circleAvatar }">
				<image v-if="displayAvatar" :src="displayAvatar" mode="aspectFill" class="mine-hero-avatar-image" />
				<text v-else class="mine-hero-avatar-initial">{{ initial }}</text>
			</view>
			<view class="mine-hero-identity">
				<text class="mine-hero-name">{{ displayName }}</text>
				<text class="mine-hero-account">账号 · {{ account }}</text>
			</view>
		</view>
		<view class="mine-hero-footer">
			<text>个人资料</text>
			<view class="mine-hero-footer-link"><text>查看与编辑</text><text class="cuIcon-right"></text></view>
		</view>
	</view>
</template>

<style scoped>
.mine-hero { position:relative; overflow:hidden; border-radius:36rpx; background:linear-gradient(126deg,#4c63e9 0%,#5e70f5 55%,#8068ed 100%); box-shadow:0 22rpx 48rpx rgba(65,79,177,.2); color:#fff; }
.mine-hero::before { content:''; position:absolute; width:360rpx; height:360rpx; right:-148rpx; top:-166rpx; border:48rpx solid rgba(255,255,255,.07); border-radius:50%; }
.mine-hero::after { content:''; position:absolute; width:230rpx; height:230rpx; left:-100rpx; bottom:-182rpx; background:rgba(255,255,255,.07); border-radius:50%; }
.mine-hero-main { position:relative; z-index:1; display:flex; align-items:center; gap:22rpx; padding:34rpx 32rpx 30rpx; }
.mine-hero-avatar { flex:none; display:flex; align-items:center; justify-content:center; width:116rpx; height:116rpx; overflow:hidden; border:5rpx solid rgba(255,255,255,.75); border-radius:30rpx; background:rgba(255,255,255,.18); box-shadow:0 12rpx 32rpx rgba(35,45,115,.18); }
.mine-hero-avatar.is-round { border-radius:50%; }
.mine-hero-avatar-image { width:100%; height:100%; display:block; }
.mine-hero-avatar-initial { font-size:48rpx; font-weight:750; }
.mine-hero-identity { display:flex; flex:1; min-width:0; flex-direction:column; gap:7rpx; }
.mine-hero-name { overflow:hidden; white-space:nowrap; text-overflow:ellipsis; font-size:35rpx; font-weight:750; line-height:1.3; }
.mine-hero-account { overflow:hidden; white-space:nowrap; text-overflow:ellipsis; color:rgba(255,255,255,.78); font-size:23rpx; }
.mine-hero-footer { position:relative; z-index:1; display:flex; align-items:center; justify-content:space-between; padding:19rpx 32rpx 22rpx; border-top:1rpx solid rgba(255,255,255,.18); color:rgba(255,255,255,.72); font-size:22rpx; }
.mine-hero-footer-link { display:flex; align-items:center; gap:9rpx; color:#fff; }
.mine-hero-footer-link .cuIcon-right { font-size:19rpx; }
</style>
