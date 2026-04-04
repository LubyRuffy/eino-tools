package netproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/http/httpproxy"
)

type Config struct {
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

type ChromiumProxyConfig struct {
	ProxyServer     string
	ProxyBypassList string
}

func Resolve(cfg Config, getenv func(string) string) Config {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	return Config{
		HTTPProxy:  firstNonEmpty(cfg.HTTPProxy, getenv("HTTP_PROXY")),
		HTTPSProxy: firstNonEmpty(cfg.HTTPSProxy, getenv("HTTPS_PROXY")),
		NoProxy:    firstNonEmpty(cfg.NoProxy, getenv("NO_PROXY")),
	}
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.HTTPProxy) != "" ||
		strings.TrimSpace(c.HTTPSProxy) != "" ||
		strings.TrimSpace(c.NoProxy) != ""
}

func NewHTTPClient(cfg Config) (*http.Client, error) {
	if !cfg.Enabled() {
		return nil, nil
	}

	proxyFunc, err := buildProxyFunc(cfg)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		if req == nil || req.URL == nil {
			return nil, nil
		}
		return proxyFunc(req.URL)
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func ChromiumConfig(cfg Config) (ChromiumProxyConfig, error) {
	if err := validateProxyURLs(cfg); err != nil {
		return ChromiumProxyConfig{}, err
	}

	proxyServer := buildChromiumProxyServer(cfg)
	if proxyServer == "" {
		return ChromiumProxyConfig{}, nil
	}

	return ChromiumProxyConfig{
		ProxyServer:     proxyServer,
		ProxyBypassList: buildChromiumBypassList(cfg.NoProxy),
	}, nil
}

func buildProxyFunc(cfg Config) (func(*url.URL) (*url.URL, error), error) {
	if err := validateProxyURLs(cfg); err != nil {
		return nil, err
	}

	return (&httpproxy.Config{
		HTTPProxy:  cfg.HTTPProxy,
		HTTPSProxy: cfg.HTTPSProxy,
		NoProxy:    cfg.NoProxy,
	}).ProxyFunc(), nil
}

func validateProxyURLs(cfg Config) error {
	for label, rawURL := range map[string]string{
		"HTTP proxy":  cfg.HTTPProxy,
		"HTTPS proxy": cfg.HTTPSProxy,
	} {
		if strings.TrimSpace(rawURL) == "" {
			continue
		}
		if _, err := url.Parse(rawURL); err != nil {
			return fmt.Errorf("invalid %s URL %q: %w", label, rawURL, err)
		}
	}
	return nil
}

func buildChromiumProxyServer(cfg Config) string {
	parts := make([]string, 0, 2)
	if httpProxy := strings.TrimSpace(cfg.HTTPProxy); httpProxy != "" {
		parts = append(parts, "http="+httpProxy)
	}
	if httpsProxy := strings.TrimSpace(cfg.HTTPSProxy); httpsProxy != "" {
		parts = append(parts, "https="+httpsProxy)
	}
	return strings.Join(parts, ";")
}

func buildChromiumBypassList(raw string) string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	seen := map[string]struct{}{}

	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if strings.HasPrefix(token, ".") {
			add("*" + token)
			add(strings.TrimPrefix(token, "."))
			continue
		}
		add(token)
	}
	return strings.Join(values, ";")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
