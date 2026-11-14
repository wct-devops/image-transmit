package core

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
)

// NormalizeProxy validates a proxy setting, defaulting the scheme to socks5.
func NormalizeProxy(raw string) (string, error) {
	proxy := strings.TrimSpace(raw)
	if !strings.Contains(proxy, "://") {
		proxy = "socks5://" + proxy
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return "", errors.New(I18n.Sprintf("Invalid proxy %s: %v", raw, err))
	}
	switch u.Scheme {
	case "socks5", "socks5h", "http", "https":
	default:
		return "", errors.New(I18n.Sprintf("Invalid proxy %s: %v", raw,
			fmt.Sprintf("unsupported scheme %q, use socks5/socks5h/http/https", u.Scheme)))
	}
	if u.Host == "" {
		return "", errors.New(I18n.Sprintf("Invalid proxy %s: %v", raw, "missing host:port"))
	}
	return proxy, nil
}

// InitRepoProxies validates the optional per-repository proxy settings.
// Repositories without a proxy connect directly; source and target often
// live in different network segments and need different proxies (or none).
func InitRepoProxies() error {
	for i := range CONF.SrcRepos {
		if err := initRepoProxy(&CONF.SrcRepos[i]); err != nil {
			return err
		}
	}
	for i := range CONF.DstRepos {
		if err := initRepoProxy(&CONF.DstRepos[i]); err != nil {
			return err
		}
	}
	return nil
}

func initRepoProxy(repo *Repo) error {
	if strings.TrimSpace(repo.Proxy) == "" {
		repo.Proxy = ""
		return nil
	}
	proxy, err := NormalizeProxy(repo.Proxy)
	if err != nil {
		return errors.New(I18n.Sprintf("Invalid proxy for repository %s: %v", repo.Registry, err))
	}
	repo.Proxy = proxy
	log.Info(I18n.Sprintf("Repository %s using proxy: %s", repo.Registry, proxy))
	return nil
}

// ProxyURL parses a normalized repository proxy setting; empty means direct.
func ProxyURL(proxy string) (*url.URL, error) {
	if proxy == "" {
		return nil, nil
	}
	return url.Parse(proxy)
}