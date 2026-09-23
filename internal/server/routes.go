package server

import "strings"

func (a *App) register() {
	a.routes["/manage/index/overview"] = endpoint{handler: a.overview, permission: "manage.overview"}
	add := func(prefix string, public, super bool, actions string, h func(*request) (any, error)) {
		for _, act := range strings.Fields(actions) {
			a.routes[normalizedPath(prefix+"/"+act)] = endpoint{handler: h, public: public, super: super}
		}
	}
	addManage := func(prefix, permission, actions string, h func(*request) (any, error)) {
		for _, act := range strings.Fields(actions) {
			a.routes[normalizedPath(prefix+"/"+act)] = endpoint{handler: h, permission: permission}
		}
	}
	add("/common/pub", true, false, "login register getSystemInfo sendCode checkVersion", a.public)
	add("/common/pub", false, false, "logout avatar bindUid offline bindGroup", a.public)
	add("/common/api", true, false, "createUser login", a.api)
	add("/common/upload", false, false, "uploadFile uploadImage uploadAvatar uploadEmoji", a.upload)
	add("/enterprise/im", false, false, "getContacts getContactInfo sendMessage forwardMessage getUserInfo searchUser userList getMessageList setMsgIsRead setting undoMessage removeMessage isNotice setChatTop delChat sendToMsg editPassword updateUserInfo editAccount readAtMsg getAdminNotice delMessage", a.im)
	add("/enterprise/friend", false, false, "index add update del setNickname getApplyMsg", a.friend)
	add("/enterprise/group", false, false, "getAllUser groupUserList groupInfo editGroupName editGroupAvatar addGroupUser setManager add removeUser setNoSpeak removeGroup setNotice groupSetting joinGroup changeOwner clearMessage", a.group)
	add("/enterprise/emoji", false, false, "index add del move", a.emoji)
	add("/enterprise/bank", false, false, "get save", a.bankCard)
	add("/enterprise/wallet", false, false, "status withdraw history entries", a.wallet)
	add("/enterprise/checkin", false, false, "status submit", a.checkIn)
	add("/enterprise/invite", false, false, "status", a.inviteStatus)
	addManage("/manage/bank", "manage.bank", "index detail edit", a.manageBankCard)
	addManage("/manage/wallet", "manage.finance", "index detail account credit review recharges entries recharge withdraw", a.manageWallet)
	add("/enterprise/files", false, false, "index", a.files)
	addManage("/manage/files", "manage.files", "index", a.files)
	addManage("/manage/user", "manage.users", "index detail checkInHistory", a.manageUser)
	addManage("/manage/user", "manage.users", "add edit setRemark setInviteCode del setStatus editPassword", a.manageUser)
	add("/manage/user", false, true, "setRole", a.manageUser)
	addManage("/manage/group", "manage.groups", "index changeOwner del addGroupUser delGroupUser setManager", a.manageGroup)
	addManage("/manage/config", "manage.settings", "getInfo getConfig getAllConfig setConfig getInviteLink sendTestEmail", a.manageConfig)
	addManage("/manage/index", "manage.overview", "clearMessage noticeList delNotice publishNotice", a.manageIndex)
	addManage("/manage/message", "manage.messages", "index getContacts dealMsg", a.manageMessage)
	addManage("/manage/task", "manage.settings", "getTaskList startTask stopTask setTaskConfig getTaskLog clearTaskLog", a.task)
	add("/manage/role", false, true, "index detail save setStatus del permissions", a.manageRole)
	add("/manage/agentSetting", false, true, "detail save", a.manageAgentSetting)
	add("/index/install", true, false, "index getEnv version checkDatabase install progress", func(r *request) (any, error) {
		return nil, clientError{"Go 服务使用命令行部署，已关闭网页安装器", 410}
	})
	add("/index/index", true, false, "index view avatar download scanQr downapp downloadApp", a.index)
	a.routes["/view"] = endpoint{handler: a.index, public: true}
	a.routes["/downapp"] = endpoint{handler: a.index, public: true}
}
func action(r *request) string {
	p := effectivePath(r.c)
	parts := strings.Split(strings.Trim(p, "/"), "/")
	return strings.ToLower(parts[len(parts)-1])
}
