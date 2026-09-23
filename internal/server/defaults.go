package server

func defaultConfig(name string) M {
	switch name {
	case "sysInfo":
		return M{"logo": "", "name": "Imgo", "state": "1", "regauth": "0", "regtype": "0", "runMode": "1", "ipregion": "0", "closeTips": "系统维护中", "description": "Go Gin 即时通信", "registerInterval": "600", "showScan": "1", "showGroupQr": "1"}
	case "chatInfo":
		return M{"stun": "", "online": "1", "webrtc": "0", "dbDelMsg": "1", "msgClear": "0", "redoTime": "120", "stunPass": "", "stunUser": "", "groupChat": "1", "simpleChat": "1", "msgClearDay": "30", "groupUserMax": "0", "sendInterval": "1", "autoAddUser": M{"status": "0", "welcome": "欢迎使用 Imgo", "user_ids": []any{}, "user_items": []any{}}, "autoAddGroup": M{"name": "交流群", "status": "0", "userMax": "100", "owner_uid": 1, "owner_info": []any{}}}
	case "fileUpload":
		return M{"disk": "local", "size": "50", "preview": "", "fileExt": []any{"jpg", "jpeg", "ico", "webp", "bmp", "gif", "pdf", "mp3", "wav", "amr", "mp4", "mov", "ppt", "pptx", "doc", "docx", "xls", "xlsx", "txt", "md", "png", "zip"}, "qiniu": M{"url": "", "bucket": "", "accessKey": "", "secretKey": ""}, "aliyun": M{"url": "", "bucket": "", "accessId": "", "endpoint": "", "accessSecret": ""}, "qcloud": M{"cdn": "", "appId": "", "bucket": "", "region": "", "secretId": "", "secretKey": ""}}
	case "smtp":
		return M{"addr": "", "host": "", "pass": "", "port": "587", "sign": "Imgo", "security": "tls"}
	case "compass":
		return M{"list": []any{}, "mode": 1, "status": 0}
	}
	return M{}
}
