package server

func (a *App) qrCode(r *request) (any, error) {
	switch action(r) {
	case "my":
		return M{"qrUrl": a.scanURL("u", a.hashID(r.uid()))}, nil
	default:
		return nil, r.fail("未知二维码操作")
	}
}
