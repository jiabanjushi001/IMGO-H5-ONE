<template>
    <view style="width:100%">
		<cu-custom bgColor="bg-gradual-green" :isBack="true"  class="cu-header">
			<template #backText></template>
			<template #content>聊天信息</template>
			<template #right>
				<view class="cuIcon-search mr-10 f-16" @tap="switchSearch()"></view>
			</template>
		</cu-custom>
		<view class="cu-bar bg-white search fixed" :style="[{top:CustomBar + 'px',minHeight:'80rpx',justifyContent:'flex-start'}]">
			<view v-for="(tab,index) in tabBars" :key="tab.id" class="uni-tab-item" :id="tab.id" :data-current="index" @click="ontabtap(tab.id)"  v-if="!isSearch">
				<text class="uni-tab-item-title" :class="params.type==tab.id ? 'uni-tab-item-title-active' : ''">{{tab.name}}</text>
			</view>
			<template v-else>
				<view class="search-form round">
					<text class="cuIcon-search"></text>
					<input type="text" v-model="params.keywords" placeholder="请输入关键字搜索" confirm-type="search"/>
					
				</view>
				<view class="action">
					<button class="cu-btn round bg-green" @tap="getMessageList()">搜索</button>
				</view>
			</template>
		</view>
		<scroll-view class="scroll-view-body" ref="scrollView" :scroll-y="true" :scroll-anchoring="true" :style="{height:scrollH+'rpx',position:'fixed',bottom:bottomHeight+'px'}">
			<view class="cu-chat" :style="{paddingTop:'100rpx'}">
			<template v-for="(item,index) in messageList" :key="index" :id="'chatItem_'+index">
				<view class="cu-item">
					<im-user :info="item.fromUser" :profile="isProfile" @longpress="at(item.fromUser)"></im-user>
					<view class="main im-wrap">
						<view class="f-12 c-666" style="width:100%;margin-bottom: 6rpx;">
							<text>{{item.fromUser.realname}} &nbsp;&nbsp;</text>
							<text class="f-11 c-999">{{sendTime(item.sendTime)}}</text>
						</view>
						<view class="im-flex im-rows-reverse self im-align-items-end">
							<!-- 文字消息 -->
							<view v-if="item.type=='text'">
								<view class="content shadow">
									<mp-html container-style="overflow: hidden;display:inline;white-space: pre-wrap" :content="emojiToHtml(item.content)"/>
								</view>
								<view class="message-quote radius-6" v-if="item.extends && item.extends.content">
									{{item.extends.content}}
								</view>
							</view>
							<!-- 图片消息 -->
							<template v-else-if="item.type=='image'">
								<template v-if='item.extends && item.extends.fixMode'>
									<image v-if="item.extends.fixMode<=2"  :src="mediaSrc(item.content)" class="radius" :mode="item.extends.fixMode==1 ? 'widthFix' : 'heightFix' "  :style="item.extends.fixMode==1 ? 'width:200px' : ''" @tap="showImgs"  :data-img="item.content" ></image>
									<image v-if="item.extends.fixMode==3"  :src="mediaSrc(item.content)" class="radius" mode="scaleToFill"  :style="[{width:item.extends.width+'px',height:item.extends.height+'px'}]" @tap="showImgs"  :data-img="item.content" ></image>
								</template>
								<template v-else>
									<image  :src="mediaSrc(item.content)" class="radius" mode="heightFix" @tap="showImgs"  :data-img="item.content" ></image>
								</template>
							</template>
							
							<!-- 语音消息 -->
							<view v-else-if="item.type=='voice'" class="im-voice-msg im-flex im-rows im-nowrap im-align-items-center radius-20" 
							:class="[index == playIndex ? 'linear-green' : '', item.fromUser.id==user.user_id ? 'im-rows-reverse' : '' , ]" :data-voice="item.content" :data-index='index' @tap='playVoice' 
							:style="{'width':(item.extends.duration*3)+'px'}">
								<text class="f-16 cuIcon-subscription rotate45" :class="[index == playIndex ? 'c-white' : '',item.fromUser.id==user.user_id ? 'rotate225' : '']"></text>
								<text class="im-voice-msg-text" :class="[index == playIndex ? 'c-white' : '']">{{item.extends.duration}} "</text> 
							</view>
							<!-- 视频消息 -->
							<view v-else-if="item.type=='video'">
								<view class="course-video radius-10"  @tap="handlePlay(item)">
									<!-- 播放弹层 -->
									<view class="relative-shadow">
										<view class="cuIcon-video icon-center f-28 c-white"></view>
									</view>
									<image v-if="item.extends"  :src="mediaSrc(item.extends.poster)" class="" mode="heightFix"></image>
								</view>
							</view>
							<!-- 文件消息 -->
							<view v-else-if="item.type=='file'">
								<view class="file-card bg-white radius-10 im-flex im-justify-content-start pd-10 im-align-items-center"  @tap.stop="previewFile(item)">
									<AuthImage :src="item.extUrl" mode="aspectFit" style="width:64rpx;height:80rpx"></AuthImage>
									<view class="im-flex im-columns ml-10">
										<view class="text-overflow file-name">{{item.fileName}}</view>
										<view class="text-gray file-size f-12">{{fileSize(item.fileSize)}}</view>
									</view>
								</view>
							</view>
							<!-- 音视频消息 -->
							<view v-else-if="item.type=='webrtc'" class="im-voice-msg im-flex im-rows im-nowrap im-align-items-center radius-20" :class="[item.fromUser.id==user.user_id ? 'im-rows-reverse' : '' , ]">
								<text class="f-16" :class="[item.extends.type == 1 ? 'cuIcon-record' : 'cuIcon-dianhua',item.fromUser.id==user.user_id ? 'rotate180' : '']"></text>
								<text class="im-voice-msg-text">{{item.content}}</text> 
							</view>
							<!-- 位置消息 -->
							<view v-else-if="item.type=='location'" @tap="openLocation(item.extends)" class="im-location-msg im-flex im-rows im-nowrap im-align-items-center radius-8 pd-10">
								<view class="f-24 cuIcon-location pr-5"></view>
								<view>
									<view class="f-14 mb-5">{{item.content}}</view>
									<view class="c-999 f-12">{{item.extends && item.extends.address}}</view> 
								</view>
								
							</view>
							<!-- 名片消息 -->
							<view v-else-if="item.type=='contact'" @tap="openContact(item.extends)" class="im-contact-msg radius-8 pt-10 pr-10 pl-10 pb-5">
								<view class="im-flex im-rows im-nowrap im-align-items-center">
								<view class='cu-avatar mr-10 radius'>
									<AuthImage v-if="item.extends && item.extends.avatar" :src="item.extends.avatar" :info="item.extends" avatar mode="aspectFill" style="width:100%;height:100%;" />
								</view>
									<view class="c-333">{{item.extends.displayName}}</view>
								</view>
								<hr class="mt-10 c-999">
								<view class="c-666 f-10">
									个人名片
								</view>
							</view>
							<!-- 其他消息 -->
							<imItem v-else :item="item" :index="index" :isSelf="true"></imItem>
						</view>
					</view>
				</view>
			</template>
			
		</view>
		<Empty v-if="!messageList.length" noDatatext="暂无记录" textcolor="#999" />
		<view class="im-pagination">
			<uni-pagination :total="dataTotal" :pageSize="params.limit" @change="changePage" title="标题文字" />
		</view>
		
		</scroll-view>
		<!-- 文件预览 -->
		<view class="cu-modal bottom-modal" :class="modelName=='preview'?'show':''" @tap="modelName=''">
			<view class="cu-dialog" v-if="modelName=='preview'">
				<view class="cu-list menu bg-white">
					<view class="cu-item" @tap="preview(1)" >
						<view class="content padding-tb-sm">
							<text class="text-center">本地预览(需下载)</text>
							<view class="text-gray text-sm">需下载，仅支持office类型文件</view>
						</view>
					</view>
					<view class="cu-item" @tap="preview(2)">
						<view class="content padding-tb-sm">
							<text>在线预览</text>
							<view class="text-gray text-sm">支持常用的文件和文档</view>
						</view>
					</view>
					</view>
			</view>
		</view>
	</view>
