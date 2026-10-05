<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import IdentityDocumentUpload from '@/components/mine/IdentityDocumentUpload.vue'
import identityApi from '@/api/identity.js'
import { useloginStore } from '@/store/login'
import pinia from '@/store/index'

const loginStore = useloginStore(pinia)
const loading = ref(true)
const saving = ref(false)
const submitted = ref(false)
const status = ref(-1)
const realName = ref('')
const idNumber = ref('')
const frontFileId = ref(0)
const backFileId = ref(0)
const frontImage = ref('')
const backImage = ref('')
const remark = ref('')

const canEdit = computed(() => !submitted.value || status.value === 2)
const statusText = computed(() => ({ '-1': '未提交', 0: '审核中', 1: '已实名', 2: '已拒绝' }[status.value] || '未提交'))
const statusClass = computed(() => ({ '-1': 'unsubmitted', 0: 'pending', 1: 'approved', 2: 'rejected' }[status.value] || 'unsubmitted'))
const statusDescription = computed(() => {
	if (status.value === 0) return '资料已提交，管理员审核前暂时不能修改。'
	if (status.value === 1) return '实名认证已通过，证件资料不可再次修改。'
	if (status.value === 2) return '审核未通过，请根据原因修改资料后重新提交。'
	return '请使用本人有效身份证，确保照片完整、清晰、无遮挡。'
})

function syncLoginAuth(value) {
	const next = value ? 1 : 0
	if (Number(loginStore.userInfo.is_auth || 0) === next) return
	loginStore.login({ ...loginStore.userInfo, is_auth: next })
}

async function load() {
	loading.value = true
	try {
		const res = await identityApi.get()
		if (Number(res.code) !== 0) throw new Error(res.msg || '读取实名认证失败')
		const data = res.data || {}
		submitted.value = !!data.submitted
		status.value = Number(data.status ?? -1)
		realName.value = data.real_name || ''
		idNumber.value = data.id_number || ''
		frontFileId.value = Number(data.front_file_id || 0)
		backFileId.value = Number(data.back_file_id || 0)
		frontImage.value = data.front_image || ''
		backImage.value = data.back_image || ''
		remark.value = data.remark || ''
		syncLoginAuth(!!data.is_auth)
	} catch (error) {
		uni.showToast({ title: error.message || '读取失败，请重试', icon: 'none' })
	} finally {
		loading.value = false
	}
}

function handleUploaded(file) {
	if (file.side === 'front') {
		frontFileId.value = file.fileId
		frontImage.value = file.url
	} else {
		backFileId.value = file.fileId
		backImage.value = file.url
	}
}

async function submit() {
	if (saving.value || !canEdit.value) return
	const name = realName.value.trim()
	const number = idNumber.value.trim().toUpperCase()
	if (name.length < 2) {
		uni.showToast({ title: '请输入身份证上的真实姓名', icon: 'none' })
		return
	}
	if (!/^\d{17}[\dX]$/.test(number)) {
		uni.showToast({ title: '请输入有效的18位身份证号', icon: 'none' })
		return
	}
	if (!frontFileId.value || !backFileId.value) {
		uni.showToast({ title: '请上传身份证正反面', icon: 'none' })
		return
	}
	if (frontFileId.value === backFileId.value) {
		uni.showToast({ title: '正反面不能使用同一张照片', icon: 'none' })
		return
	}
	saving.value = true
	try {
		const res = await identityApi.save({ real_name: name, id_number: number, front_file_id: frontFileId.value, back_file_id: backFileId.value })
		if (Number(res.code) !== 0) throw new Error(res.msg || '提交失败')
		uni.showToast({ title: '已提交，等待审核', icon: 'none' })
		await load()
	} catch (error) {
		uni.showToast({ title: error.message || '提交失败，请重试', icon: 'none' })
	} finally {
		saving.value = false
	}
}

onShow(load)
</script>

<template>
	<view class="identity-page">
		<cu-custom bgColor="text-white" bgStyle="background:linear-gradient(126deg,#4c63e9 0%,#5e70f5 55%,#6b6eeb 100%);color:#fff;" :isBack="true" :fallbackToHome="true">
			<template #backText></template>
			<template #content>实名认证</template>
		</cu-custom>
		<view class="identity-body">
			<view class="identity-hero">
				<view class="identity-hero-icon"><text class="cuIcon-card"></text></view>
				<view class="identity-hero-copy">
					<view class="identity-title">身份信息认证</view>
					<view class="identity-subtitle">用于确认账号实名信息，证件资料仅供审核</view>
				</view>
				<text class="identity-status" :class="statusClass">{{ statusText }}</text>
			</view>

			<view v-if="loading" class="identity-loading">正在读取实名认证信息…</view>
			<template v-else>
				<view class="identity-message" :class="statusClass">
					<text class="identity-message-dot"></text>
					<view>
						<view class="identity-message-title">{{ statusText }}</view>
						<view class="identity-message-text">{{ statusDescription }}</view>
						<view v-if="remark" class="identity-reason">审核备注：{{ remark }}</view>
					</view>
				</view>

				<view class="identity-card">
					<view class="identity-section-title">身份资料</view>
					<view class="identity-field">
						<text class="identity-label">真实姓名</text>
						<input v-if="canEdit" v-model="realName" maxlength="50" placeholder="请输入身份证上的姓名" />
						<text v-else class="identity-value">{{ realName || '—' }}</text>
					</view>
					<view class="identity-field">
						<text class="identity-label">身份证号</text>
						<input v-if="canEdit" v-model="idNumber" maxlength="18" placeholder="请输入18位身份证号" />
						<text v-else class="identity-value identity-number">{{ idNumber || '—' }}</text>
					</view>
				</view>

				<view class="identity-card">
					<view class="identity-section-title">证件照片</view>
					<view class="identity-upload-list">
						<view>
							<view class="identity-upload-heading">身份证人像面</view>
							<IdentityDocumentUpload label="上传人像面" hint="保持文字和头像清晰完整" side="front" :src="frontImage" :readonly="!canEdit" @uploaded="handleUploaded" />
						</view>
						<view>
							<view class="identity-upload-heading">身份证国徽面</view>
							<IdentityDocumentUpload label="上传国徽面" hint="保持签发机关和有效期清晰" side="back" :src="backImage" :readonly="!canEdit" @uploaded="handleUploaded" />
						</view>
					</view>
				</view>

				<button v-if="canEdit" class="identity-submit" :disabled="saving" @tap="submit">{{ saving ? '提交中…' : (submitted ? '重新提交审核' : '提交实名认证') }}</button>
				<button v-else-if="status === 0" class="identity-refresh" @tap="load">刷新审核状态</button>
				<view class="identity-privacy"><text class="cuIcon-lock"></text> 身份证号加密保存，本人页面完整显示；请勿在聊天中发送证件资料。</view>
			</template>
		</view>
	</view>
