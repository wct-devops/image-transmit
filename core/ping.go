package core

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/containers/image/v5/pkg/tlsclientconfig"
)

func PingRegistry(registry string, insecure bool, proxy string) error {
	registry = strings.TrimPrefix(strings.TrimPrefix(registry, "https://"), "http://")
	registry = strings.TrimSuffix(registry, "/")

	tr := tlsclientconfig.NewTransport()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	proxyURL, err := ProxyURL(proxy)
	if err != nil {
		return err
	}
	if proxyURL != nil {
		tr.Proxy = http.ProxyURL(proxyURL)
	} else {
		tr.Proxy = nil
	}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	resp, err := client.Get(fmt.Sprintf("https://%s/v2/", registry))
	if err != nil && insecure {
		resp, err = client.Get(fmt.Sprintf("http://%s/v2/", registry))
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("registry %s returned HTTP %d", registry, resp.StatusCode)
	}
	return nil
}