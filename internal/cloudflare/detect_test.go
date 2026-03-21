package cloudflare

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFromPageContent(t *testing.T) {
	assert.True(t, DetectFromPageContent(
		"Just a moment...",
		"https://example.com/cdn-cgi/challenge-platform/h/g",
		"<html><body><div id=\"challenge-running\"></div></body></html>",
	))
	assert.False(t, DetectFromPageContent(
		"Example Domain",
		"https://example.com/",
		"<html><body><h1>Example Domain</h1></body></html>",
	))
}

func TestIsHTTPBlock(t *testing.T) {
	headers := http.Header{}
	headers.Set("Server", "cloudflare")
	headers.Set("CF-Ray", "123")
	assert.True(t, IsHTTPBlock(http.StatusForbidden, headers))
	assert.False(t, IsHTTPBlock(http.StatusOK, headers))
}

func TestDetectFromCommandOutput(t *testing.T) {
	urlValue, detected := DetectFromCommandOutput(
		`curl -sS https://www.dogster.com/`,
		"",
		"Just a moment... https://www.dogster.com/cdn-cgi/challenge-platform",
	)
	assert.True(t, detected)
	assert.Equal(t, "https://www.dogster.com/cdn-cgi/challenge-platform", urlValue)
}
