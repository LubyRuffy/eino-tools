package webfetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/LubyRuffy/eino-tools/internal/cloudflare"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	values map[string]string
}

type fakeCookieProvider struct {
	cookies []RequestCookie
}

func (f fakeCookieProvider) GetCookies(domain string) []RequestCookie {
	return append([]RequestCookie{}, f.cookies...)
}

func (f *fakeCache) Get(key string) (string, bool, error) {
	if f.values == nil {
		return "", false, nil
	}
	value, ok := f.values[key]
	return value, ok, nil
}

func (f *fakeCache) Set(key string, content string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = content
	return nil
}

func TestTool_InfoName(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ToolName, info.Name)
}

func TestTool_Fetch_EmptyURL(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	_, fetchErr := tl.Fetch(context.Background(), "", false)
	require.Error(t, fetchErr)
	assert.Contains(t, fetchErr.Error(), "URL is required")
}

func TestTool_Fetch_UsesBrowserFetchForProtectedDomain(t *testing.T) {
	protectedDomains := cloudflare.NewProtectedDomains(0)
	protectedDomains.Mark("https://dogster.com")

	called := false
	tl, err := New(Config{
		ProtectedDomains: protectedDomains,
		BrowserFetch: func(ctx context.Context, rawURL string, render bool) (string, error) {
			called = true
			assert.Equal(t, "https://dogster.com", rawURL)
			assert.True(t, render)
			return "browser fetched content", nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), "https://dogster.com", true)
	require.NoError(t, fetchErr)
	assert.True(t, called)
	assert.Equal(t, "browser fetched content", result)
}

func TestTool_Fetch_CloudflareChallengeCallsBrowserFetchFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "cloudflare")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html><head><title>Just a moment...</title></head><body>challenge-running</body></html>"))
	}))
	defer server.Close()

	called := false
	tl, err := New(Config{
		ProtectedDomains: cloudflare.NewProtectedDomains(0),
		BrowserFetch: func(ctx context.Context, rawURL string, render bool) (string, error) {
			called = true
			assert.Equal(t, server.URL, rawURL)
			assert.False(t, render)
			return "browser fallback", nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), server.URL, false)
	require.NoError(t, fetchErr)
	assert.True(t, called)
	assert.Equal(t, "browser fallback", result)
}

func TestTool_Fetch_UsesInjectedHTMLFetcher(t *testing.T) {
	tl, err := New(Config{
		HTMLFetcher: func(ctx context.Context, rawURL string) (string, error) {
			assert.Equal(t, "https://example.com", rawURL)
			return "<html><body><h1>Injected</h1><p>custom html</p></body></html>", nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), "https://example.com", false)
	require.NoError(t, fetchErr)
	assert.Contains(t, result, "Injected")
	assert.Contains(t, result, "custom html")
}

func TestTool_Fetch_RenderTrueUsesInjectedRenderFetcher(t *testing.T) {
	called := false
	tl, err := New(Config{
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			called = true
			assert.Equal(t, "https://example.com/rendered", rawURL)
			return "# Injected Render\n\ncustom markdown", nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), "https://example.com/rendered", true)
	require.NoError(t, fetchErr)
	assert.True(t, called)
	assert.Equal(t, "# Injected Render\n\ncustom markdown", result)
}

func TestTool_Fetch_AppliesDefaultHeadersAndCookies(t *testing.T) {
	var receivedUserAgent string
	var receivedAccept string
	var receivedAcceptLanguage string
	var receivedUpgrade string
	var receivedCookie string
	var receivedCustom string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		receivedAccept = r.Header.Get("Accept")
		receivedAcceptLanguage = r.Header.Get("Accept-Language")
		receivedUpgrade = r.Header.Get("Upgrade-Insecure-Requests")
		receivedCookie = r.Header.Get("Cookie")
		receivedCustom = r.Header.Get("X-Test-Header")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h1>Injected headers</h1><p>ok</p></body></html>"))
	}))
	defer server.Close()

	tl, err := New(Config{
		HeaderProvider: func(rawURL string) http.Header {
			header := http.Header{}
			header.Set("X-Test-Header", "hello")
			return header
		},
		CookieProvider: fakeCookieProvider{
			cookies: []RequestCookie{
				{Name: "cf_clearance", Value: "clear-token", Path: "/"},
				{Name: "session", Value: "abc123"},
			},
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), server.URL, false)
	require.NoError(t, fetchErr)
	assert.Contains(t, result, "Injected headers")
	assert.Contains(t, receivedUserAgent, "Mozilla/5.0")
	assert.Contains(t, receivedAccept, "text/html")
	assert.NotEmpty(t, receivedAcceptLanguage)
	assert.Equal(t, "1", receivedUpgrade)
	assert.Equal(t, "hello", receivedCustom)
	assert.Contains(t, receivedCookie, "cf_clearance=clear-token")
	assert.Contains(t, receivedCookie, "session=abc123")
}

func TestTool_Fetch_UsesInjectedChallengeDetector(t *testing.T) {
	detectedErr := errors.New("custom challenge")

	called := false
	tl, err := New(Config{
		HTMLFetcher: func(ctx context.Context, rawURL string) (string, error) {
			return "", detectedErr
		},
		ChallengeDetector: func(err error) (string, bool) {
			if errors.Is(err, detectedErr) {
				return "https://example.com/challenge", true
			}
			return "", false
		},
		ProtectedDomains: cloudflare.NewProtectedDomains(0),
		BrowserFetch: func(ctx context.Context, rawURL string, render bool) (string, error) {
			called = true
			return "browser fallback", nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), "https://example.com", false)
	require.NoError(t, fetchErr)
	assert.True(t, called)
	assert.Equal(t, "browser fallback", result)
}

func TestTool_Fetch_ChallengeHandlerAndRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNumber := calls.Add(1)
		if callNumber == 1 {
			w.Header().Set("Server", "cloudflare")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("<html><head><title>Just a moment...</title></head><body>challenge-running</body></html>"))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h1>Passed</h1><p>ok</p></body></html>"))
	}))
	defer server.Close()

	handlerCalls := 0
	tl, err := New(Config{
		ProtectedDomains: cloudflare.NewProtectedDomains(0),
		ChallengeHandler: func(ctx context.Context, req ChallengeRequest) error {
			handlerCalls++
			assert.Equal(t, ToolName, req.ToolName)
			assert.Equal(t, server.URL, req.URL)
			return nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), server.URL, false)
	require.NoError(t, fetchErr)
	assert.Contains(t, result, "Passed")
	assert.Equal(t, 1, handlerCalls)
	assert.EqualValues(t, 2, calls.Load())
}
