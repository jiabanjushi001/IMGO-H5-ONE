<template>
	<view class="mine-page">
		<view class="mine-content">
			<view class="mine-profile-block">
				<MineProfileHero
					:user="loginStore.userInfo"
					:circle-avatar="!!appSetting.circleAvatar"
					@edit="editInfo"
				/>
			</view>
			<WalletSummaryCard
				:available-cents="walletAvailableCents"
				:pending-cents="walletPendingCents"
				:loading="walletLoading"
				:error="walletError"
				@refresh="loadWallet"
				@withdraw="openWithdraw"
				@history="openWithdrawHistory"
				@entries="openWalletEntries"
			/>
			<view class="mine-checkin-block">
				<MineCheckInCard
					:loading="checkInLoading"
					:error="checkInError"
					:signed="checkInSigned"
					:total-days="checkInTotal"
					@open="openCheckIn"
				/>
			</view>
			<view class="mine-invite-block">
				<MineInviteCard :direct-count="inviteDirectCount" @open="openInvite" />
			</view>
			<MineQuickActions
				:show-about="!!(globalConfig && globalConfig.demon_mode)"
				:show-scan="showScan"
				:version-name="verisonName"
				@scan="scan"
				@bank="showBankCard"
				@secure="showsecure"
				@settings="showSetting"
				@about="about"
				@check-version="checkVersion"
			/>
		</view>
	</view>
</template>

<script>
import { useloginStore } from '@/store/login'
import pinia from '@/store/index'
import scan from '@/common/scan.js'
import checkInApi from '@/api/check-in.js'
import inviteApi from '@/api/invite.js'
import walletApi from '@/api/wallet.js'
import MineProfileHero from '@/components/mine/MineProfileHero.vue'
import MineCheckInCard from '@/components/mine/MineCheckInCard.vue'
import MineQuickActions from '@/components/mine/MineQuickActions.vue'
import MineInviteCard from '@/components/mine/MineInviteCard.vue'
import WalletSummaryCard from '@/components/wallet/WalletSummaryCard.vue'
// #ifdef APP-PLUS
import appUpdate from '@/common/appUpdate.js'
// #endif

const loginStore = useloginStore(pinia)

export default {
	components: { MineProfileHero, MineCheckInCard, MineInviteCard, MineQuickActions, WalletSummaryCard },
	props: {
		active: { type: Boolean, default: false },
		refreshKey: { type: Number, default: 0 }
	},
	data() {
		return {
			loginStore,
			globalConfig: loginStore.globalConfig,
			appSetting: loginStore.appSetting,
			versionCode: '',
			verisonName: '',
			checkInLoading: false,
			checkInError: false,
			checkInSigned: false,
			checkInTotal: 0,
			inviteDirectCount: -1,
			walletLoading: true,
			walletError: false,
			walletAvailableCents: 0,
			walletPendingCents: 0
		}
	},
	computed: {
		showScan() {
			const value = this.globalConfig?.sysInfo?.showScan
			return value !== '0' && value !== 0 && value !== false
		}
	},
	watch: {
		active: {
			immediate: true,
			handler(active) {
				if (active) {
					this.loadCheckIn()
					this.loadWallet()
					this.loadInvite()
				}
			}
		},
		refreshKey() {
			if (this.active) {
				this.loadCheckIn()
				this.loadWallet()
				this.loadInvite()
			}
		}
	},
	mounted() {
		// #ifdef APP-PLUS
		plus.runtime.getProperty(plus.runtime.appid, inf => {
			this.versionCode = inf.versionCode
			this.verisonName = inf.version
		})
		// #endif
	},
	methods: {
		async loadWallet() {
			this.walletLoading = true
			this.walletError = false
			try {
				const res = await walletApi.status()
				if (res.code !== 0) throw new Error(res.msg || '获取钱包余额失败')
				this.walletAvailableCents = Number(res.data.available_cents) || 0
				this.walletPendingCents = Number(res.data.pending_cents) || 0
			} catch (error) {
				this.walletError = true
			} finally {
				this.walletLoading = false
			}
		},
		openWithdraw() { uni.navigateTo({ url: '/pages/mine/wallet-withdraw' }) },
		openWithdrawHistory() { uni.navigateTo({ url: '/pages/mine/wallet-history' }) },
		openWalletEntries() { uni.navigateTo({ url: '/pages/mine/wallet-ledger' }) },
		async loadCheckIn() {
			this.checkInLoading = true
			this.checkInError = false
			try {
				const res = await checkInApi.status()
				if (res.code !== 0) throw new Error(res.msg || '获取签到状态失败')
				this.checkInSigned = !!res.data.signed_today
				this.checkInTotal = Number(res.data.total_days) || 0
			} catch (error) {
				this.checkInError = true
			} finally {
				this.checkInLoading = false
			}
		},
		openCheckIn() { uni.navigateTo({ url: '/pages/mine/check-in' }) },
		async loadInvite() {
			try {
				const res = await inviteApi.status()
				if (res.code !== 0) throw new Error(res.msg || '获取邀请信息失败')
				this.inviteDirectCount = Number(res.data.direct_count) || 0
			} catch (error) {
				this.inviteDirectCount = -1
			}
		},
		openInvite() { uni.navigateTo({ url: '/pages/mine/invite' }) },
		logout() {
			const clientId = uni.getStorageSync('client_id')
			this.$api.LoginApi.logout({ client_id: clientId }).then(res => {
				if (res.code === 0) loginStore.logout()
			})
		},
		about() { uni.navigateTo({ url: '/pages/mine/about' }) },
		showSetting() { uni.navigateTo({ url: '/pages/mine/setting' }) },
		showsecure() { uni.navigateTo({ url: '/pages/mine/secure' }) },
		showBankCard() { uni.navigateTo({ url: '/pages/mine/bank-card' }) },
		editInfo() { uni.navigateTo({ url: '/pages/mine/profile' }) },
		scan() { scan.scanQr() },
		checkVersion() {
			// #ifdef APP-PLUS
			appUpdate(true)
			// #endif
		}
	}
}
</script>

<style scoped>
.mine-page { box-sizing:border-box; min-height:100%; padding:22rpx 22rpx 180rpx; background:radial-gradient(circle at 94% 3%,rgba(123,105,239,.10),transparent 29%),#f4f7fc; }
.mine-content { box-sizing:border-box; width:100%; max-width:900rpx; margin:0 auto; }
.mine-profile-block,.mine-checkin-block,.mine-invite-block { margin-bottom:24rpx; }
</style>
