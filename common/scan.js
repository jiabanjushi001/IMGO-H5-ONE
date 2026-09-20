import {
	postJsonRequest,
} from '@/utils/request.js';

const scanPath = (value) => {
	try {
		const url = new URL(value);
		if (!['http:', 'https:'].includes(url.protocol)) return '';
		const path = url.pathname;
		return /^\/scan\/[ug]\/[a-zA-Z0-9._-]+$/.test(path) ? path : '';
	} catch (e) {
		return '';
	}
};

const verifyQr=(path, replace=false)=>{
	const lastPart = path.split('/').pop();
	// Only the path is sent to our configured API; the QR host is never contacted.
	return postJsonRequest(path,{realToken:lastPart}).then((res)=>{
		if(res.code==0){
			switch(res.data.action){
				case 'groupInfo':
					uni[replace ? 'redirectTo' : 'navigateTo']({
						url: '/pages/message/group/info?group_id='+ encodeURIComponent(res.data.id) + '&token=' + encodeURIComponent(res.data.token || lastPart)
					})
				break;
				case 'userInfo':
					uni.navigateTo({
						url:"/pages/contacts/detail?id="+res.data.id
					})
				break;
			}
		}
	})
}

const scanQr=()=>{
	// #ifndef H5
	uni.scanCode({
		success: function (res) {
			checkQr(res.result);
		}
	});
	// #endif
	// #ifdef H5
	 uni.navigateTo({
	 	url:'/pages/index/scan'
	 })
	// #endif	
}

const checkQr=(data, replace=false)=>{
	const path = typeof data === 'string' ? scanPath(data) : '';
	if(path){
		return verifyQr(path, replace);
	}else{
		uni.showModal({
			title: '已识别内容',
			content: data,
			confirmText:'复制内容',
			success: function (e) {
				if (e.confirm) {
					uni.setClipboardData({
						data: data,
						success: function () {
							uni.showToast({
								title:'复制成功',
								icon:'none'
							})
						}
					});
				} 
			}
		});
	}
}

export default {
	scanQr,
	checkQr
}
