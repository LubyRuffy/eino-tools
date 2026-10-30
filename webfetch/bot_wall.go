package webfetch

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// botWallError is separate from Cloudflare's manual-verification workflow.
type botWallError struct {
	statusCode int
}

func (e *botWallError) Error() string {
	if e.statusCode != 0 {
		return fmt.Sprintf("HTTP client blocked by bot wall (status %d)", e.statusCode)
	}
	return "browser rendering returned a bot wall challenge page"
}

func isHTTPBotWall(statusCode int, headers http.Header) bool {
	if statusCode != http.StatusUnauthorized && statusCode != http.StatusForbidden {
		return false
	}
	for name, values := range headers {
		name = strings.ToLower(name)
		if name == "x-datadome" || strings.HasPrefix(name, "x-datadome-") {
			return true
		}
		if name == "server" {
			for _, value := range values {
				if strings.Contains(strings.ToLower(value), "datadome") {
					return true
				}
			}
		}
	}
	return false
}

// Match challenge endpoint URLs, rather than a mention of the vendor or its
// ordinary client-side tag. This also recognizes challenge links in Markdown.
var botWallChallengeResource = regexp.MustCompile(`(?i)(?:https?:)?//(?:[a-z0-9-]+\.)*captcha-delivery\.com/(?:captcha|interstitial)(?:[/?#]|$)`)

// A challenge bootloader is meaningful when loaded as a script, not when an
// article simply cites its URL.
var botWallChallengeScript = regexp.MustCompile(`(?i)<script\b[^>]*\s+src\s*=\s*["'](?:https?:)?//(?:[a-z0-9-]+\.)*captcha-delivery\.com/[ci]\.js(?:[?#][^"']*)?["']`)

func isBotWallPage(content string) bool {
	return botWallChallengeResource.MatchString(content) || botWallChallengeScript.MatchString(content)
}
