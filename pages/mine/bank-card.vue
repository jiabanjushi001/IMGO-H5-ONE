<template>
	<view class="bank-page">
		<cu-custom bgColor="text-white" bgStyle="background:linear-gradient(126deg,#4c63e9 0%,#5e70f5 55%,#6b6eeb 100%);color:#fff;" :isBack="true" :fallbackToHome="true">
			<template #backText></template>
			<template #content>绑定银行卡</template>
		</cu-custom>
		<view class="bank-body">
			<view class="bank-heading">
				<text class="bank-icon cuIcon-card"></text>
				<view>
					<view class="bank-title">我的收款银行卡</view>
					<view class="bank-subtitle">填写收款姓名、银行卡号及开户行信息，保存后等待后台处理</view>
				</view>
			</view>
			<view v-if="loading" class="bank-loading">正在读取绑卡信息…</view>
			<view v-else>
				<view v-if="bound" class="bank-current">
					<view class="bank-current-top"><text>当前银行卡</text><text class="bank-status" :class="statusClass">{{ statusText }}</text></view>
					<view class="bank-number">{{ maskedAccount }}</view>
						<view class="bank-owner">收款姓名：{{ savedName }}</view>
						<view v-if="savedBankName" class="bank-owner">收款银行：{{ savedBankName }}</view>
						<view v-if="savedBranchName" class="bank-owner">支行名称：{{ savedBranchName }}</view>
					<view v-if="remark" class="bank-remark">处理备注：{{ remark }}</view>
				</view>
				<view v-if="canEdit" class="bank-form">
					<view class="bank-form-title">{{ bound ? '修改绑卡信息' : '填写绑卡信息' }}</view>
					<view class="bank-field">
						<text class="bank-label">收款姓名</text>
						<input v-model.trim="name" maxlength="40" placeholder="请输入银行卡持有人姓名" autocomplete="off" />
					</view>
					<view class="bank-field">
						<text class="bank-label">收款账号</text>
						<input v-model="cardNumber" type="number" maxlength="30" :placeholder="bound ? '留空则不修改卡号' : '请输入银行卡号'" autocomplete="off" />
					</view>
					<view class="bank-field">
						<text class="bank-label">收款银行</text>
						<input v-model.trim="bankName" maxlength="120" placeholder="请输入开户行" autocomplete="off" />
					</view>
					<view class="bank-field">
						<text class="bank-label">支行名称</text>
						<input v-model.trim="branchName" maxlength="120" placeholder="请输入支行名称" autocomplete="off" />
					</view>
					<view v-if="bound" class="bank-hint">为保护卡号，已绑定卡号只显示末四位。修改姓名时可将卡号留空；修改卡号请填写完整新卡号。</view>
					<button class="bank-submit" :disabled="saving" @tap="save">{{ saving ? '保存中…' : (bound ? '保存修改' : '提交绑卡') }}</button>
				</view>
				<view v-else class="bank-locked">
					<view class="bank-locked-title">{{ status === 0 ? '已提交，等待管理员处理' : '银行卡已通过审核' }}</view>
					<view class="bank-locked-detail">只有管理员拒绝后才能修改绑卡资料。</view>
					<button class="bank-refresh" @tap="load">刷新处理状态</button>
				</view>
				<view class="bank-tips">{{ canEdit && bound ? '被拒绝后重新提交，状态会变为“未处理”。' : '' }}请勿在聊天中发送银行卡号。</view>
			</view>
		</view>
	</view>
</template>

<script setup>
import { computed, shallowRef } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import bankApi from '@/api/bank.js'

const loading = shallowRef(true)
const saving = shallowRef(false)
const bound = shallowRef(false)
const name = shallowRef('')
const savedName = shallowRef('')
const bankName = shallowRef('')
const savedBankName = shallowRef('')
const branchName = shallowRef('')
const savedBranchName = shallowRef('')
const cardNumber = shallowRef('')
const maskedAccount = shallowRef('')
const status = shallowRef(0)
const remark = shallowRef('')
const statusText = computed(() => ['未处理', '同意', '拒绝'][status.value] || '未处理')
const statusClass = computed(() => ['pending', 'approved', 'rejected'][status.value] || 'pending')
const canEdit = computed(() => !bound.value || status.value === 2)

async function load() {
	loading.value = true
	try {
		const res = await bankApi.get()
		if (res.code !== 0) throw new Error(res.msg || '加载绑卡信息失败')
		const card = res.data || {}
		bound.value = !!card.bound
		name.value = card.receipt_name || ''
		savedName.value = card.receipt_name || ''
		bankName.value = card.bank_name || ''
		savedBankName.value = card.bank_name || ''
		branchName.value = card.branch_name || ''
		savedBranchName.value = card.branch_name || ''
		maskedAccount.value = card.receipt_account_masked || ''
		status.value = Number(card.status || 0)
		remark.value = card.remark || ''
		cardNumber.value = ''
	} catch (error) {
		uni.showToast({ title: error.message || '加载失败，请重试', icon: 'none' })
	} finally {
		loading.value = false
	}
}

