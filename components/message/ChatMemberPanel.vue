<script setup>
import AuthImage from '@/components/AuthImage.vue'

const props = defineProps({
	members: { type: Array, default: () => [] },
	count: { type: Number, default: 0 },
	isGroup: { type: Boolean, default: false },
	canAdd: { type: Boolean, default: false },
	canManage: { type: Boolean, default: false }
})
const emit = defineEmits(['member', 'add', 'manage', 'view-all'])
</script>

<template>
	<view class="member-card">
		<view class="member-heading">
			<view class="member-heading-left">
				<view class="member-heading-icon cuIcon-people"></view>
				<text class="member-title">{{ props.isGroup ? '群成员' : '聊天成员' }}</text>
			</view>
			<text v-if="props.isGroup" class="member-count">{{ props.count }} 人</text>
		</view>
		<view class="member-grid">
			<view v-for="item in props.members" :key="item.user_id || item.userInfo?.id" class="member-item" @tap="emit('member', item.userInfo)">
				<view class="member-avatar">
					<AuthImage
						v-if="item.userInfo?.avatar"
						class="member-avatar-image"
						:src="item.userInfo.avatar"
						:info="item.userInfo"
						avatar
						mode="aspectFill"
					/>
					<text v-else class="member-initial">{{ (item.userInfo?.displayName || '?').slice(0, 1) }}</text>
				</view>
				<text class="member-name">{{ item.userInfo?.displayName || '群成员' }}</text>
			</view>
			<view v-if="props.canAdd" class="member-item member-action" @tap="emit('add')">
				<view class="member-action-icon cuIcon-add"></view>
				<text class="member-action-label">添加成员</text>
			</view>
			<view v-if="props.canManage" class="member-item member-action" @tap="emit('manage')">
				<view class="member-action-icon member-action-remove cuIcon-move"></view>
				<text class="member-action-label">管理成员</text>
			</view>
		</view>
		<view v-if="props.isGroup" class="member-footer" @tap="emit('view-all')">
			<text>查看全部群成员</text>
			<text class="cuIcon-right"></text>
		</view>
	</view>
</template>

<style scoped>
.member-card { overflow:hidden; border:1rpx solid #e9edf7; border-radius:30rpx; background:#fff; box-shadow:0 12rpx 34rpx rgba(48,61,107,.055); }
.member-heading { display:flex; align-items:center; justify-content:space-between; padding:29rpx 30rpx 8rpx; }
.member-heading-left { display:flex; align-items:center; gap:13rpx; }
.member-heading-icon { display:flex; align-items:center; justify-content:center; width:46rpx; height:46rpx; border-radius:14rpx; background:#eef1ff; color:#526dff; font-size:27rpx; }
.member-title { color:#192138; font-size:30rpx; font-weight:700; }
.member-count { padding:7rpx 17rpx; border-radius:20rpx; background:#f3f5fb; color:#7b849a; font-size:22rpx; }
.member-grid { display:grid; grid-template-columns:repeat(5,minmax(0,1fr)); gap:24rpx 8rpx; padding:23rpx 24rpx 31rpx; }
.member-item { display:flex; min-width:0; align-items:center; flex-direction:column; gap:10rpx; }
.member-avatar,.member-action-icon { display:flex; overflow:hidden; align-items:center; justify-content:center; box-sizing:border-box; width:86rpx; height:86rpx; border-radius:24rpx; }
.member-avatar { background:linear-gradient(135deg,#627cff,#7781da); color:#fff; box-shadow:0 9rpx 20rpx rgba(67,84,167,.13); }
.member-avatar :deep(.auth-image), .member-avatar-image { width:100%; height:100%; }
.member-initial { font-size:34rpx; font-weight:700; }
.member-name,.member-action-label { display:block; overflow:hidden; width:100%; white-space:nowrap; text-overflow:ellipsis; text-align:center; font-size:21rpx; line-height:1.35; color:#667088; }
.member-action-icon { border:2rpx dashed #cbd3ef; background:#f6f8ff; color:#526dff; font-size:40rpx; }
.member-action-remove { border-color:#dfd9f6; background:#f8f6ff; color:#8665c8; }
.member-action-label { color:#69728b; }
.member-footer { display:flex; align-items:center; justify-content:center; gap:10rpx; min-height:83rpx; border-top:1rpx solid #f0f2f8; color:#526dff; font-size:24rpx; font-weight:600; }
.member-footer .cuIcon-right { font-size:19rpx; }
</style>