</template>
<script>
	import imItem from '@/components/message/im-item.vue';
	import imUser from '@/components/im-user.vue';
	import { chat } from '@/mixins/chat.js'
	import { useloginStore } from '@/store/login';
	import { useMsgStore } from '@/store/message';
	import { storeToRefs } from 'pinia';
	import pinia from '@/store/index'
	const msgStore = useMsgStore(pinia)
	const {newMessage,msgList,getContact,appendMsg,checkMsg,unread} = storeToRefs(msgStore);
	const userStore = useloginStore(pinia)
    export default {
		components: {
			imItem,
			imUser
		},
		mixins:[chat],
        data() {
            return {
				user:userStore.userInfo,
                modelName:'',
				messageList:[],
                tabBars: [{
                    name: '全部',
                    id: 'all'
                }, {
                    name: '文本',
                    id: 'text'
                }, {
                    name: '图片',
                    id: 'image'
                }, {
                    name: '视频',
                    id: 'video'
                }, {
                    name: '文件',
                    id: 'file'
                }],
				scrollInto:'',
				isSearch:false,
				dataTotal:0,
				emojiMap:[],
				isProfile:false,
				bottomHeight:0,
				paddingT:0,
                params:{
					toContactId: 0,
					is_group:0,
					type: "all",
					keywords: "",
					page: 1,
					limit: 10
				}
            }
        },
		computed:{
			scrollH:function(){
				let sys = uni.getSystemInfoSync();
				let winWidth = sys.windowWidth;
				let winrate = 750/winWidth;
				let bottomHeight=uni.upx2px(100);
				this.bottomHeight=bottomHeight;
				const query=this.getQuery();
				
				setTimeout(() => {
					query.select('.cu-header').boundingClientRect();
					query.exec(data => {
						this.paddingT=data[0].height;
					});
				}, 10)
				// #ifdef H5
				let winHeight =parseInt((sys.windowHeight - this.navBarHeight - this.paddingT)*winrate);
				// #endif
				
				// #ifdef APP-PLUS
				let winHeight =parseInt((sys.windowHeight - (this.inlineTools + this.paddingT+bottomHeight))*winrate);
				// #endif
				
				// #ifndef H5 || APP-PLUS
				this.bottomHeight+=this.inlineTools;
				// 微信小程序需要减去状态栏+底部导航栏+底部横线
				let winHeight =parseInt((sys.windowHeight-(this.inlineTools + this.paddingT + this.navBarHeight))*winrate)
				// #endif
				return winHeight+18
			}
		},
        onLoad: function(options) {
        	this.params.toContactId = options.id;
        	let contact=msgStore.getContact(options.id);
			this.params.is_group=contact.is_group;
        	if(!contact){
        		uni.showToast({
        			title:'联系人不存在',
        			icon:'none',
        			duration:1500,
        			complete:(res)=>{
        				uni.reLaunch({
        					url: '/pages/index/index'
        				})
        			}
        		})
        		return;
        	}
        	this.contact=contact;
			this.getMessageList();
			if(contact.is_group==1 && (contact.role<3 || contact.setting.profile=='1')){
				this.isProfile=true;
			}
        },
        methods: {
			getQuery(){
				// #ifdef MP
				const query = uni.createSelectorQuery().in(this);
				// #endif
				
				// #ifndef MP
				const query = uni.createSelectorQuery();
				// #endif
				return query;
			},
			getMessageList() {
				this.$api.msgApi.getMessageList(this.params).then(res => {
					let data=res.data;
					this.messageList = data;
					this.dataTotal=res.count;
				})
			},
            ontabtap(type) {
				if (this.params.type === type) {
				    return;
				}
				this.params.type=type;
				this.params.page=1;
				this.scrollInto = type;
            	this.getMessageList();
            },
			changePage(e){
				this.params.page=e.current;
				this.getMessageList();
			},
			search(){
				this.getMessageList();
			},
			switchSearch(){
				this.params.keywords='';
				this.isSearch=!this.isSearch;
				if(!this.isSearch){
					this.getMessageList()
				}
			},
        }
    }
