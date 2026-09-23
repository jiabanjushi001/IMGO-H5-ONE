package server

import (
	"context"
	"strings"
	"time"
)

// Stream authorised ciphertext rows. This supports old data without persisting plaintext indexes.
func (a *App) searchEncrypted(r *request, where string, args []any, keyword string) ([]M, error) {
	ctx, cancel := context.WithTimeout(r.ctx(), 30*time.Second)
	defer cancel()
	limit, offset := r.pagination()
	result := []M{}
	keyword = strings.ToLower(keyword)
	cursor := int64(0)
	count := int64(0)
	for {
		q := "SELECT * FROM " + a.t("message") + " WHERE " + where
		vs := append([]any{}, args...)
		if cursor > 0 {
			q += " AND msg_id<?"
			vs = append(vs, cursor)
		}
		batch, e := rows(ctx, a.db, q+" ORDER BY msg_id DESC LIMIT 256", vs...)
		if e != nil {
			return nil, e
		}
		if len(batch) == 0 {
			break
		}
		for _, m := range batch {
			plain, e := decryptContent(a.cfg.ChatKey, str(m["content"]))
			if e != nil {
				return nil, e
			}
			if strings.Contains(strings.ToLower(plain), keyword) {
				if count >= offset && int64(len(result)) < limit {
					result = append(result, m)
				}
				count++
			}
		}
		cursor = number(batch[len(batch)-1]["msg_id"])
	}
	r.count = count
	return result, nil
}
