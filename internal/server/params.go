package server

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// decodeForm accepts PHP/qs bracket notation: setting[x], user_ids[0], messages[0][id].
func decodeForm(v url.Values) M {
	root := M{}
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := strings.Split(strings.ReplaceAll(k, "]", ""), "[")
		if len(parts) > 12 {
			continue
		}
		if len(parts) == 1 {
			if len(v[k]) > 0 {
				root[k] = v[k][0]
			}
			continue
		}
		if parts[len(parts)-1] == "" {
			parts = parts[:len(parts)-1]
			setForm(root, parts, v[k])
			continue
		}
		if len(v[k]) > 0 {
			setForm(root, parts, v[k][0])
		}
	}
	return normalizeForm(root).(M)
}
func setForm(m M, parts []string, value any) {
	if len(parts) == 1 {
		m[parts[0]] = value
		return
	}
	child, ok := m[parts[0]].(M)
	if !ok {
		child = M{}
		m[parts[0]] = child
	}
	setForm(child, parts[1:], value)
}
func normalizeForm(v any) any {
	m, ok := v.(M)
	if !ok {
		return v
	}
	for k, val := range m {
		m[k] = normalizeForm(val)
	}
	if len(m) == 0 {
		return m
	}
	max := -1
	for k := range m {
		n, e := strconv.Atoi(k)
		if e != nil || n < 0 || n > 10000 {
			return m
		}
		if n > max {
			max = n
		}
	}
	if max+1 != len(m) {
		return m
	}
	out := make([]any, len(m))
	for k, val := range m {
		n, _ := strconv.Atoi(k)
		out[n] = val
	}
	return out
}