async function save() {
	if (saving.value) return
	if (!canEdit.value) {
		uni.showToast({ title: '管理员拒绝后才能修改', icon: 'none' })
		return
	}
	const holder = name.value.trim()
	const number = cardNumber.value.replace(/\s/g, '')
	const bank = bankName.value.trim()
	const branch = branchName.value.trim()
	if (!holder || holder.length < 2) {
		uni.showToast({ title: '请输入收款姓名', icon: 'none' })
		return
	}
	if ((!bound.value || number) && !/^\d{12,30}$/.test(number)) {
		uni.showToast({ title: '请输入12至30位银行卡号', icon: 'none' })
		return
	}
	if (bank.length < 2 || bank.length > 120) {
		uni.showToast({ title: '请输入收款银行', icon: 'none' })
		return
	}
	if (branch.length < 2 || branch.length > 120) {
		uni.showToast({ title: '请输入支行名称', icon: 'none' })
		return
	}
	saving.value = true
	try {
		const res = await bankApi.save({ name: holder, card_number: number, bank_name: bank, branch_name: branch })
		if (res.code === 409) {
			await load()
			uni.showToast({ title: res.msg || '请刷新处理状态', icon: 'none' })
			return
		}
		if (res.code !== 0) throw new Error(res.msg || '保存失败')
		uni.showToast({ title: '已提交，等待处理', icon: 'none' })
		await load()
	} catch (error) {
		uni.showToast({ title: error.message || '保存失败', icon: 'none' })
	} finally {
		saving.value = false
	}
}

onShow(load)
</script>

<style scoped>
.bank-page { min-height: 100vh; background: #f4f6fb; color: #253245; }
.bank-body { width: 100%; max-width: 900rpx; margin: 0 auto; padding: 28rpx; box-sizing: border-box; }
.bank-heading { display: flex; align-items: center; gap: 22rpx; padding: 24rpx 8rpx 38rpx; }
.bank-icon {
	width: 86rpx;
	height: 86rpx;
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 25rpx;
	color: #fff;
	background: linear-gradient(135deg, #526dff, #6a74e8);
	box-shadow: 0 10rpx 24rpx rgba(67, 84, 167, .18);
	font-size: 44rpx;
}
.bank-title { font-weight: 700; font-size: 34rpx; }
.bank-subtitle { margin-top: 10rpx; color: #788397; font-size: 23rpx; }
.bank-loading { padding: 60rpx; color: #788397; text-align: center; }
.bank-current, .bank-form { background: #fff; border-radius: 24rpx; padding: 32rpx; margin-bottom: 24rpx; box-shadow: 0 10rpx 32rpx rgba(31,51,78,.05); }
.bank-current { background: linear-gradient(135deg,#315baf,#364685); color: #fff; }
.bank-current-top { display: flex; align-items: center; justify-content: space-between; font-size: 25rpx; color: #dae6ff; }
.bank-status { padding: 6rpx 18rpx; border-radius: 40rpx; font-size: 22rpx; }
.bank-status.pending { background: rgba(255,204,68,.24); color: #ffe08a; }
.bank-status.approved { background: rgba(96,229,176,.25); color: #a1ffd3; }
.bank-status.rejected { background: rgba(255,130,130,.28); color: #ffb7b7; }
.bank-number { margin: 42rpx 0 28rpx; font-size: 37rpx; letter-spacing: 2rpx; font-weight: 600; }
.bank-owner, .bank-remark { font-size: 25rpx; color: #e3eaff; line-height: 1.7; }
.bank-remark { border-top: 1px solid rgba(255,255,255,.2); margin-top: 15rpx; padding-top: 15rpx; }
.bank-form-title { font-size: 30rpx; font-weight: 700; margin-bottom: 12rpx; }
.bank-field { border-bottom: 1px solid #edf0f4; padding: 26rpx 0 22rpx; }
.bank-label { display: block; color: #566477; font-size: 25rpx; margin-bottom: 15rpx; }
.bank-field input { font-size: 29rpx; height: 55rpx; color: #253245; }
.bank-hint, .bank-tips { font-size: 23rpx; line-height: 1.7; color: #8290a1; margin-top: 24rpx; }
.bank-locked { background: #fff; border-radius: 24rpx; padding: 34rpx; box-shadow: 0 10rpx 32rpx rgba(31,51,78,.05); }
.bank-locked-title { color: #253245; font-size: 29rpx; font-weight: 600; }
.bank-locked-detail { margin: 18rpx 0 26rpx; color: #718097; font-size: 24rpx; line-height: 1.6; }
.bank-refresh {
	background: #eef1ff;
	color: #4c63e9;
	border: 1px solid #d8defe;
	font-size: 25rpx;
	border-radius: 14rpx;
}
.bank-submit {
	margin-top: 36rpx;
	border: 0;
	border-radius: 16rpx;
	color: white;
	font-size: 29rpx;
	background: linear-gradient(126deg, #4c63e9 0%, #5e70f5 100%);
	box-shadow: 0 12rpx 28rpx rgba(76, 99, 233, .28);
}
.bank-submit[disabled] { opacity: .6; }
.bank-tips { padding: 0 12rpx; }
</style>
