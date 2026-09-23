package server

func (a *App) walletEntries(r *request, userID int64) (any, error) {
	n, err := r.one("SELECT COUNT(*) n FROM "+a.t("imgo_wallet_entry")+" WHERE user_id=?", userID)
	if err != nil {
		return nil, err
	}
	r.count = number(n["n"])
	limit, offset := r.pagination()
	return r.list("SELECT entry_id,event,available_delta,pending_delta,reference_id,note,created_at FROM "+a.t("imgo_wallet_entry")+" WHERE user_id=? ORDER BY entry_id DESC LIMIT ? OFFSET ?", userID, limit, offset)
}
