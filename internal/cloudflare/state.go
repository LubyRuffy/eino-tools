package cloudflare

import (
	"strings"
	"sync"
	"time"
)

type ProtectedDomains struct {
	mu      sync.Mutex
	ttl     time.Duration
	domains map[string]time.Time
}

func NewProtectedDomains(ttl time.Duration) *ProtectedDomains {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &ProtectedDomains{
		ttl:     ttl,
		domains: make(map[string]time.Time),
	}
}

func (s *ProtectedDomains) Mark(input string) {
	if s == nil {
		return
	}
	domain := NormalizeDomain(strings.TrimSpace(input))
	if strings.Contains(input, "://") {
		domain = ExtractDomainFromURL(input)
	}
	if domain == "" {
		return
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	s.domains[domain] = now.Add(s.ttl)
}

func (s *ProtectedDomains) Contains(domain string) bool {
	if s == nil {
		return false
	}
	normalized := NormalizeDomain(domain)
	if normalized == "" {
		return false
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	for protectedDomain := range s.domains {
		if protectedDomain == normalized {
			return true
		}
		if strings.HasSuffix(normalized, "."+protectedDomain) {
			return true
		}
		if strings.HasSuffix(protectedDomain, "."+normalized) {
			return true
		}
	}
	return false
}

func (s *ProtectedDomains) pruneLocked(now time.Time) {
	for domain, expiresAt := range s.domains {
		if now.After(expiresAt) {
			delete(s.domains, domain)
		}
	}
}
