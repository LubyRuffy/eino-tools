package mcpserver

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewToolset_UsesCanonicalToolNamesOnly(t *testing.T) {
	tools, err := NewToolset(context.Background(), Config{
		BaseDir: t.TempDir(),
		Name:    "eino-tools",
		Version: "dev",
	})
	require.NoError(t, err)

	names, err := ToolNames(context.Background(), tools)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"web_search",
		"web_fetch",
		"exec",
		"read",
		"write",
		"edit",
		"ls",
		"tree",
		"glob",
		"grep",
		"python_runner",
		"screenshot",
	}, names)
	require.NotContains(t, names, "fetchurl")
	require.NotContains(t, names, "bashcmd")
}

func TestProxyConfig_UsesCLIFlagsBeforeEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://env-http:8080")
	t.Setenv("HTTPS_PROXY", "http://env-https:8443")
	t.Setenv("NO_PROXY", "env.local")

	proxyCfg := resolveProxyConfig(Config{
		HTTPProxy:  "http://cli-http:8080",
		HTTPSProxy: "http://cli-https:8443",
		NoProxy:    "cli.local",
	}, getenvMap(t))

	require.Equal(t, "http://cli-http:8080", proxyCfg.HTTPProxy)
	require.Equal(t, "http://cli-https:8443", proxyCfg.HTTPSProxy)
	require.Equal(t, "cli.local", proxyCfg.NoProxy)
}

func TestNewNetworkHTTPClient_FallsBackToEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://env-http:8080")
	t.Setenv("HTTPS_PROXY", "http://env-https:8443")
	t.Setenv("NO_PROXY", "localhost,.svc")

	client, err := newNetworkHTTPClient(Config{}, getenvMap(t))
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.Transport)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.Proxy)

	httpProxyURL, err := transport.Proxy(&http.Request{URL: mustURL(t, "http://example.com")})
	require.NoError(t, err)
	require.Equal(t, "http://env-http:8080", httpProxyURL.String())

	httpsProxyURL, err := transport.Proxy(&http.Request{URL: mustURL(t, "https://example.com")})
	require.NoError(t, err)
	require.Equal(t, "http://env-https:8443", httpsProxyURL.String())

	noProxyURL, err := transport.Proxy(&http.Request{URL: mustURL(t, "http://localhost")})
	require.NoError(t, err)
	require.Nil(t, noProxyURL)
}

func getenvMap(t *testing.T) func(string) string {
	t.Helper()
	return func(key string) string {
		return os.Getenv(key)
	}
}

func mustURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	require.NoError(t, err)
	return parsed
}
