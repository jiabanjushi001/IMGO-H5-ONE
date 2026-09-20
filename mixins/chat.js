export const chat = {
	data() {
		return {
			
		}
	},
	created: function() {

	},
	methods: {
		// 播放视频,禁止多个同时播放
		handlePlay (item) {
			uni.navigateTo({
				url: '/pages/message/video?name='+item.fileName+'&src='+encodeURI(item.content),
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
			let audioExt=['mp3','wav','acc'];
			let extension = item.content.split('.').pop().toLowerCase();
			if(audioExt.includes(extension) || val==2){
				uni.navigateTo({
					url: '/pages/mine/webview?title=文件预览&src='+encodeURIComponent(item.preview),
					animationType:"slide-in-bottom"
				});
				return;
			}
			// #ifdef APP-PLUS || MP-WEIXIN
			let exts=['doc', 'xls', 'ppt', 'pdf', 'docx', 'xlsx', 'pptx'];
			if(exts.includes(extension)){
				uni.showLoading({title: '文件加载中'});
				uni.downloadFile({
				  url: item.content,
				  success: function (res) {
					uni.hideLoading();
				    var filePath = res.tempFilePath;
				    uni.openDocument({
				      filePath: filePath,
				      showMenu: true,
				      success: function (res) {
				        console.info('打开文档成功');
				      }
				    });
				  },
				  fail() {
				  	uni.hideLoading();
				  }
				});
			}else{
				uni.showToast({
					title:'该文件不支持预览！',
					icon:'none'
				})
			}
			// #endif
			
			// #ifdef H5
			const tempLink = document.createElement("a");
			tempLink.style.display = "none";
			tempLink.href = item.download;
			tempLink.setAttribute("download", item.fileName);
			tempLink.setAttribute("target", "_blank");
			document.body.appendChild(tempLink);
			tempLink.click();
			document.body.removeChild(tempLink);
			// #endif
		},
		// 图片预览
		showImgs : function(e){
			var imgs        = [];
			var imgsCurrent = e.currentTarget.dataset.img;
			for (var i = 0; i < this.messageList.length; i++) {
				if (this.messageList[i].type == 'image' || this.messageList[i].type == 'emoji') {
					imgs.push(this.messageList[i].content);
				}
			}
			uni.previewImage({urls : imgs, current : imgsCurrent});
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
		// 打开用户详情
		openContact(item){
			uni.navigateTo({
				url:"/pages/contacts/detail?id="+item.id
			})
		},
		// 自动解析消息中的表情
		emojiToHtml(str){
			let emojiMap=this.emojiMap;
			return str.replace(/\[!(\w+)\]/gi, function (str, match) {
				var file = match;
				return emojiMap[file] ? "<img class='mr-5' style=\"width:18px;height:18px\" emoji-name=\"".concat(match, "\" src=\"").concat(emojiMap[file], "\" />") : "[!".concat(match, "]");
			  });
		},
		fileSize(size){
			return this.$util.getFileSize(size);
		},
		sendTime:function(mstime){
			return this.$util.timeFormat(mstime);
		},
	}
}
