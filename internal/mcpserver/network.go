package mcpserver

import (
	"net/http"

	"github.com/LubyRuffy/eino-tools/netproxy"
)

type proxyConfig = netproxy.Config

func resolveProxyConfig(cfg Config, getenv func(string) string) proxyConfig {
	return netproxy.Resolve(netproxy.Config{
		HTTPProxy:  cfg.HTTPProxy,
		HTTPSProxy: cfg.HTTPSProxy,
		NoProxy:    cfg.NoProxy,
	}, getenv)
}

func newNetworkHTTPClient(cfg Config, getenv func(string) string) (*http.Client, error) {
	return netproxy.NewHTTPClient(resolveProxyConfig(cfg, getenv))
}
