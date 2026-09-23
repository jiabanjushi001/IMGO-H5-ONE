package server

import (
	"net/url"
	"strings"
)

// inviteURL appends the generated token to a configurable registration URL.
func (a *App) inviteURL(token string) string {
	if a.cfg.InviteURL != "" {
		return a.cfg.InviteURL + token
	}
	return a.url("/index.html/#/register?inviteCode=" + token)
}

// scanURL chooses the address encoded into QR images. It may differ from the
// API's loopback address when a phone opens the development preview over LAN.
func (a *App) scanURL(kind, token string) string {
	base := strings.TrimSpace(a.cfg.QRBaseURL)
	if base == "" {
		base = a.cfg.BaseURL
	}
	return strings.TrimRight(base, "/") + "/scan/" + kind + "/" + token
}

// groupInviteURL keeps the signed invitation in the H5 hash route, where the
// logged-in client can preview the group and submit it to joinGroup.
func (a *App) groupInviteURL(groupID, token string) string {
	base := strings.TrimSpace(a.cfg.H5URL)
	if base == "" {
		base = strings.SplitN(strings.TrimSpace(a.cfg.InviteURL), "#", 2)[0]
	}
	if base == "" {
		base = a.cfg.BaseURL
	}
	return strings.TrimRight(base, "/") + "/#/pages/message/group/info?group_id=" + url.QueryEscape(groupID) + "&token=" + url.QueryEscape(token)
}
