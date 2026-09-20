<script setup>
import { computed, shallowRef, watch } from 'vue'

const props = defineProps({
	visible: { type: Boolean, default: false },
	notice: { type: String, default: '' },
	groupName: { type: String, default: '' },
	canEdit: { type: Boolean, default: false },
	saving: { type: Boolean, default: false }
})
const emit = defineEmits(['cancel', 'save'])
const draft = shallowRef('')
const characterCount = computed(() => [...draft.value].length)
const maxLength = 4000

watch(() => props.visible, isVisible => {
	if (isVisible) draft.value = props.notice || ''
}, { immediate: true })

function save() {
	if (!props.canEdit || props.saving) return
	if (characterCount.value > maxLength) {
		uni.showToast({ title: '群公告最多 4000 字', icon: 'none' })
		return
	}
	emit('save', draft.value.trim())
}
</script>

<template>
	<view class="cu-modal bottom-modal notice-editor" :class="{ show: props.visible }" @tap.self="emit('cancel')">
		<view class="cu-dialog notice-sheet">
			<view class="sheet-handle"></view>
			<view class="sheet-heading">
				<view class="sheet-title-block">
					<text class="sheet-title">{{ props.canEdit ? '编辑群公告' : '群公告' }}</text>
					<text class="sheet-subtitle">{{ props.groupName || '当前群聊' }} · {{ props.canEdit ? '更新后群成员可见' : '仅群主或管理员可编辑' }}</text>
				</view>
				<view class="sheet-close cuIcon-close" aria-label="关闭" @tap="emit('cancel')"></view>
			</view>
			<view class="sheet-body">
				<view class="editor-label"><text class="cuIcon-notification"></text> 公告内容</view>
				<view class="editor-field" :class="{ 'is-readonly': !props.canEdit }">
					<textarea v-model="draft" class="editor-textarea" :disabled="!props.canEdit" :maxlength="-1" placeholder="写下群公告，让每位成员都能看到…" placeholder-class="editor-placeholder" />
					<view class="editor-count" :class="{ 'is-over-limit': characterCount > maxLength }">{{ characterCount }} / {{ maxLength }}</view>
				</view>
				<view class="editor-hint"><text class="cuIcon-info"></text><text>公告会同步显示在群聊顶部；清空内容后顶部滚动条自动隐藏。</text></view>
			</view>
			<view class="sheet-actions">
				<view class="sheet-button secondary" @tap="emit('cancel')">{{ props.canEdit ? '取消' : '关闭' }}</view>
				<view v-if="props.canEdit" class="sheet-button primary" :class="{ disabled: props.saving || characterCount > maxLength }" @tap="save">{{ props.saving ? '保存中…' : '保存公告' }}</view>
			</view>
		</view>
	</view>
</template>

<style scoped>
.notice-editor { background:rgba(21,29,52,.46); }
.notice-editor .notice-sheet { box-sizing:border-box; width:100%; max-width:640px; padding:14rpx 30rpx calc(28rpx + env(safe-area-inset-bottom)); border-radius:38rpx 38rpx 0 0; background:#fff; text-align:left; box-shadow:0 -24rpx 70rpx rgba(24,34,75,.17); }
.sheet-handle { width:74rpx; height:8rpx; margin:2rpx auto 29rpx; border-radius:10rpx; background:#d9deea; }
.sheet-heading { display:flex; align-items:flex-start; justify-content:space-between; gap:20rpx; margin-bottom:35rpx; }
.sheet-title-block { display:flex; min-width:0; flex-direction:column; gap:9rpx; }
.sheet-title { color:#182033; font-size:36rpx; font-weight:750; line-height:1.35; }
.sheet-subtitle { overflow:hidden; white-space:nowrap; text-overflow:ellipsis; color:#929bb0; font-size:23rpx; }
.sheet-close { display:flex; flex:none; align-items:center; justify-content:center; width:59rpx; height:59rpx; border-radius:18rpx; background:#f2f4f9; color:#8b94a8; font-size:25rpx; }
.sheet-body { padding-bottom:28rpx; }
.editor-label { display:flex; align-items:center; gap:11rpx; margin-bottom:15rpx; color:#333c52; font-size:25rpx; font-weight:650; }
.editor-label .cuIcon-notification { color:#526dff; font-size:30rpx; }
.editor-field { box-sizing:border-box; min-height:310rpx; padding:21rpx 23rpx 14rpx; border:2rpx solid #e3e9f9; border-radius:24rpx; background:#f8faff; }
.editor-field:focus-within { border-color:#7285ff; box-shadow:0 0 0 6rpx rgba(82,109,255,.08); }
.editor-field.is-readonly { background:#f6f7fa; }
.editor-textarea { box-sizing:border-box; width:100%; height:260rpx; padding:0; background:transparent; color:#253049; font-size:27rpx; line-height:1.6; text-align:left; }
.editor-count { margin-top:6rpx; text-align:right; color:#9ca5b9; font-size:21rpx; }
.editor-count.is-over-limit { color:#f05d70; }
.editor-hint { display:flex; align-items:flex-start; gap:9rpx; margin-top:20rpx; color:#8b95ac; font-size:22rpx; line-height:1.55; }
.editor-hint .cuIcon-info { flex:none; margin-top:2rpx; color:#8e9df0; font-size:24rpx; }
.sheet-actions { display:flex; gap:18rpx; padding-top:23rpx; border-top:1rpx solid #eef0f7; }
.sheet-button { display:flex; align-items:center; justify-content:center; box-sizing:border-box; min-height:85rpx; border-radius:22rpx; font-size:27rpx; font-weight:650; }
.sheet-button.secondary { flex:1; background:#f0f3fa; color:#69738b; }
.sheet-button.primary { flex:1.6; background:linear-gradient(125deg,#5069ef,#775ff2); color:#fff; box-shadow:0 12rpx 27rpx rgba(85,94,221,.22); }
.sheet-button.disabled { opacity:.56; }
</style>
