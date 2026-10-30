package webfetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LubyRuffy/eino-tools/internal/cloudflare"
	"github.com/LubyRuffy/eino-tools/netproxy"
	"github.com/go-rod/rod/lib/launcher/flags"
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

// D1/D2: a recognized HTTP bot wall upgrades the existing render path once.
func TestTool_Fetch_DataDomeUpgradesToRender(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Server", "DataDome")
				w.Header().Set("X-DataDome", "protected")
				w.Header().Set("X-DataDome-Riskscore", "0.999")
				w.WriteHeader(status)
				_, _ = w.Write([]byte("<html><body>Request blocked</body></html>"))
			}))
			defer server.Close()
			calls := 0
			const markdown = "# Rendered article\n\nUseful content."
			tl, err := New(Config{
				RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
					calls++
					assert.Equal(t, server.URL, rawURL)
					return markdown, nil
				},
			})
			require.NoError(t, err)
			result, err := tl.Fetch(context.Background(), server.URL, false)
			require.NoError(t, err)
			assert.Equal(t, markdown, result)
			assert.Equal(t, 1, calls)
		})
	}
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
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			t.Error("Cloudflare fallback must keep using BrowserFetch")
			return "", errors.New("unexpected render")
		},
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

func TestTool_Fetch_NonRenderDoesNotReturnRenderCache(t *testing.T) {
	cache := &fakeCache{values: map[string]string{}}
	tl, err := New(Config{
		HTMLFetcher: func(ctx context.Context, rawURL string) (string, error) {
			return "plain html body", nil
		},
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			return "rendered markdown", nil
		},
		Cache: cache,
	})
	require.NoError(t, err)

	rendered, err := tl.Fetch(context.Background(), "https://example.com/page", true)
	require.NoError(t, err)
	assert.Equal(t, "rendered markdown", rendered)

	plain, err := tl.Fetch(context.Background(), "https://example.com/page", false)
	require.NoError(t, err)
	assert.Equal(t, "plain html body", plain, "non-render fetch must not reuse render-mode cache")
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

func TestNewRodLauncher_AppliesBrowserBin(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "Brave Browser")
	require.NoError(t, os.WriteFile(bin, []byte("x"), 0o755))

	launch, err := newRodLauncher(context.Background(), true, netproxy.Config{}, bin)
	require.NoError(t, err)
	require.Equal(t, bin, launch.Get(flags.Bin))
}

func TestNewRodLauncher_InvalidBrowserBin_ReturnsError(t *testing.T) {
	_, err := newRodLauncher(context.Background(), true, netproxy.Config{}, filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestNewRodLauncher_AppliesProxyConfig(t *testing.T) {
	launch, err := newRodLauncher(context.Background(), true, netproxy.Config{
		HTTPProxy:  "http://proxy-http:8080",
		HTTPSProxy: "http://proxy-https:8443",
		NoProxy:    "localhost,.svc",
	}, "")
	require.NoError(t, err)
	require.Equal(t, "http=http://proxy-http:8080;https=http://proxy-https:8443", launch.Get(flags.ProxyServer))

	bypassValues, ok := launch.GetFlags(flags.Flag("proxy-bypass-list"))
	require.True(t, ok)
	require.Equal(t, []string{"localhost;*.svc;svc"}, bypassValues)
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

// D3: either the server header or any DataDome response header is sufficient.
func TestTool_Fetch_DataDomeResponseSignals(t *testing.T) {
	for _, header := range []struct{ name, value string }{
		{"Server", "edge DATADOME"},
		{"X-DataDome", "protected"},
		{"X-DataDome-Riskscore", "0.999"},
	} {
		t.Run(header.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(header.name, header.value)
				w.WriteHeader(http.StatusUnauthorized)
			}))
			defer server.Close()
			calls := 0
			tl, err := New(Config{RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
				calls++
				return "rendered article", nil
			}})
			require.NoError(t, err)
			result, err := tl.Fetch(context.Background(), server.URL, false)
			require.NoError(t, err)
			assert.Equal(t, "rendered article", result)
			assert.Equal(t, 1, calls)
		})
	}
}

