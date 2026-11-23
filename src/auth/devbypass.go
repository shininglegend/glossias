package auth

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// CheckDevBypass decides whether the server may start with devUser
// (DEV_USER) set. DEV_USER turns off Clerk: any request with the fixed
// dev_auth header acts as that user. That is only acceptable on a developer's
// machine, so it requires appURL (PUBLIC_APP_URL) to be set explicitly to a
// local address; an unset PUBLIC_APP_URL is refused because production may
// leave it unset. main exits when this returns an error.
func CheckDevBypass(devUser, appURL string) error {
	if devUser == "" {
		return nil
	}
	if appURL == "" {
		return fmt.Errorf("DEV_USER is set but PUBLIC_APP_URL is not; DEV_USER disables authentication and is only allowed with PUBLIC_APP_URL set to a localhost URL")
	}
	u, err := url.Parse(appURL)
	if err != nil || !isLocalHost(u.Hostname()) {
		return fmt.Errorf("DEV_USER is set but PUBLIC_APP_URL (%q) is not a localhost URL; DEV_USER disables authentication and must never be set outside local development", appURL)
	}
	return nil
}

func isLocalHost(host string) bool {
	host = strings.ToLower(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
