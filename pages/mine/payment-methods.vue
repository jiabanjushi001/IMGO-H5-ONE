<script setup>
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import bankApi from '@/api/bank.js'
import paymentMethodApi from '@/api/payment-method.js'
import AuthImage from '@/components/AuthImage.vue'
import PaymentQrUpload from '@/components/wallet/PaymentQrUpload.vue'

const loading = ref(true)
const savingType = ref('')
const bank = ref({})
const alipay = ref({ name: '', qrFileId: 0, qrImage: '', bound: false, status: -1, remark: '', legacyQrMissing: false })
const wechat = ref({ name: '', qrFileId: 0, qrImage: '', bound: false, status: -1, remark: '' })
const bankStatus = computed(() => {
	if (!bank.value.bound) return '未绑定'
	return ['审核中', '已通过', '已拒绝'][Number(bank.value.status)] || '审核中'
})
const bankDescription = computed(() => {
	if (!bank.value.bound) return '绑定银行卡后需等待管理员审核'
	return `${bank.value.bank_name || '银行卡'} ${bank.value.receipt_account_masked || ''}`.trim()
})

async function load() {
	loading.value = true
	try {
		const [bankRes, paymentRes] = await Promise.all([bankApi.get(), paymentMethodApi.get()])
		if (Number(bankRes.code) !== 0 || Number(paymentRes.code) !== 0) throw new Error(bankRes.msg || paymentRes.msg || '读取收款方式失败')
		bank.value = bankRes.data || {}
		const ali = paymentRes.data && paymentRes.data.alipay || {}
		const wx = paymentRes.data && paymentRes.data.wechat || {}
		const alipayQrFileId = Number(ali.qr_file_id || 0)
		alipay.value = { name: ali.receipt_name || '', qrFileId: alipayQrFileId, qrImage: ali.qr_image || '', bound: !!ali.bound, status: ali.bound ? Number(ali.status) : -1, remark: ali.remark || '', legacyQrMissing: !!ali.bound && !alipayQrFileId }
		wechat.value = { name: wx.receipt_name || '', qrFileId: Number(wx.qr_file_id || 0), qrImage: wx.qr_image || '', bound: !!wx.bound, status: wx.bound ? Number(wx.status) : -1, remark: wx.remark || '' }
	} catch (error) {
		uni.showToast({ title: error.message || '读取失败，请重试', icon: 'none' })
	} finally { loading.value = false }
}

function openBank() { uni.navigateTo({ url: '/pages/mine/bank-card' }) }
async function setAlipayQr(file) {
	alipay.value.qrFileId = file.fileId
	alipay.value.qrImage = file.url
	await submitAlipay(true)
}
async function setWechatQr(file) {
	wechat.value.qrFileId = file.fileId
	wechat.value.qrImage = file.url
	await submitWechat(true)
}
function validName(name) { return String(name || '').trim().length >= 2 }
function canEdit(method) { return !method.bound || Number(method.status) === 2 || !!method.legacyQrMissing }
function methodStatus(method) {
	if (!method.bound) return '未提交'
	return ['审核中', '已通过', '已拒绝'][Number(method.status)] || '审核中'
}
function methodDescription(method, emptyText) {
	if (!method.bound) return emptyText
	return Number(method.status) === 1 ? '审核已通过，可用于提现' : Number(method.status) === 2 ? '审核未通过，可修改后重新提交' : '资料已提交，请等待管理员审核'
}

async function submitAlipay(fromUpload = false) {
	if (savingType.value) return
	if (!canEdit(alipay.value)) { uni.showToast({ title: '当前资料不可修改', icon: 'none' }); return }
	const name = alipay.value.name.trim()
	if (!validName(name)) { uni.showToast({ title: fromUpload ? '图片已上传，请填写收款姓名后提交审核' : '请输入收款姓名', icon: 'none' }); return }
	if (!alipay.value.qrFileId) { uni.showToast({ title: '请上传支付宝收款码', icon: 'none' }); return }
	savingType.value = 'alipay'
	try {
		const res = await paymentMethodApi.save({ payment_type: 'alipay', receipt_name: name, qr_file_id: alipay.value.qrFileId })
		if (Number(res.code) !== 0) throw new Error(res.msg || '保存失败')
		uni.showToast({ title: '已提交，等待审核', icon: 'none' })
		await load()
	} catch (error) { uni.showToast({ title: error.message || '保存失败', icon: 'none' }) }
	finally { savingType.value = '' }
}
function saveAlipay() { return submitAlipay(false) }

