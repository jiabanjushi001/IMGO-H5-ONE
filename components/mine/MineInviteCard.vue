<script setup>
import { computed } from 'vue'

const props = defineProps({
	directCount: { type: Number, default: -1 }
})
const emit = defineEmits(['open'])
const subtitle = computed(() => props.directCount < 0
	? '分享你的专属邀请码'
	: `已邀请直属好友 ${props.directCount} 人`)
</script>

<template>
	<view class="mine-invite" @tap="emit('open')">
		<view class="mine-invite-icon"><text class="cuIcon-friendadd"></text></view>
		<view class="mine-invite-copy">
			<text class="mine-invite-title">邀请好友</text>
			<text class="mine-invite-subtitle">{{ subtitle }}</text>
		</view>
		<view class="mine-invite-action">去邀请<text class="cuIcon-right"></text></view>
	</view>
</template>

<style scoped>
.mine-invite { position:relative; display:flex; align-items:center; gap:19rpx; overflow:hidden; min-height:134rpx; padding:22rpx 25rpx; border:1rpx solid #dbe5ff; border-radius:30rpx; background:linear-gradient(105deg,#e9efff,#f4f0ff 92%); box-shadow:0 9rpx 28rpx rgba(77,96,187,.07); }
.mine-invite::after { position:absolute; right:-50rpx; top:-65rpx; width:230rpx; height:230rpx; border:28rpx solid rgba(255,255,255,.43); border-radius:50%; content:''; }
.mine-invite-icon { z-index:1; display:flex; align-items:center; justify-content:center; flex:none; width:80rpx; height:80rpx; border-radius:24rpx; background:#fff; color:#6570ec; box-shadow:0 7rpx 20rpx rgba(87,102,203,.1); font-size:35rpx; }
.mine-invite-copy { z-index:1; display:flex; flex:1; min-width:0; flex-direction:column; gap:7rpx; }
.mine-invite-title { color:#293a76; font-size:28rpx; font-weight:750; }
.mine-invite-subtitle { overflow:hidden; white-space:nowrap; text-overflow:ellipsis; color:#7280aa; font-size:21rpx; }
.mine-invite-action { z-index:1; display:flex; align-items:center; gap:4rpx; flex:none; color:#5068dc; font-size:22rpx; font-weight:700; }
.mine-invite-action .cuIcon-right { font-size:15rpx; }
</style>
