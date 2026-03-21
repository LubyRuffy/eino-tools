package fetchurl

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFetchCache struct {
	values map[string]string
}

func (f *fakeFetchCache) Get(key string) (string, bool, error) {
	if f.values == nil {
		return "", false, nil
	}
	value, ok := f.values[key]
	return value, ok, nil
}

func (f *fakeFetchCache) Set(key string, content string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[key] = content
	return nil
}

func TestTool_Fetch_EmptyURL(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	_, fetchErr := tl.Fetch(context.Background(), "", false)
	require.Error(t, fetchErr)
	assert.Contains(t, fetchErr.Error(), "URL is required")
}

func TestTool_Fetch_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tl, err := New(Config{})
	require.NoError(t, err)

	_, fetchErr := tl.Fetch(context.Background(), server.URL, false)
	require.Error(t, fetchErr)
	assert.Contains(t, fetchErr.Error(), "HTTP error")
}

func TestTool_Fetch_ExtractsReadableText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>Hello</h1><p>Readable body.</p><script>ignored()</script></body></html>`))
	}))
	defer server.Close()

	tl, err := New(Config{})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), server.URL, false)
	require.NoError(t, fetchErr)
	assert.Contains(t, result, "Hello")
	assert.Contains(t, result, "Readable body.")
	assert.NotContains(t, result, "ignored")
}

func TestTool_Fetch_RenderFallbackUsesInjectedRenderer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><div id="app"></div></body></html>`))
	}))
	defer server.Close()

	tl, err := New(Config{
		RenderFetcher: func(ctx context.Context, rawURL string) (string, error) {
			return "# Rendered\n\nfrom renderer", nil
		},
	})
	require.NoError(t, err)

	result, fetchErr := tl.Fetch(context.Background(), server.URL, false)
	require.NoError(t, fetchErr)
	assert.Equal(t, "# Rendered\n\nfrom renderer", result)
}

func TestTool_Fetch_CloudflareChallengeCallsHandlerAndRetries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNumber := calls.Add(1)
		if callNumber == 1 {
			w.Header().Set("Server", "cloudflare")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<html><head><title>Just a moment...</title></head><body>challenge-running</body></html>`))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>Passed</h1><p>ok</p></body></html>`))
	}))
	defer server.Close()

	handlerCalls := 0
	tl, err := New(Config{
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