</script>

<style  lang="scss">
    /* #ifndef APP-PLUS */
    page {
        width: 100%;
        min-height: 100%;
        display: flex;
    }

    /* #endif */

    #tab-bar {
        background-color: #ffffff;
    }

    .scroll-h {
		width:100%;
		height: 80rpx;
        flex-direction: row;
        /* #ifndef APP-PLUS */
        white-space: nowrap;
        /* #endif */
        /* flex-wrap: nowrap; */
        /* border-color: #cccccc;
		border-bottom-style: solid;
		border-bottom-width: 1px; */
    }

    .line-h {
        height: 1rpx;
        background-color: #cccccc;
    }

    .uni-tab-item {
        /* #ifndef APP-PLUS */
        display: inline-block;
        /* #endif */
        flex-wrap: nowrap;
        padding-left: 34rpx;
        padding-right: 34rpx;
    }

    .uni-tab-item-title {
        color: #555;
        font-size: 30rpx;
        height: 80rpx;
        line-height: 80rpx;
        flex-wrap: nowrap;
        /* #ifndef APP-PLUS */
        white-space: nowrap;
        /* #endif */
    }

    .uni-tab-item-title-active {
        color: #007AFF;
    }

	page{
	  padding-bottom: 100upx;
	}
	
	#more-oprate{
		min-height:100%;
		justify-content: flex-end;
		flex-direction: column;
	}
	.cu-chat{
		background-color: #f1f1f1;
	}
	.cu-chat .cu-item.self {
	    justify-content: flex-start;
	    text-align: right;
	}
	.bg-light-green{
		background-color: #95ec69;
	}
	.im{padding:30rpx;}
	.im-system-msg{color:#FFFFFF; font-size:26rpx; line-height:38rpx; padding:5px 10px; display:block; border-radius:6rpx;}
	.im-msg{margin-bottom:28px; display:flex; flex-direction:row; flex-wrap:nowrap;}
	.im-voice-msg{height:80rpx; padding:0 20rpx; background-color:#E7F0F3; color:#2B2E3D; min-width:160rpx; max-width:400rpx;}
	.im-voice-msg-text{font-size:22rpx; margin:0 5rpx;}
	.im-location-msg{ background-color:#E7F0F3; color:#2B2E3D;text-align: left !important;}
	.im-contact-msg{ width:360rpx; background-color:#E7F0F3; color:#2B2E3D;text-align: left !important;}
	.cu-chat .cu-item{
		padding:20rpx 20rpx 0 20rpx !important;
	}
	.cu-chat .cu-item:last-child{
		padding:20rpx !important;
	}
	.cu-chat .cu-item>.main {
	    margin: 0 0.8rem;
	    display: flex;
	    align-items: center;
		}
	.cu-chat .cu-item>.main .content{
		padding:10rpx 20rpx;
		min-height:60rpx;
	}
	.course-video{
		max-width:400rpx;
		overflow: hidden;
		position: relative;
		max-height: 360rpx;
	}
	
	.video-model{
		background-color: #3838388f;z-index:10000;width: 100%;height: 100%;position: fixed;top:0;overflow:hidden;;
	}
	.close-model{
		position: absolute;top:180rpx;right:20rpx;background-color: #3838388f;padding:4rpx 10rpx
	}
	.video-box{width:100%}
	.icon-center{
			position: absolute;
		    top: 50%;
		    z-index: 4;
		    transform: translate(-50%, -50%);
		    left: 50%;
			padding: 0 4rpx 0 6rpx;
	}
	.relative-shadow{
		position: absolute;width:100%;height:100%;background: #83838387;z-index:1;
	}
	.file-card{
		width:420rpx;
		height:120rpx;
		.file-icon{
			width:60rpx;
			height:80rpx;
		}
		.file-name{
			text-align: left !important;
			width:300rpx;
		}
		.file-size{
			text-align: left !important;
			margin-top:8rpx;
		}
	}
	
	.icon-spin{
		animation: spin 1s linear infinite;
	}
	@keyframes spin {
	  to {
	    transform: rotate(360deg);
	  }
	}
	
	.main .content ::v-deep uni-text , 
	.main .content ::v-deep uni-text span,
	 .main .content ::v-deep text,
	  .main .content ::v-deep uni-rich-text{
		word-wrap: break-word !important;
		word-break: break-all !important;
	}
	
	.main .content ::v-deep ._block ._a{
		pointer-events: none !important;
	}
	
	.text-container{
		-webkit-user-select:text !important;
		user-select:text !important;
		font-size:48rpx;
		word-wrap: break-word !important;
		text-align: left;
		line-height: 1.5;
		letter-spacing: 1.2px;
		color:#333;
	}
	.read-status{
		font-weight: 600;
	}
	
	
	
	</style>
	
	<style lang="scss" scoped>
		.message-quote{
			padding:8rpx;
			font-size:24rpx;
			margin-top:16rpx;
			background-color: #e3e3e3;
			overflow: hidden !important;
			text-overflow: ellipsis;
			white-space: nowrap !important;
			max-width:380rpx;
			text-align: left;
		}
		// 设置表情图片居中
		::v-deep .emoji-image{
			vertical-align: text-top !important;
		}
		
		.cu-chat ::v-deep .cu-item {
		    padding: 20rpx;
		}
		.cu-chat ::v-deep .cu-item:last-child{
		    padding-bottom:60rpx;
		}
		.back-unread{
			background-color: #e3e3e3;
			padding:4rpx 10rpx;
			border-radius: 50%;
			font-size: 22rpx;
		}
		uni-page-body {
		    padding-bottom: 0;
		}
		.im-pagination{
			z-index:10;
			position: fixed;
			bottom: 0;
			padding:20rpx;
			background-color: #fff;
			width:100%;
		}
		.cu-bar.fixed, .nav.fixed{
			z-index:998 !important;
		}
</style>
