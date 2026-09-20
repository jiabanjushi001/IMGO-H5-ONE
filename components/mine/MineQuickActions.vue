<script setup>
import { computed } from 'vue'

const props = defineProps({
	showAbout: { type: Boolean, default: false },
	showScan: { type: Boolean, default: true },
	versionName: { type: String, default: '' }
})
const emit = defineEmits(['scan', 'bank', 'secure', 'settings', 'about', 'check-version'])
// #ifdef APP-PLUS
const isAppPlus = true
// #endif
// #ifndef APP-PLUS
const isAppPlus = false
// #endif
const showMoreServices = computed(() => isAppPlus || props.showAbout)
const shortcuts = computed(() => [
	{ key: 'scan', label: '扫一扫', icon: 'cuIcon-scan', tone: 'blue' },
	{ key: 'bank', label: '银行卡', icon: 'cuIcon-card', tone: 'violet' },
	{ key: 'secure', label: '账号安全', icon: 'cuIcon-safe', tone: 'cyan' },
	{ key: 'settings', label: '通用设置', icon: 'cuIcon-settings', tone: 'amber' }
].filter(item => item.key !== 'scan' || props.showScan))
</script>

<template>
	<view class="mine-actions">
		<view class="mine-actions-heading"><text>常用功能</text><text>快捷访问</text></view>
		<view class="mine-actions-grid" :class="{ 'mine-actions-grid-compact': !showScan }">
			<view v-for="item in shortcuts" :key="item.key" class="mine-action-tile" @tap="emit(item.key)">
				<view class="mine-action-icon" :class="`tone-${item.tone}`"><text :class="item.icon"></text></view>
				<text class="mine-action-label">{{ item.label }}</text>
				<text class="cuIcon-right mine-action-arrow"></text>
			</view>
		</view>
		<view v-if="showMoreServices" class="mine-actions-heading mine-actions-heading-secondary"><text>更多服务</text></view>
		<view v-if="showMoreServices" class="mine-secondary-list">
			<!-- #ifdef APP-PLUS -->
			<view class="mine-secondary-row" @tap="emit('check-version')"><text class="cuIcon-hot mine-secondary-icon"></text><text>检查更新</text><text class="mine-secondary-meta">{{ versionName }}</text><text class="cuIcon-right mine-secondary-arrow"></text></view>
			<!-- #endif -->
			<view v-if="showAbout" class="mine-secondary-row" @tap="emit('about')"><text class="cuIcon-info mine-secondary-icon"></text><text>关于 IM</text><text class="cuIcon-right mine-secondary-arrow"></text></view>
		</view>
	</view>
</template>

<style scoped>
.mine-actions-heading { display:flex; align-items:center; justify-content:space-between; margin:4rpx 4rpx 17rpx; color:#27314a; font-size:28rpx; font-weight:750; }
.mine-actions-heading text:last-child:not(:first-child) { color:#a0a9bc; font-size:20rpx; font-weight:400; }
.mine-actions-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:16rpx; }
.mine-actions-grid-compact { grid-template-columns:repeat(3,minmax(0,1fr)); }
.mine-actions-grid-compact .mine-action-tile { flex-direction:column; justify-content:center; gap:9rpx; min-height:148rpx; padding:16rpx 8rpx; }
.mine-actions-grid-compact .mine-action-label { flex:none; max-width:100%; text-align:center; }
.mine-actions-grid-compact .mine-action-arrow { display:none; }
.mine-action-tile { display:flex; align-items:center; gap:14rpx; min-width:0; min-height:113rpx; padding:18rpx 20rpx; border:1rpx solid #e9edf6; border-radius:26rpx; background:#fff; box-shadow:0 8rpx 26rpx rgba(41,55,101,.045); }
.mine-action-icon { flex:none; display:flex; align-items:center; justify-content:center; width:62rpx; height:62rpx; border-radius:18rpx; font-size:30rpx; }
.tone-blue { background:#eaf0ff; color:#4e72ef; }
.tone-violet { background:#f1edff; color:#8265e7; }
.tone-cyan { background:#e8f7fb; color:#3296b1; }
.tone-amber { background:#fff3e4; color:#ca8a34; }
.mine-action-label { flex:1; overflow:hidden; white-space:nowrap; text-overflow:ellipsis; color:#344058; font-size:24rpx; font-weight:650; }
.mine-action-arrow { color:#b8c1d0; font-size:15rpx; }
.mine-actions-heading-secondary { margin-top:34rpx; }
.mine-secondary-list { overflow:hidden; border:1rpx solid #e9edf6; border-radius:28rpx; background:#fff; box-shadow:0 8rpx 26rpx rgba(41,55,101,.045); }
.mine-secondary-row { display:flex; align-items:center; gap:17rpx; min-height:98rpx; padding:0 27rpx; color:#344058; font-size:24rpx; }
.mine-secondary-row + .mine-secondary-row { border-top:1rpx solid #eff2f8; }
.mine-secondary-icon { color:#6778b3; font-size:31rpx; }
.mine-secondary-arrow { margin-left:auto; color:#b8c1d0; font-size:18rpx; }
.mine-secondary-meta { margin-left:auto; color:#9aa4b4; font-size:20rpx; }
.mine-secondary-meta + .mine-secondary-arrow { margin-left:0; }
</style>
