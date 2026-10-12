<script setup>
import { ref, watch } from 'vue'
import PreparedVideoLibrary from './PreparedVideoLibrary.vue'

const props = defineProps({
	visible: { type: Boolean, default: false },
	allowPreparedVideo: { type: Boolean, default: false },
	showVoice: { type: Boolean, default: true }
})
const emit = defineEmits(['close', 'start'])
const showLibrary = ref(false)

watch(() => props.visible, visible => {
	if (!visible) showLibrary.value = false
})

function start(type, videoMode = '', video = null) {
	emit('start', { type, videoMode, video })
}
</script>

<template>
	<view v-if="visible" class="mode-mask" @tap="emit('close')">
		<view class="mode-sheet" @tap.stop>
			<view class="mode-handle"></view>
			<view class="mode-title">选择通话方式</view>
			<view class="mode-options">
				<view v-if="showVoice" class="mode-option" @tap="start(0)">
					<view class="mode-icon voice"><text class="cuIcon-dianhua"></text></view>
					<view class="mode-copy"><text class="mode-name">语音通话</text><text class="mode-desc">使用麦克风通话</text></view>
					<text class="cuIcon-right mode-arrow"></text>
				</view>
				<view class="mode-option" @tap="start(1, 'live')">
					<view class="mode-icon live"><text class="cuIcon-record"></text></view>
					<view class="mode-copy"><text class="mode-name">普通视频</text><text class="mode-desc">使用实时摄像头</text></view>
					<text class="cuIcon-right mode-arrow"></text>
				</view>
				<view v-if="allowPreparedVideo" class="mode-option" :class="{ active: showLibrary }" @tap="showLibrary = !showLibrary">
					<view class="mode-icon recorded"><text class="cuIcon-playfill"></text></view>
					<view class="mode-copy"><text class="mode-name">预录视频</text><text class="mode-desc">从已上传视频中选择</text></view>
					<text :class="showLibrary ? 'cuIcon-unfold' : 'cuIcon-right'" class="mode-arrow"></text>
				</view>
			</view>
			<PreparedVideoLibrary v-if="allowPreparedVideo" :visible="visible && showLibrary" v-show="showLibrary" @select="video => start(1, 'recorded', video)" />
			<view class="cancel-button" @tap="emit('close')">取消</view>
		</view>
	</view>
</template>

<style scoped>
.mode-mask { position:fixed; inset:0; z-index:10020; display:flex; align-items:flex-end; justify-content:center; background:rgba(17,24,39,.48); }
.mode-sheet { width:100%; max-width:750rpx; overflow:hidden; border-radius:30rpx 30rpx 0 0; background:#fff; box-shadow:0 -20rpx 60rpx rgba(20,28,62,.16); }
.mode-handle { width:72rpx; height:7rpx; margin:16rpx auto 8rpx; border-radius:999rpx; background:#d8ddeb; }
.mode-title { padding:12rpx 30rpx 20rpx; color:#202b45; font-size:30rpx; font-weight:800; text-align:center; }
.mode-options { margin:0 24rpx 20rpx; overflow:hidden; border:1rpx solid #eceef6; border-radius:24rpx; }
.mode-option { display:flex; align-items:center; gap:20rpx; min-height:112rpx; padding:15rpx 22rpx; border-bottom:1rpx solid #eceef6; background:#fff; }
.mode-option:last-child { border-bottom:0; }
.mode-option.active { background:#f7f8ff; }
.mode-icon { display:flex; align-items:center; justify-content:center; width:72rpx; height:72rpx; flex:0 0 auto; border-radius:22rpx; color:#fff; font-size:33rpx; }
.mode-icon.voice { background:linear-gradient(145deg,#3fc89a,#27ad80); }
.mode-icon.live { background:linear-gradient(145deg,#5f78ff,#4d5ee5); }
.mode-icon.recorded { background:linear-gradient(145deg,#aa73f8,#7658e7); }
.mode-copy { display:flex; min-width:0; flex:1; flex-direction:column; }
.mode-name { color:#242e47; font-size:27rpx; font-weight:700; }
.mode-desc { margin-top:6rpx; color:#98a0b2; font-size:21rpx; }
.mode-arrow { color:#b6bdcd; font-size:27rpx; }
.cancel-button { height:94rpx; border-top:10rpx solid #f3f4f8; color:#5d6780; font-size:28rpx; line-height:84rpx; text-align:center; }
</style>
