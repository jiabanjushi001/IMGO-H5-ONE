package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type M map[string]any
type DB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func rows(ctx context.Context, db DB, q string, args ...any) ([]M, error) {
	r, e := db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	cols, e := r.Columns()
	if e != nil {
		return nil, e
	}
	types, e := r.ColumnTypes()
	if e != nil {
		return nil, e
	}
	out := []M{}
	for r.Next() {
		vals := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		if e = r.Scan(ptr...); e != nil {
			return nil, e
		}
		m := M{}
		for i, k := range cols {
			v := vals[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
				t := types[i].DatabaseTypeName()
				if strings.Contains(t, "INT") || t == "DECIMAL" {
					if n, e := strconv.ParseInt(string(b), 10, 64); e == nil {
						v = n
					}
				}
				if t == "JSON" {
					var j any
					if json.Unmarshal(b, &j) == nil {
						v = j
					}
				}
			}
			m[k] = v
		}
		out = append(out, m)
	}
	return out, r.Err()
}
func one(ctx context.Context, db DB, q string, args ...any) (M, error) {
	r, e := rows(ctx, db, q, args...)
	if e != nil {
		return nil, e
	}
	if len(r) == 0 {
		return nil, sql.ErrNoRows
	}
	return r[0], nil
}
func number(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		n, _ := x.Int64()
		return n
	case string:
		if x == "true" {
			return 1
		}
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case []byte:
		n, _ := strconv.ParseInt(string(x), 10, 64)
		return n
	case bool:
		if x {
			return 1
		}
	}
	return 0
}
func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
func obj(v any) M {
	switch x := v.(type) {
	case M:
		return x
	case map[string]any:
		return M(x)
	case string:
		var m M
		if json.Unmarshal([]byte(x), &m) == nil && m != nil {
			return m
		}
	}
	return M{}
}
func js(v any) string { b, _ := json.Marshal(v); return string(b) }
func ids(v any) []int64 {
	out := []int64{}
	seen := map[int64]bool{}
	add := func(x any) {
		n := number(x)
		if n > 0 && !seen[n] {
			out = append(out, n)
			seen[n] = true
		}
	}
	switch a := v.(type) {
	case []any:
		for _, x := range a {
			add(x)
		}
	case []string:
		for _, x := range a {
			add(x)
		}
	case string:
		var b []any
		if json.Unmarshal([]byte(a), &b) == nil {
			for _, x := range b {
				add(x)
			}
		} else {
			for _, x := range strings.Split(a, ",") {
				add(x)
			}
		}
	default:
		add(a)
	}
	return out
}
func marks(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }
func values(ns []int64) []any {
	v := make([]any, len(ns))
	for i, n := range ns {
		v[i] = n
	}
	return v
}
func insert(ctx context.Context, db DB, table string, m M) (int64, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vs := make([]any, 0, len(keys))
	for _, k := range keys {
		vs = append(vs, m[k])
	}
	r, e := db.ExecContext(ctx, "INSERT INTO "+table+" (`"+strings.Join(keys, "`,`")+"`) VALUES ("+marks(len(keys))+")", vs...)
	if e != nil {
		return 0, e
	}
	return r.LastInsertId()
}
func update(ctx context.Context, db DB, table string, m M, where string, args ...any) error {
	if len(m) == 0 {
		return errors.New("没有可更新的字段")
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	set := []string{}
	vs := []any{}
	for _, k := range keys {
		set = append(set, "`"+k+"`=?")
		vs = append(vs, m[k])
	}
	_, e := db.ExecContext(ctx, "UPDATE "+table+" SET "+strings.Join(set, ",")+" WHERE "+where, append(vs, args...)...)
	return e
}
func pick(m M, keys ...string) M {
	r := M{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			r[k] = v
		}
	}
	return r
}
func (a *App) t(name string) string { return "`" + a.cfg.Prefix + name + "`" }
