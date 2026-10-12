<script setup>
defineProps({
	methods: { type: Array, default: () => [] },
	selectedType: { type: String, default: 'bank' }
})
const emit = defineEmits(['select', 'manage'])
</script>

<template>
	<view class="method-picker">
		<view class="method-heading"><text>收款方式</text><text @tap.stop="emit('manage')">管理收款方式 <text class="cuIcon-right"></text></text></view>
		<view class="method-list">
			<view v-for="item in methods" :key="item.type" class="method-row" :class="{ selected: selectedType === item.type, unavailable: !item.ready }" @tap="item.ready ? emit('select', item.type) : emit('manage')">
				<view class="method-icon" :class="`is-${item.type}`"><text :class="item.icon"></text></view>
				<view class="method-copy"><text>{{ item.label }}</text><text>{{ item.description }}</text></view>
				<text v-if="item.ready" class="method-check" :class="selectedType === item.type ? 'cuIcon-roundcheckfill' : 'cuIcon-round'"></text>
				<text v-else class="method-bind">去处理</text>
			</view>
		</view>
	</view>
</template>

<style scoped>
.method-picker { margin-top:32rpx; }
.method-heading { display:flex; justify-content:space-between; align-items:center; margin-bottom:13rpx; color:#2a344b; font-size:27rpx; font-weight:700; }
.method-heading > text:last-child { color:#6475df; font-size:21rpx; font-weight:500; }
.method-list { overflow:hidden; border:1rpx solid #e8ecf4; border-radius:22rpx; }
.method-row { display:flex; align-items:center; gap:15rpx; min-height:102rpx; padding:12rpx 20rpx; background:#fff; box-sizing:border-box; }
.method-row + .method-row { border-top:1rpx solid #edf0f6; }
.method-row.selected { background:#f7f8ff; }
.method-row.unavailable { background:#fafbfc; }
.method-icon { display:flex; flex:none; align-items:center; justify-content:center; width:60rpx; height:60rpx; border-radius:17rpx; background:#eef1ff; color:#586def; font-size:30rpx; }
.method-icon.is-alipay { background:#eaf5ff; color:#2696e8; }
.method-icon.is-wechat { background:#eaf9f1; color:#20a864; }
.method-copy { flex:1; min-width:0; display:flex; flex-direction:column; gap:5rpx; }
.method-copy text:first-child { color:#354159; font-size:25rpx; font-weight:650; }
.method-copy text:last-child { overflow:hidden; color:#929cad; font-size:20rpx; text-overflow:ellipsis; white-space:nowrap; }
.method-check { flex:none; color:#566fff; font-size:31rpx; }
.method-bind { flex:none; color:#6575dc; font-size:21rpx; }
</style>