async function submitWechat(fromUpload = false) {
	if (savingType.value) return
	if (!canEdit(wechat.value)) { uni.showToast({ title: '当前资料不可修改', icon: 'none' }); return }
	const name = wechat.value.name.trim()
	if (!validName(name)) { uni.showToast({ title: fromUpload ? '图片已上传，请填写收款姓名后提交审核' : '请输入收款姓名', icon: 'none' }); return }
	if (!wechat.value.qrFileId) { uni.showToast({ title: '请上传微信收款码', icon: 'none' }); return }
	savingType.value = 'wechat'
	try {
		const res = await paymentMethodApi.save({ payment_type: 'wechat', receipt_name: name, qr_file_id: wechat.value.qrFileId })
		if (Number(res.code) !== 0) throw new Error(res.msg || '保存失败')
		uni.showToast({ title: '已提交，等待审核', icon: 'none' })
		await load()
	} catch (error) { uni.showToast({ title: error.message || '保存失败', icon: 'none' }) }
	finally { savingType.value = '' }
}
function saveWechat() { return submitWechat(false) }

onShow(load)
</script>

<template>
	<view class="payment-page">
		<cu-custom bgColor="text-white" bgStyle="background:linear-gradient(126deg,#4c63e9 0%,#5e70f5 55%,#6b6eeb 100%);color:#fff;" :isBack="true" :fallbackToHome="true"><template #content>收款方式</template></cu-custom>
		<view class="payment-body">
			<view class="payment-hero"><view class="payment-hero-icon"><text class="cuIcon-pay"></text></view><view><view>绑定收款方式</view><text>提现前请确认收款资料真实、准确</text></view></view>
			<view v-if="loading" class="payment-loading">正在读取收款方式…</view>
			<template v-else>
				<view class="payment-card bank-entry" @tap="openBank">
					<view class="payment-card-head"><view class="payment-brand is-bank"><text class="cuIcon-card"></text></view><view class="payment-head-copy"><text>银行卡</text><text>{{ bankDescription }}</text></view><text class="payment-state">{{ bankStatus }}</text><text class="cuIcon-right payment-arrow"></text></view>
				</view>
				<view class="payment-card">
					<view class="payment-card-head"><view class="payment-brand is-alipay"><text class="cuIcon-pay"></text></view><view class="payment-head-copy"><text>支付宝</text><text>{{ methodDescription(alipay, '提交后由管理员审核') }}</text></view><text class="payment-state" :class="`is-${alipay.status}`">{{ methodStatus(alipay) }}</text></view>
					<view class="payment-field"><text>收款姓名</text><input v-model.trim="alipay.name" :disabled="!canEdit(alipay)" maxlength="100" placeholder="请输入支付宝实名姓名" /></view>
					<view class="payment-qr"><PaymentQrUpload v-if="canEdit(alipay)" :src="alipay.qrImage" label="支付宝收款码" @uploaded="setAlipayQr" /><AuthImage v-else-if="alipay.qrImage" class="payment-qr-locked" :src="alipay.qrImage" mode="aspectFit" /></view>
					<view v-if="alipay.remark" class="payment-remark">处理备注：{{ alipay.remark }}</view>
					<button v-if="canEdit(alipay)" class="payment-save" :disabled="!!savingType" @tap="saveAlipay">{{ savingType === 'alipay' ? '提交中…' : (alipay.bound ? '重新提交支付宝收款码' : '提交支付宝审核') }}</button>
				</view>
				<view class="payment-card">
					<view class="payment-card-head"><view class="payment-brand is-wechat"><text class="cuIcon-weixin"></text></view><view class="payment-head-copy"><text>微信收款码</text><text>{{ methodDescription(wechat, '上传后由管理员审核') }}</text></view><text class="payment-state" :class="`is-${wechat.status}`">{{ methodStatus(wechat) }}</text></view>
					<view class="payment-field"><text>收款姓名</text><input v-model.trim="wechat.name" :disabled="!canEdit(wechat)" maxlength="100" placeholder="请输入收款人姓名" /></view>
					<view class="payment-qr"><PaymentQrUpload v-if="canEdit(wechat)" :src="wechat.qrImage" label="微信收款码" @uploaded="setWechatQr" /><AuthImage v-else-if="wechat.qrImage" class="payment-qr-locked" :src="wechat.qrImage" mode="aspectFit" /></view>
					<view v-if="wechat.remark" class="payment-remark">处理备注：{{ wechat.remark }}</view>
					<button v-if="canEdit(wechat)" class="payment-save is-wechat" :disabled="!!savingType" @tap="saveWechat">{{ savingType === 'wechat' ? '提交中…' : (wechat.bound ? '重新提交微信收款码' : '提交微信审核') }}</button>
				</view>
				<view class="payment-tip"><text class="cuIcon-lock"></text><text>收款资料仅用于提现审核和线下打款，请勿在聊天中发送。</text></view>
			</template>
		</view>
	</view>
