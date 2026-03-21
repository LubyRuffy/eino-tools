package cloudflare

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProtectedDomains_MarkAndContains(t *testing.T) {
	store := NewProtectedDomains(time.Minute)
	store.Mark("https://dogster.com/cdn-cgi/challenge-platform")

	assert.True(t, store.Contains("dogster.com"))
	assert.True(t, store.Contains("www.dogster.com"))
	assert.False(t, store.Contains("example.com"))
}

func TestExtractHTTPDomainsFromText(t *testing.T) {
	domains := ExtractHTTPDomainsFromText(`curl https://a.example.com/x && wget https://b.example.com/y`)
	assert.Equal(t, []string{"a.example.com", "b.example.com"}, domains)
}
