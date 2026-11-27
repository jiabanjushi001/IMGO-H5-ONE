package server

func (a *App) setGroupFriendAddPermission(r *request, userID int64, whereUser string) (any, error) {
	if userID <= 1 {
		return nil, r.fail("该账号不支持设置群添加权限")
	}
	value := r.n("group_friend_add")
	if value != 0 && value != 1 {
		return nil, r.fail("群添加权限参数无效")
	}
	if _, err := r.one("SELECT user_id FROM "+a.t("user")+" WHERE "+whereUser, userID); err != nil {
		return nil, err
	}
	if err := update(r.ctx(), a.db, a.t("user"), M{"group_friend_add": value}, whereUser, userID); err != nil {
		return nil, err
	}
	if a.hub != nil {
		a.hub.send([]int64{userID}, "groupFriendAddPermission", M{"group_friend_add": value})
	}
	return M{"group_friend_add": value}, nil
}