</template>

<style scoped>
.payment-page { min-height:100vh; color:#27334b; background:radial-gradient(circle at 100% 0%,rgba(112,121,250,.13),transparent 40%),#f4f7fc; }
.payment-body { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; padding:28rpx 26rpx 80rpx; }
.payment-hero { display:flex; align-items:center; gap:19rpx; margin:8rpx 6rpx 28rpx; font-size:33rpx; font-weight:750; }
.payment-hero text { display:block; margin-top:7rpx; color:#8490a4; font-size:22rpx; font-weight:400; }
.payment-hero-icon { display:flex; align-items:center; justify-content:center; width:78rpx; height:78rpx; border-radius:23rpx; color:#fff; background:linear-gradient(135deg,#526dff,#7168e9); font-size:38rpx; }
.payment-loading { padding:80rpx 20rpx; color:#8490a4; text-align:center; }
.payment-card { margin-bottom:24rpx; padding:30rpx; border:1rpx solid #e7ebf5; border-radius:28rpx; background:#fff; box-shadow:0 12rpx 38rpx rgba(39,54,91,.055); }
.bank-entry { padding:24rpx 28rpx; }
.payment-card-head { display:flex; align-items:center; gap:16rpx; }
.payment-brand { display:flex; flex:none; align-items:center; justify-content:center; width:64rpx; height:64rpx; border-radius:19rpx; font-size:31rpx; }
.payment-brand.is-bank { color:#596de8; background:#eef0ff; }.payment-brand.is-alipay { color:#2498e7; background:#eaf6ff; }.payment-brand.is-wechat { color:#20a765; background:#e9f9f1; }
.payment-head-copy { flex:1; min-width:0; display:flex; flex-direction:column; gap:5rpx; }
.payment-head-copy text:first-child { color:#2d3952; font-size:27rpx; font-weight:720; }
.payment-head-copy text:last-child { overflow:hidden; color:#8a95a8; font-size:21rpx; text-overflow:ellipsis; white-space:nowrap; }
.payment-state,.payment-bound { flex:none; padding:7rpx 14rpx; border-radius:999rpx; background:#eef1ff; color:#5f70db; font-size:20rpx; }.payment-bound { color:#23875b; background:#e8f7ef; }
.payment-state.is-1 { color:#23875b; background:#e8f7ef; }.payment-state.is-2 { color:#c05a4e; background:#fff0ed; }
.payment-arrow { color:#a6afc0; }
.payment-field { padding:27rpx 0 20rpx; border-bottom:1rpx solid #edf0f5; }
.payment-field > text { display:block; margin-bottom:12rpx; color:#566277; font-size:23rpx; }
.payment-field input { width:100%; height:54rpx; color:#263249; font-size:28rpx; }
.payment-field input[disabled] { color:#747e91; }
.payment-qr { margin-top:25rpx; }
.payment-qr-locked { display:block; width:320rpx; height:320rpx; border-radius:18rpx; background:#f5f7fa; }
.payment-remark { margin-top:22rpx; padding:18rpx 20rpx; border-radius:14rpx; color:#a05f39; background:#fff7ed; font-size:22rpx; line-height:1.5; }
.payment-save { margin-top:30rpx; border:0; border-radius:17rpx; color:#fff; background:linear-gradient(126deg,#4c63e9,#6571ed); font-size:27rpx; }.payment-save.is-wechat { background:linear-gradient(126deg,#20a464,#37b878); }.payment-save::after { border:0; }.payment-save[disabled] { opacity:.55; }
.payment-tip { display:flex; align-items:flex-start; gap:10rpx; padding:4rpx 10rpx; color:#8a95a7; font-size:21rpx; line-height:1.65; }
</style>
