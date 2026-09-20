<template>
	<view>
		<cu-custom bgColor="bg-gradual-green" :isBack="true">
			<template #backText></template>
			<template #content>帮助中心</template>
		</cu-custom>
		<view style="height:20rpx;"></view>
		<uni-collapse accordion>
			<uni-collapse-item :title="item.title" v-for="item in list" :open="item.id==1">
				<view class="content f-14 pd-10 lh-15x">
					<rich-text :nodes="item.content"></rich-text>
				</view>
			</uni-collapse-item>
		</uni-collapse>
		
	</view>
</template>

<script>
	import { useloginStore } from '@/store/login'
	import pinia from '@/store/index'
	const loginStore = useloginStore(pinia)
	import packageData from "../../package.json"
	export default {
		data() {
			return {
				loginStore:loginStore,
				globalConfig:loginStore.globalConfig,
				packData:packageData,
				list:[
					{
						id:1,
						title:'如何开启手机通知',
						content:`<div class='vditor-reset' id='preview'><ol>
<li>手机系统设置中，当前APP【允许通知】需要开启，通知设置底部有一个【消息推送】点击进去也要开启。</li>
<li>手机系统设置中，声音提醒需要开启。手机系统设置需非静音/勿扰模式。</li>
<li>Android系统需核实设备系统声音设置-&gt;通知铃声，是否有选到“无”的铃声文件；若是系统版本大于等于17.2的iOS设备，需在iOS手机系统-&gt;设置-&gt;声音与触感-&gt;默认提醒，确认是否有选择提醒铃声。</li>
<li>Android系统手机收不到消息提醒，还需检查以下几项：<br />
①手机系统设置中，【APP-&gt;允许通知-&gt;是否设为重要】是否设置为【重要】。<br />
②手机系统设置中，【锁屏后通知】相关开关，是否设置为锁屏后通知。<br />
③手机系统设置中如果有通知过滤规则，需将当前APP的通知过滤规则设置为【重要】。<br />
④手机后台设置中，网络是否可用。<br />
⑤如果手机在后台运行、锁屏状态、未运行状态时无法收到提醒，需检查【自启动设置】。<br />
⑥请确认手机是否开启了省电策略功能，可在手机系统设置中-&gt;应用管理-&gt;当前APP-&gt;省电策略-&gt;设置为无限制。或在系统设置中-&gt;电池-&gt;当前APP-&gt;启动管理-&gt;允许后台活动、允许关联启动、允许自启动，关闭自动管理。（不同手机型号可能路径不同）</li>
<li>安卓手机必须要将应用锁定到任务栏，保证当前APP一直在后台运行，还需要将APP保持为后台联网状态。</li>
<li>iOS系统手机提醒需前往【设置-&gt;当前APP-&gt;通知】处查看。</li>
</ol>
</div>`
					}
				]
			}
		},
		onShow() {
			
		}, 
		methods: {
			logout(){
				let client_id=uni.getStorageSync('client_id');
				this.$api.LoginApi.logout({client_id:client_id}).then(res => {
					if (res.code == 0) {
						loginStore.logout()
					}
				})
				
			},
			showSetting(){
				uni.showToast({
					title:'请在web端进行资料设置',
					icon:'none'
				})
			},
			editInfo(){
				uni.navigateTo({
					url:"/pages/mine/profile"
				})
			},
			setAvatar(){
				uni.navigateTo({
					url:"/pages/mine/avatar"
				})
			}
		}
	}
</script>

<style>
	.login-logo {
		width: 180rpx;
		height: 180rpx;
		font-size: 80rpx;
		text-align: center;
		line-height: 120rpx;
		border-radius: 18rpx;
	}
	.footer-version{
		width:100%;
		text-align: center;
		position: fixed;
		bottom: 40rpx;
	}
	.app-name{
		font-size:40rpx;
		font-weight: blod;
		text-align: center;
	}
	.app-version{
		text-align: center;
	}
</style>
