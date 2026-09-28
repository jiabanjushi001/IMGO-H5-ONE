package server

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Cloudflare publishes these origin-facing networks at https://www.cloudflare.com/ips/.
// CF-Connecting-IP is accepted only when Nginx's sanitized X-Forwarded-For value
// is an address in one of these networks, so direct clients cannot spoof it.
var cloudflarePrefixes = []netip.Prefix{
	netip.MustParsePrefix("103.21.244.0/22"),
	netip.MustParsePrefix("103.22.200.0/22"),
	netip.MustParsePrefix("103.31.4.0/22"),
	netip.MustParsePrefix("104.16.0.0/13"),
	netip.MustParsePrefix("104.24.0.0/14"),
	netip.MustParsePrefix("108.162.192.0/18"),
	netip.MustParsePrefix("131.0.72.0/22"),
	netip.MustParsePrefix("141.101.64.0/18"),
	netip.MustParsePrefix("162.158.0.0/15"),
	netip.MustParsePrefix("172.64.0.0/13"),
	netip.MustParsePrefix("173.245.48.0/20"),
	netip.MustParsePrefix("188.114.96.0/20"),
	netip.MustParsePrefix("190.93.240.0/20"),
	netip.MustParsePrefix("197.234.240.0/22"),
	netip.MustParsePrefix("198.41.128.0/17"),
	netip.MustParsePrefix("2400:cb00::/32"),
	netip.MustParsePrefix("2405:8100::/32"),
	netip.MustParsePrefix("2405:b500::/32"),
	netip.MustParsePrefix("2606:4700::/32"),
	netip.MustParsePrefix("2803:f800::/32"),
	netip.MustParsePrefix("2a06:98c0::/29"),
	netip.MustParsePrefix("2c0f:f248::/32"),
}

func isCloudflareIP(ip netip.Addr) bool {
	for _, prefix := range cloudflarePrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func parseRequestIP(value string) (netip.Addr, bool) {
	ip, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.Unmap(), true
}

func forwardedClientIP(value string) (netip.Addr, bool) {
	for _, candidate := range strings.Split(value, ",") {
		ip, ok := parseRequestIP(candidate)
		if !ok {
			continue
		}
		return ip, true
	}
	return netip.Addr{}, false
}

func (a *App) clientIP(c *gin.Context) string {
	// The service is deployed behind reverse proxies. The first valid address
	// in X-Forwarded-For is the original visitor, regardless of proxy count.
	if forwardedIP, ok := forwardedClientIP(c.GetHeader("X-Forwarded-For")); ok {
		if isCloudflareIP(forwardedIP) {
			if visitorIP, valid := parseRequestIP(c.GetHeader("CF-Connecting-IP")); valid {
				return visitorIP.String()
			}
		}
		return forwardedIP.String()
	}
	if realIP, ok := parseRequestIP(c.GetHeader("X-Real-IP")); ok {
		if isCloudflareIP(realIP) {
			if visitorIP, valid := parseRequestIP(c.GetHeader("CF-Connecting-IP")); valid {
				return visitorIP.String()
			}
		}
		return realIP.String()
	}
	remoteIP := strings.TrimSpace(c.ClientIP())
	if proxyIP, ok := parseRequestIP(remoteIP); ok && isCloudflareIP(proxyIP) {
		if visitorIP, valid := parseRequestIP(c.GetHeader("CF-Connecting-IP")); valid {
			return visitorIP.String()
		}
	}
	return remoteIP
}

func (a *App) recordChatIP(ctx context.Context, uid int64, ip string) (int64, error) {
	connectedAt := time.Now().Unix()
	if uid <= 0 || ip == "" {
		return connectedAt, nil
	}
	_, err := a.db.ExecContext(ctx, "UPDATE "+a.t("user")+" SET last_chat_time=?,last_chat_ip=? WHERE user_id=?", connectedAt, ip, uid)
	return connectedAt, err
}
