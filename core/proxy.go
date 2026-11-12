package core

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	log "github.com/cihub/seelog"
)

// InitProxy applies the optional proxy setting from cfg.yaml.
// Registry and DingTalk HTTP clients all use http.ProxyFromEnvironment,
// and Go's net/http natively dials socks5/socks5h proxies, so exporting
// HTTP(S)_PROXY before any request routes all traffic through the proxy.
// Must be called before the first HTTP request (env is cached on first use).
func InitProxy() error {
	if CONF == nil || len(strings.TrimSpace(CONF.Proxy)) == 0 {
		return nil
	}
	proxy := strings.TrimSpace(CONF.Proxy)
	if !strings.Contains(proxy, "://") {
		proxy = "socks5://" + proxy
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return errors.New(I18n.Sprintf("Invalid proxy %s: %v", CONF.Proxy, err))
	}
	switch u.Scheme {
	case "socks5", "socks5h", "http", "https":
	default:
		return errors.New(I18n.Sprintf("Invalid proxy %s: %v", CONF.Proxy,
			fmt.Sprintf("unsupported scheme %q, use socks5/socks5h/http/https", u.Scheme)))
	}
	if u.Host == "" {
		return errors.New(I18n.Sprintf("Invalid proxy %s: %v", CONF.Proxy, "missing host:port"))
	}
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if err := os.Setenv(key, proxy); err != nil {
			return err
		}
	}
	CONF.Proxy = proxy
	log.Info(I18n.Sprintf("Using proxy: %s", proxy))
	return nil
}