// D4: ordinary authorization errors must not launch a browser.
func TestTool_Fetch_PlainHTTPRejectionsDoNotRender(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			calls := 0
			tl, err := New(Config{RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
				calls++
				return "unexpected render", nil
			}})
			require.NoError(t, err)
			result, err := tl.Fetch(context.Background(), server.URL, false)
			require.Error(t, err)
			assert.Empty(t, result)
			assert.Contains(t, err.Error(), fmt.Sprintf("HTTP error: %d", status))
			assert.Zero(t, calls)
		})
	}
}

// D5: failed rendering cannot fall through into Cloudflare handoffs or retry.
func TestTool_Fetch_DataDomeRenderFailureIsTerminal(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("X-DataDome", "protected")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	renderErr := errors.New("browser unavailable")
	renderCalls, browserCalls, handlerCalls, detectorCalls := 0, 0, 0, 0
	cache := &fakeCache{}
	tl, err := New(Config{
		Cache: cache,
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			renderCalls++
			return "partial output must be discarded", renderErr
		},
		BrowserFetch: func(ctx context.Context, rawURL string, render bool) (string, error) {
			browserCalls++
			return "unexpected fallback", nil
		},
		ChallengeHandler: func(ctx context.Context, req ChallengeRequest) error {
			handlerCalls++
			return nil
		},
		ChallengeDetector: func(err error) (string, bool) {
			detectorCalls++
			return server.URL, true
		},
	})
	require.NoError(t, err)
	result, err := tl.Fetch(context.Background(), server.URL, false)
	require.Error(t, err)
	assert.Empty(t, result)
	assert.ErrorIs(t, err, renderErr)
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{"http client", "bot wall", "browser rendering", "attempted once"} {
		assert.Contains(t, message, fragment)
	}
	assert.NotContains(t, message, "cloudflare")
	assert.Equal(t, 1, renderCalls)
	assert.EqualValues(t, 1, requests.Load())
	assert.Zero(t, browserCalls)
	assert.Zero(t, handlerCalls)
	assert.Zero(t, detectorCalls)
	assert.Empty(t, cache.values)
}

// D6: an explicit render request does no HTTP fetch and no second render.
func TestTool_Fetch_RenderFailureDoesNotRetry(t *testing.T) {
	renderErr := errors.New("render failed")
	httpCalls, renderCalls := 0, 0
	tl, err := New(Config{
		HTMLFetcher: func(ctx context.Context, rawURL string) (string, error) {
			httpCalls++
			return "unexpected HTTP fetch", nil
		},
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			renderCalls++
			return "", renderErr
		},
	})
	require.NoError(t, err)
	result, err := tl.Fetch(context.Background(), "https://example.com/article", true)
	assert.ErrorIs(t, err, renderErr)
	assert.Empty(t, result)
	assert.Zero(t, httpCalls)
	assert.Equal(t, 1, renderCalls)
}

// D7/D10: HTML and Markdown challenge resources cannot become cached articles.
func TestTool_Fetch_RejectsRenderedDataDomeChallenge(t *testing.T) {
	for _, challenge := range []string{
		`<html><body><iframe src="https://geo.captcha-delivery.com/captcha/?id=test"></iframe></body></html>`,
		`[Verify your browser](https://geo.captcha-delivery.com/captcha/?id=test)`,
		`<script src="https://geo.captcha-delivery.com/interstitial/?id=test"></script>`,
		`<script src="https://ct.captcha-delivery.com/c.js"></script>`,
		`<script src="https://ct.captcha-delivery.com/i.js"></script>`,
	} {
		for _, render := range []bool{false, true} {
			t.Run(fmt.Sprintf("render=%t/%s", render, challenge), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-DataDome", "protected")
					w.WriteHeader(http.StatusForbidden)
				}))
				defer server.Close()
				cache := &fakeCache{}
				calls := 0
				tl, err := New(Config{
					Cache: cache,
					RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
						calls++
						return challenge, nil
					},
				})
				require.NoError(t, err)
				result, err := tl.Fetch(context.Background(), server.URL, render)
				require.Error(t, err)
				assert.Empty(t, result)
				assert.Equal(t, 1, calls)
				assert.Empty(t, cache.values)
				assert.NotContains(t, err.Error(), "captcha-delivery.com")
			})
		}
	}
}