</template>

<style scoped>
.identity-page { min-height:100vh; color:#263249; background:radial-gradient(circle at 95% 2%,rgba(113,96,235,.12),transparent 28%),#f4f6fb; }
.identity-body { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; padding:28rpx 26rpx 70rpx; }
.identity-hero { display:flex; align-items:center; gap:20rpx; padding:16rpx 6rpx 30rpx; }
.identity-hero-icon { display:flex; flex:none; align-items:center; justify-content:center; width:88rpx; height:88rpx; border-radius:27rpx; color:#fff; background:linear-gradient(135deg,#5068ee,#7767e8); box-shadow:0 12rpx 28rpx rgba(76,99,233,.23); font-size:43rpx; }
.identity-hero-copy { flex:1; min-width:0; }
.identity-title { font-size:34rpx; font-weight:760; }
.identity-subtitle { margin-top:7rpx; color:#7c8799; font-size:22rpx; }
.identity-status { flex:none; padding:8rpx 17rpx; border-radius:999rpx; font-size:21rpx; font-weight:650; }
.identity-status.unsubmitted { color:#697589; background:#e9edf4; }
.identity-status.pending { color:#a56a00; background:#fff0ce; }
.identity-status.approved { color:#118a59; background:#dbf7eb; }
.identity-status.rejected { color:#c44848; background:#ffe5e5; }
.identity-loading { padding:80rpx 20rpx; color:#8390a2; text-align:center; }
.identity-message { display:flex; gap:18rpx; margin-bottom:24rpx; padding:24rpx 26rpx; border:1rpx solid #e4e8f1; border-radius:22rpx; background:#fff; }
.identity-message.pending { border-color:#f1dfb5; background:#fffaf0; }
.identity-message.approved { border-color:#cbeedd; background:#f3fcf8; }
.identity-message.rejected { border-color:#f1cccc; background:#fff7f7; }
.identity-message-dot { flex:none; width:14rpx; height:14rpx; margin-top:9rpx; border-radius:50%; background:#8793a5; }
.identity-message.pending .identity-message-dot { background:#efa925; }
.identity-message.approved .identity-message-dot { background:#25b77a; }
.identity-message.rejected .identity-message-dot { background:#e45d5d; }
.identity-message-title { font-size:27rpx; font-weight:700; }
.identity-message-text,.identity-reason { margin-top:7rpx; color:#748096; font-size:23rpx; line-height:1.65; }
.identity-reason { color:#b14b4b; }
.identity-card { margin-bottom:24rpx; padding:30rpx; border-radius:26rpx; background:#fff; box-shadow:0 10rpx 34rpx rgba(35,49,82,.055); }
.identity-section-title { margin-bottom:8rpx; font-size:29rpx; font-weight:760; }
.identity-field { display:flex; align-items:center; min-height:96rpx; border-bottom:1rpx solid #edf0f5; }
.identity-field:last-child { border-bottom:0; }
.identity-label { flex:none; width:150rpx; color:#526076; font-size:25rpx; }
.identity-field input,.identity-value { flex:1; min-width:0; color:#202b40; font-size:28rpx; text-align:right; }
.identity-number { letter-spacing:1rpx; }
.identity-upload-list { display:grid; grid-template-columns:1fr; gap:26rpx; margin-top:24rpx; }
.identity-upload-heading { margin:0 0 13rpx 4rpx; color:#526076; font-size:24rpx; font-weight:650; }
.identity-submit,.identity-refresh { margin-top:34rpx; border:0; border-radius:18rpx; color:#fff; background:linear-gradient(126deg,#4c63e9,#675fe7); box-shadow:0 14rpx 32rpx rgba(76,99,233,.24); font-size:29rpx; }
.identity-refresh { color:#4c63e9; border:1rpx solid #d8defe; background:#eef1ff; box-shadow:none; }
.identity-submit[disabled] { opacity:.58; }
.identity-privacy { display:flex; align-items:flex-start; gap:10rpx; padding:26rpx 10rpx 0; color:#8a95a7; font-size:21rpx; line-height:1.65; }
@media (min-width: 760px) { .identity-upload-list { grid-template-columns:1fr 1fr; } }
</style>
