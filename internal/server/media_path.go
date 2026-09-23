package server

import (
	"net/url"
	"strings"
)

// mediaPath returns a domain-independent URL for resources served by IMGO.
// External resources keep their origin; navigation/invitation URLs use url().
func (a *App) mediaPath(value string) string {
	if value == "" {
		return ""
	}
	u, err := url.Parse(value)
	if err != nil {
		return value
	}
	p := strings.TrimPrefix(u.EscapedPath(), "./")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	local := false
	for _, prefix := range []string{"/storage/", "/avatar/", "/filedown/", "/static/", "/upload/", "/uploads/"} {
		if strings.HasPrefix(p, prefix) {
			local = true
			break
		}
	}
	if u.Scheme != "" || u.Host != "" {
		if (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "") || !local {
			return value
		}
	}
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		p += "#" + u.EscapedFragment()
	}
	return p
}

func (a *App) mediaExtensions(kind string, v M) M {
	if kind == "video" {
		v["poster"] = a.videoPoster(str(v["poster"]))
	}
	return v
}
