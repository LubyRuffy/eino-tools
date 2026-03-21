package cloudflare

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var titleMarkers = []string{
	"just a moment",
	"checking your browser",
	"attention required",
}

var urlMarkers = []string{
	"/cdn-cgi/challenge-platform/",
	"__cf_chl_",
}

var bodyMarkers = []string{
	"challenge-running",
	"cf_chl_",
	"cf browser verification",
	"cf-mitigated",
	"enable javascript and cookies to continue",
	"checking if the site connection is secure",
	"__cf_chl_opt",
}

var httpURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

type ChallengeError struct {
	URL    string
	Reason string
}

func (e *ChallengeError) Error() string {
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		reason = "cloudflare challenge detected"
	}
	urlText := strings.TrimSpace(e.URL)
	if urlText == "" {
		return reason
	}
	return fmt.Sprintf("%s: %s", reason, urlText)
}

func IsChallengeError(err error) bool {
	if err == nil {
		return false
	}
	var target *ChallengeError
	return errors.As(err, &target)
}

func DetectFromPageContent(title, pageURL, html string) bool {
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))
	normalizedURL := strings.ToLower(strings.TrimSpace(pageURL))
	normalizedHTML := strings.ToLower(strings.TrimSpace(html))
	joined := strings.Join([]string{normalizedTitle, normalizedURL, normalizedHTML}, "\n")
	if joined == "" {
		return false
	}

	for _, marker := range titleMarkers {
		if strings.Contains(normalizedTitle, marker) {
			return true
		}
	}
	for _, marker := range urlMarkers {
		if strings.Contains(normalizedURL, marker) {
			return true
		}
	}
	for _, marker := range bodyMarkers {
		if strings.Contains(normalizedHTML, marker) || strings.Contains(normalizedURL, marker) {
			return true
		}
	}
	return false
}

func IsHTTPBlock(statusCode int, headers http.Header) bool {
	if headers == nil {
		headers = http.Header{}
	}
	server := strings.ToLower(strings.TrimSpace(headers.Get("Server")))
	cfRay := strings.TrimSpace(headers.Get("CF-Ray"))
	cfMitigated := strings.ToLower(strings.TrimSpace(headers.Get("CF-Mitigated")))
	cloudflareHint := strings.Contains(server, "cloudflare") || cfRay != "" || cfMitigated != ""
	if !cloudflareHint {
		return false
	}

	switch statusCode {
	case http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	}

	if cfMitigated == "challenge" || cfMitigated == "managed_challenge" {
		return true
	}

	for _, value := range headers.Values("Set-Cookie") {
		if strings.Contains(strings.ToLower(value), "cf_clearance=") {
			return true
		}
	}
	return false
}

func ExtractFirstURL(input string) string {
	matched := httpURLPattern.FindString(strings.TrimSpace(input))
	if matched == "" {
		return ""
	}
	return strings.TrimRight(matched, ".,;:!?)]}\"'")
}

func DetectFromCommandOutput(command, stdout, stderr string) (string, bool) {
	combined := strings.Join([]string{
		strings.TrimSpace(command),
		strings.TrimSpace(stdout),
		strings.TrimSpace(stderr),
	}, "\n")
	if !IsLikelyChallengeText(combined) {
		return "", false
	}
	for _, candidate := range []string{stdout, stderr, command} {
		if urlValue := ExtractFirstURL(candidate); urlValue != "" {
			return urlValue, true
		}
	}
	return "", true
}

func ExtractHTTPDomainsFromText(input string) []string {
	matches := httpURLPattern.FindAllString(strings.TrimSpace(input), -1)
	if len(matches) == 0 {
		return nil
	}
	output := make([]string, 0, len(matches))
	seen := make(map[string]struct{})
	for _, rawURL := range matches {
		domain := ExtractDomainFromURL(rawURL)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		output = append(output, domain)
	}
	return output
}

func NormalizeDomain(domain string) string {
	trimmed := strings.TrimSpace(strings.ToLower(domain))
	trimmed = strings.TrimPrefix(trimmed, ".")
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			trimmed = strings.ToLower(strings.TrimSpace(parsed.Hostname()))
		}
	}
	if idx := strings.Index(trimmed, ":"); idx > 0 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "."))
}

func ExtractDomainFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return NormalizeDomain(parsed.Hostname())
}

func IsLikelyChallengeText(input string) bool {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return false
	}
	for _, marker := range titleMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	for _, marker := range bodyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	for _, marker := range urlMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