func TestTool_Fetch_RenderedArticleMayMentionDataDome(t *testing.T) {
	for _, article := range []string{
		"# Bot protection\n\nDataDome is a vendor. Its resources use captcha-delivery.com.",
		`<article>Article content</article><script src="https://js.datadome.co/tags.js"></script>`,
		"Read more at https://geo.captcha-delivery.com/documentation/ for background.",
		"The challenge loader is documented at https://ct.captcha-delivery.com/c.js.",
		`[Challenge script reference](https://ct.captcha-delivery.com/c.js)`,
	} {
		t.Run(article, func(t *testing.T) {
			tl, err := New(Config{RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
				return article, nil
			}})
			require.NoError(t, err)
			result, err := tl.Fetch(context.Background(), "https://example.com/article", true)
			require.NoError(t, err)
			assert.Equal(t, article, result)
		})
	}
}

func TestTool_Fetch_DataDomeSignalBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		header string
	}{
		{"successful protected article", http.StatusOK, "X-DataDome"},
		{"rate limit", http.StatusTooManyRequests, "X-DataDome"},
		{"similar header name", http.StatusUnauthorized, "X-DataDomeRisk"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(test.header, "protected")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte("<html><body><h1>Article</h1><p>Ordinary readable content.</p></body></html>"))
			}))
			defer server.Close()
			calls := 0
			tl, err := New(Config{RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
				calls++
				return "unexpected render", nil
			}})
			require.NoError(t, err)
			result, err := tl.Fetch(context.Background(), server.URL, false)
			if test.status == http.StatusOK {
				require.NoError(t, err)
				assert.Contains(t, result, "Ordinary readable content")
			} else {
				require.Error(t, err)
				assert.Empty(t, result)
				assert.Contains(t, err.Error(), fmt.Sprintf("HTTP error: %d", test.status))
			}
			assert.Zero(t, calls)
		})
	}
}

// D9: both extraction failure and extracted JS-shell text retain render retry.
func TestTool_Fetch_JSShellStillRetriesWithRender(t *testing.T) {
	for _, html := range []string{
		`<html><body><div id="app"></div></body></html>`,
		`<html><body><p>Please enable javascript to view this page.</p></body></html>`,
	} {
		t.Run(html, func(t *testing.T) {
			calls := 0
			tl, err := New(Config{
				HTMLFetcher: func(ctx context.Context, rawURL string) (string, error) { return html, nil },
				RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
					calls++
					return "rendered application article", nil
				},
			})
			require.NoError(t, err)
			result, err := tl.Fetch(context.Background(), "https://example.com/article", false)
			require.NoError(t, err)
			assert.Equal(t, "rendered application article", result)
			assert.Equal(t, 1, calls)
		})
	}
}

// D10: successful upgrades cache by requested mode, preserving mode isolation.
func TestTool_Fetch_DataDomeUpgradeCanBeCached(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("X-DataDome", "protected")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	cache := &fakeCache{}
	calls := 0
	tl, err := New(Config{
		Cache: cache,
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			calls++
			return fmt.Sprintf("rendered article %d", calls), nil
		},
	})
	require.NoError(t, err)
	for range 2 {
		result, err := tl.Fetch(context.Background(), server.URL, false)
		require.NoError(t, err)
		assert.Equal(t, "rendered article 1", result)
	}
	assert.Equal(t, 1, calls)
	assert.EqualValues(t, 1, requests.Load())
	rendered, err := tl.Fetch(context.Background(), server.URL, true)
	require.NoError(t, err)
	assert.Equal(t, "rendered article 2", rendered)
	assert.Equal(t, 2, calls)
	assert.Equal(t, "rendered article 1", cache.values[server.URL])
	assert.Equal(t, "rendered article 2", cache.values[server.URL+"::render"])
}

// D11: callers learn the generic HTTP-first and single-upgrade contract.
func TestTool_InfoDescribesAutomaticRenderUpgrade(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	params, err := info.ParamsOneOf.ToJSONSchema()
	require.NoError(t, err)
	encoded, err := json.Marshal(params)
	require.NoError(t, err)
	var schema struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	require.NoError(t, json.Unmarshal(encoded, &schema))
	for _, desc := range []string{info.Desc, schema.Properties["render"].Description} {
		lower := strings.ToLower(desc)
		for _, fragment := range []string{"http", "bot wall", "once", "direct"} {
			assert.Contains(t, lower, fragment)
		}
		for _, fixture := range []string{"captcha-delivery.com", "0.999", "Rendered article", "Request blocked"} {
			assert.NotContains(t, desc, fixture)
		}
	}
	assert.Contains(t, info.Desc, "render=true")
}
