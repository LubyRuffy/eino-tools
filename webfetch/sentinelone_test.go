package webfetch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetch_SentinelOneCVE tests fetching a real SentinelOne CVE page with browser rendering
func TestFetch_SentinelOneCVE(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a browser fetcher using the same Rod approach as the render fetcher
	browserFetch := func(ctx context.Context, rawURL string, render bool) (string, error) {
		browser, launch, err := launchRodBrowser(ctx, true, Config{}.ProxyConfig, "")
		if err != nil {
			return "", err
		}
		defer browser.Close()
		defer launch.Kill()

		page := browser.MustPage().Context(ctx)
		defer page.Close()
		page = page.Timeout(60 * time.Second)

		if err := page.Navigate(rawURL); err != nil {
			return "", err
		}
		if err := page.WaitLoad(); err != nil {
			return "", err
		}

		// Wait for Cloudflare challenge to complete
		// Look for either CVE content or wait up to 30 seconds
		maxRetries := 30
		for i := 0; i < maxRetries; i++ {
			time.Sleep(1 * time.Second)
			html, _ := page.HTML()
			lower := strings.ToLower(html)

			// Check if we're past the challenge
			if !strings.Contains(lower, "security verification") &&
				!strings.Contains(lower, "just a moment") &&
				(strings.Contains(lower, "cve") || strings.Contains(lower, "vulnerability")) {
				break
			}
		}

		// Get the HTML content
		html, err := page.HTML()
		if err != nil {
			return "", err
		}

		// Extract readable text using go-readability
		return ExtractReadableText(html, rawURL)
	}

	tl, err := New(Config{
		BrowserFetch: browserFetch,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Test with render=true to use browser rendering
	result, fetchErr := tl.Fetch(ctx, "https://www.sentinelone.com/vulnerability-database/cve-2026-50522/", true)
	require.NoError(t, fetchErr, "fetch should succeed with browser rendering")

	// Verify we got actual content
	assert.NotEmpty(t, result, "result should not be empty")

	// Check for expected content patterns in a CVE page
	lower := strings.ToLower(result)
	if strings.Contains(lower, "security verification") || strings.Contains(lower, "just a moment") {
		t.Skip("site still behind bot-wall challenge; skipping environment-dependent assertions")
	}

	// Should contain CVE identifier
	assert.True(t, strings.Contains(lower, "cve-2026-50522") || strings.Contains(lower, "cve"),
		"should contain CVE reference")

	// Should contain vulnerability-related content
	hasVulnContent := strings.Contains(lower, "vulnerability") ||
		strings.Contains(lower, "description") ||
		strings.Contains(lower, "severity") ||
		strings.Contains(lower, "cvss") ||
		strings.Contains(lower, "affected")

	assert.True(t, hasVulnContent, "should contain vulnerability-related content")

	// Should be substantial content (not just empty page or error message)
	assert.Greater(t, len(result), 100, "should have substantial content (>100 chars)")

	// Log the first 500 characters for debugging
	if len(result) > 500 {
		t.Logf("First 500 chars of result:\n%s", result[:500])
	} else {
		t.Logf("Full result:\n%s", result)
	}
}
