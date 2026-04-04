package netproxy

import (
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolve_UsesExplicitConfigBeforeEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://env-http:8080")
	t.Setenv("HTTPS_PROXY", "http://env-https:8443")
	t.Setenv("NO_PROXY", "env.local")

	cfg := Resolve(Config{
		HTTPProxy:  "http://cli-http:8080",
		HTTPSProxy: "http://cli-https:8443",
		NoProxy:    "cli.local",
	}, os.Getenv)

	require.Equal(t, "http://cli-http:8080", cfg.HTTPProxy)
	require.Equal(t, "http://cli-https:8443", cfg.HTTPSProxy)
	require.Equal(t, "cli.local", cfg.NoProxy)
}

func TestNewHTTPClient_UsesProxyConfig(t *testing.T) {
	client, err := NewHTTPClient(Config{
		HTTPProxy:  "http://proxy-http:8080",
		HTTPSProxy: "http://proxy-https:8443",
		NoProxy:    "localhost,.svc",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.Transport)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.Proxy)

	httpProxyURL, err := transport.Proxy(&http.Request{URL: mustURL(t, "http://example.com")})
	require.NoError(t, err)
	require.Equal(t, "http://proxy-http:8080", httpProxyURL.String())

	httpsProxyURL, err := transport.Proxy(&http.Request{URL: mustURL(t, "https://example.com")})
	require.NoError(t, err)
	require.Equal(t, "http://proxy-https:8443", httpsProxyURL.String())

	noProxyURL, err := transport.Proxy(&http.Request{URL: mustURL(t, "http://localhost")})
	require.NoError(t, err)
	require.Nil(t, noProxyURL)
}

func TestChromiumConfig_MapsProxySettings(t *testing.T) {
	cfg, err := ChromiumConfig(Config{
		HTTPProxy:  "http://proxy-http:8080",
		HTTPSProxy: "http://proxy-https:8443",
		NoProxy:    "localhost,.svc",
	})
	require.NoError(t, err)
	require.Equal(t, "http=http://proxy-http:8080;https=http://proxy-https:8443", cfg.ProxyServer)
	require.Equal(t, "localhost;*.svc;svc", cfg.ProxyBypassList)
}

func mustURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed
}
