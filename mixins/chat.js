import {
	normalizeMediaUrl,
	needsAuthMediaUrl,
	resolveMediaDisplayUrl,
	downloadAuthedFile,
	triggerAuthedDownload
} from '@/utils/avatar.js'

export const chat = {
	data() {
		return {
			mediaUrlMap: {},
			playIndex: -1
		}
	},
	created: function() {

	},
	methods: {
		sendTime(time){
			return this.$util ? this.$util.timeFormat(time) : time
		},
		mediaSrc(url){
			if (!url) return ''
			if (/^(?:blob:|data:)/i.test(String(url))) return url
			const abs = normalizeMediaUrl(url)
			if (!abs) return url
			if (this.mediaUrlMap && this.mediaUrlMap[abs]) return this.mediaUrlMap[abs]
			if (needsAuthMediaUrl(abs)) {
				this.ensureMediaUrl(abs)
				return (this.mediaUrlMap && this.mediaUrlMap[abs]) || ''
			}
			return abs
		},
		ensureMediaUrl(abs){
			if (!abs || !needsAuthMediaUrl(abs)) return
			resolveMediaDisplayUrl(abs).then((blobUrl) => {
				if (!this.mediaUrlMap) this.mediaUrlMap = {}
				if (this.mediaUrlMap[abs] === blobUrl) return
				this.mediaUrlMap = { ...this.mediaUrlMap, [abs]: blobUrl }
			}).catch(() => {})
		},
		// 播放视频,禁止多个同时播放
		handlePlay (item) {
			uni.navigateTo({
				url: '/pages/message/video?name='+encodeURIComponent(item.fileName || 'video')+'&src='+encodeURIComponent(item.content),
				animationType:"slide-in-bottom"
			});
		},
		// 文件预览
		previewFile(item){
			if(this.islongPress){
				return;
			}
			this.curMsg=item;
			this.modelName='preview';
		},
		preview(val){
			let item=this.curMsg;
			if (!item) return
			let audioExt=['mp3','wav','acc'];
			let extension = String(item.content || '').split('.').pop().toLowerCase().split('?')[0];
			if(audioExt.includes(extension) || val==2){
				const previewUrl = item.preview || item.content
				if (needsAuthMediaUrl(previewUrl)) {
					uni.showLoading({ title: '文件加载中' })
					downloadAuthedFile(previewUrl).then(({ displayUrl, tempFilePath }) => {
						uni.hideLoading()
						if (typeof document !== 'undefined' && /^blob:/i.test(displayUrl)) {
							uni.navigateTo({
								url: '/pages/mine/webview?title=文件预览&src='+encodeURIComponent(displayUrl),
								animationType:"slide-in-bottom"
							})
							return
						}
						uni.openDocument({
							filePath: tempFilePath,
							showMenu: true,
							fail: () => {
								uni.navigateTo({
									url: '/pages/mine/webview?title=文件预览&src='+encodeURIComponent(displayUrl),
									animationType:"slide-in-bottom"
								})
							}
						})
					}).catch(() => {
						uni.hideLoading()
						uni.showToast({ title: '文件加载失败', icon: 'none' })
					})
					return
				}
				uni.navigateTo({
					url: '/pages/mine/webview?title=文件预览&src='+encodeURIComponent(previewUrl),
					animationType:"slide-in-bottom"
				});
				return;
			}
			// #ifdef APP-PLUS || MP-WEIXIN
			let exts=['doc', 'xls', 'ppt', 'pdf', 'docx', 'xlsx', 'pptx'];
			if(exts.includes(extension)){
				uni.showLoading({title: '文件加载中'});
				downloadAuthedFile(item.content).then(({ tempFilePath }) => {
					uni.hideLoading();
					uni.openDocument({
						filePath: tempFilePath,
						showMenu: true,
						success: function () {
							console.info('打开文档成功');
						}
					});
				}).catch(() => {
					uni.hideLoading();
					uni.showToast({ title: '文件加载失败', icon: 'none' })
				});
			}else{
				uni.showToast({
					title:'该文件不支持预览！',
					icon:'none'
				})
			}
			// #endif
			
			// #ifdef H5
			triggerAuthedDownload(item.download || item.content, item.fileName || 'file').catch(() => {
				uni.showToast({ title: '下载失败', icon: 'none' })
			})
			// #endif
		},
		// 图片预览
		async showImgs (e){
			const rawCurrent = e.currentTarget.dataset.img;
			const items = (this.messageList || []).filter((m) => m.type === 'image' || m.type === 'emoji')
			const urls = await Promise.all(items.map((m) => resolveMediaDisplayUrl(m.content)))
			const current = await resolveMediaDisplayUrl(rawCurrent)
			const validUrls = urls.filter(Boolean)
			if (!validUrls.length) return
			uni.previewImage({
				urls: validUrls,
				current: current || validUrls[0]
			});
		},
		// 播放语音
		playVoice (e) {
			const voicelUrl = e.currentTarget.dataset.voice;
			const index = e.currentTarget.dataset.index;
			resolveMediaDisplayUrl(voicelUrl).then((url) => {
				if (!url) {
					uni.showToast({ title: '语音加载失败', icon: 'none' })
					return
				}
				if (typeof this.playNow === 'function') {
					if (this.playIndex == -1){
						return this.playNow(url, index);
					}
					if (this.playIndex == index) {
						this.playIndex = -1;
						return
					}
					this.playIndex = -1;
					return this.playNow(url, index);
				}
				if (this._voiceCtx) {
					try { this._voiceCtx.stop() } catch (err) {}
				}
				if (this.playIndex == index) {
					this.playIndex = -1
					return
				}
				const ctx = uni.createInnerAudioContext()
				this._voiceCtx = ctx
				ctx.src = url
				ctx.autoplay = true
				this.playIndex = index
				ctx.onEnded(() => { this.playIndex = -1 })
				ctx.onStop(() => { this.playIndex = -1 })
				ctx.onError(() => {
					this.playIndex = -1
					uni.showToast({ title: '语音播放失败', icon: 'none' })
				})
			}).catch(() => {
				uni.showToast({ title: '语音加载失败', icon: 'none' })
			})
		},
		openLocation(item){
			uni.openLocation({
				latitude: item.latitude,
				longitude: item.longitude,
				success: function () {
					console.log('success');
				}
			});
		},
		openContact(item){
			uni.navigateTo({
				url:"/pages/contacts/detail?id="+item.id
			})
		},
		emojiToHtml(str){
			let emojiMap=this.emojiMap;
			return String(str || '').replace(/\[!(\w+)\]/gi, function (str, match) {
				var file = match;
				return emojiMap[file] ? "<img class='mr-5' style=\"width:18px;height:18px\" emoji-name=\"".concat(match, "\" src=\"").concat(emojiMap[file], "\" />") : "[!".concat(match, "]");
			  });
		},
		fileSize(size){
			if (this.$util && typeof this.$util.getFileSize === 'function') {
				return this.$util.getFileSize(size)
			}
			if (!size && size !== 0) return ''
			if (size < 1024) return size + 'B'
			if (size < 1024 * 1024) return (size / 1024).toFixed(1) + 'KB'
			return (size / 1024 / 1024).toFixed(1) + 'MB'
		}
	}
}
