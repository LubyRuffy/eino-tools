package webfetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"github.com/LubyRuffy/eino-tools/internal/cloudflare"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const ToolName = "web_fetch"

type Cache interface {
	Get(key string) (string, bool, error)
	Set(key string, content string) error
}

type RequestCookie struct {
	Name  string
	Value string
	Path  string
}

type CookieProvider interface {
	GetCookies(domain string) []RequestCookie
}

type ProtectedDomains interface {
	Mark(input string)
	Contains(domain string) bool
}

type HTMLFetcher func(ctx context.Context, rawURL string) (string, error)
type RenderFetcher func(ctx context.Context, rawURL string) (string, error)
type BrowserFetcher func(ctx context.Context, rawURL string, render bool) (string, error)
type ChallengeDetector func(err error) (string, bool)
type HeaderProvider func(rawURL string) http.Header

type ChallengeRequest struct {
	ToolName  string
	URL       string
	TimeoutMS int
}

type ChallengeHandler func(ctx context.Context, req ChallengeRequest) error

type Config struct {
	HTTPClient             *http.Client
	Cache                  Cache
	HeaderProvider         HeaderProvider
	CookieProvider         CookieProvider
	HTMLFetcher            HTMLFetcher
	RenderFetcher          RenderFetcher
	BrowserFetch           BrowserFetcher
	ChallengeDetector      ChallengeDetector
	ProtectedDomains       ProtectedDomains
	ChallengeHandler       ChallengeHandler
	ChallengeTimeoutMS     int
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	httpClient             *http.Client
	cache                  Cache
	headerProvider         HeaderProvider
	cookieProvider         CookieProvider
	htmlFetcher            HTMLFetcher
	renderFetcher          RenderFetcher
	browserFetch           BrowserFetcher
	challengeDetector      ChallengeDetector
	protectedDomains       ProtectedDomains
	challengeHandler       ChallengeHandler
	challengeTimeoutMS     int
	shouldPassthroughError shared.ErrorPassthrough
}

func New(cfg Config) (*Tool, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	timeoutMS := cfg.ChallengeTimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 120000
	}
	t := &Tool{
		httpClient:             client,
		cache:                  cfg.Cache,
		headerProvider:         cfg.HeaderProvider,
		cookieProvider:         cfg.CookieProvider,
		browserFetch:           cfg.BrowserFetch,
		challengeDetector:      cfg.ChallengeDetector,
		protectedDomains:       cfg.ProtectedDomains,
		challengeHandler:       cfg.ChallengeHandler,
		challengeTimeoutMS:     timeoutMS,
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}
	if cfg.RenderFetcher != nil {
		t.renderFetcher = cfg.RenderFetcher
	} else {
		t.renderFetcher = t.fetchRenderedMarkdown
	}
	if cfg.HTMLFetcher != nil {
		t.htmlFetcher = cfg.HTMLFetcher
	} else {
		t.htmlFetcher = t.fetchHTML
	}
	return t, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Fetch the content of a web page and extract the text (render mode returns Markdown via Readability)",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Desc:     "page url address",
				Required: true,
			},
			"render": {
				Type:     schema.Boolean,
				Desc:     "Optional: whether to render the page with a headless browser. Default false.",
				Required: false,
			},
		}),
	}, nil
}

func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (out string, err error) {
	defer shared.ToolInvokableDefer(&out, &err, t.shouldPassthroughError)

	params, err := shared.ParseToolArgs(argumentsInJSON)
	if err != nil {
		return "", err
	}
	urlValue := shared.GetStringParam(params, "url")
	render := shared.GetBoolParam(params, "render")
	return t.Fetch(ctx, urlValue, render)
}

func (t *Tool) Fetch(ctx context.Context, rawURL string, render bool) (string, error) {
	if t == nil {
		return "", fmt.Errorf("tool is nil")
	}
	if strings.TrimSpace(rawURL) == "" {
		return "", fmt.Errorf("URL is required")
	}

	if t.shouldUseBrowserFetch(rawURL) {
		return t.executeBrowserFetch(ctx, rawURL, render)
	}

	cacheKey := strings.TrimSpace(rawURL)
	renderCacheKey := fmt.Sprintf("%s::render", cacheKey)
	if render {
		cacheKey = renderCacheKey
	}
	if t.cache != nil {
		if render {
			if cached, ok, err := t.cache.Get(renderCacheKey); err == nil && ok {
				return cached, nil
			}
		} else {
			if cached, ok, err := t.cache.Get(renderCacheKey); err == nil && ok {
				return cached, nil
			}
			if cached, ok, err := t.cache.Get(strings.TrimSpace(rawURL)); err == nil && ok {
				return cached, nil
			}
		}
	}

	result, err := t.executeFetch(ctx, rawURL, render)
	if err != nil {
		if challengeURL, isChallenge := t.detectChallenge(err, rawURL); isChallenge {
			if t.protectedDomains != nil {
				t.protectedDomains.Mark(challengeURL)
			}
			if browserResult, browserErr := t.tryBrowserFetch(ctx, rawURL, render); browserErr == nil && strings.TrimSpace(browserResult) != "" {
				result = browserResult
				err = nil
			} else if browserErr != nil {
				err = browserErr
			}
		}

		if err != nil {
			if targetURL, isChallenge := t.detectChallenge(err, rawURL); t.challengeHandler != nil && isChallenge {
				if handoffErr := t.challengeHandler(ctx, ChallengeRequest{
					ToolName:  ToolName,
					URL:       targetURL,
					TimeoutMS: t.challengeTimeoutMS,
				}); handoffErr != nil {
					return "", handoffErr
				}
				result, err = t.executeFetch(ctx, rawURL, render)
				if err != nil {
					if _, stillChallenge := t.detectChallenge(err, rawURL); stillChallenge {
						return "", fmt.Errorf("Cloudflare challenge still exists after manual verification, please retry manual verification")
					}
					return "", err
				}
			} else {
				return "", err
			}
		}
	}

	if t.cache != nil && result != "" {
		if err := t.cache.Set(cacheKey, result); err != nil {
			return "", fmt.Errorf("failed to set cache: %w", err)
		}
	}
	return result, nil
}

func (t *Tool) detectChallenge(err error, fallbackURL string) (string, bool) {
	if err == nil {
		return "", false
	}
	if cloudflare.IsChallengeError(err) {
		return challengeURLFromError(err, fallbackURL), true
	}
	if t.challengeDetector != nil {
		if detectedURL, ok := t.challengeDetector(err); ok {
			detectedURL = strings.TrimSpace(detectedURL)
			if detectedURL == "" {
				detectedURL = strings.TrimSpace(fallbackURL)
			}
			return detectedURL, true
		}
	}
	if cloudflare.IsLikelyChallengeText(err.Error()) {
		return challengeURLFromError(err, fallbackURL), true
	}
	return "", false
}

func (t *Tool) shouldUseBrowserFetch(rawURL string) bool {
	if t == nil || t.protectedDomains == nil || t.browserFetch == nil {
		return false
	}
	domain := cloudflare.ExtractDomainFromURL(rawURL)
	if domain == "" {
		return false
	}
	return t.protectedDomains.Contains(domain)
}

func (t *Tool) executeBrowserFetch(ctx context.Context, rawURL string, render bool) (string, error) {
	if t == nil || t.browserFetch == nil {
		return "", fmt.Errorf("browser fetch is not configured")
	}
	return t.browserFetch(ctx, rawURL, render)
}

func (t *Tool) tryBrowserFetch(ctx context.Context, rawURL string, render bool) (string, error) {
	if t == nil || t.browserFetch == nil {
		return "", nil
	}
	return t.browserFetch(ctx, rawURL, render)
}

func (t *Tool) executeFetch(ctx context.Context, rawURL string, render bool) (string, error) {
	if render {
		return t.renderFetcher(ctx, rawURL)
	}

	htmlContent, err := t.htmlFetcher(ctx, rawURL)
	if err != nil {
		return "", err
	}

	articleText, err := ExtractReadableText(htmlContent, rawURL)
	if err != nil {
		if ShouldRetryWithRender(htmlContent) {
			return t.executeFetch(ctx, rawURL, true)
		}
		return "", fmt.Errorf("failed to extract readability content: %w", err)
	}

	if ShouldRetryWithRender(articleText) {
		return t.executeFetch(ctx, rawURL, true)
	}
	return articleText, nil
}

func (t *Tool) fetchHTML(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	t.applyRequestHeaders(req)
	t.applyRequestCookies(req)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	bodyText := string(body)

	finalURL := strings.TrimSpace(rawURL)
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	if cloudflare.IsHTTPBlock(resp.StatusCode, resp.Header) || cloudflare.DetectFromPageContent("", finalURL, bodyText) {
		return "", &cloudflare.ChallengeError{
			URL:    finalURL,
			Reason: fmt.Sprintf("cloudflare challenge detected (%d)", resp.StatusCode),
		}
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}
	return bodyText, nil
}

func (t *Tool) applyRequestHeaders(req *http.Request) {
	if req == nil {
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	if t == nil || t.headerProvider == nil || req.URL == nil {
		return
	}
	extraHeaders := t.headerProvider(req.URL.String())
	for key, values := range extraHeaders {
		if len(values) == 0 {
			continue
		}
		req.Header.Del(key)
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
}

func (t *Tool) applyRequestCookies(req *http.Request) {
	if req == nil || req.URL == nil || t == nil || t.cookieProvider == nil {
		return
	}
	domain := cloudflare.ExtractDomainFromURL(req.URL.String())
	if domain == "" {
		return
	}
	for _, cookie := range t.cookieProvider.GetCookies(domain) {
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: cookie.Value,
			Path:  normalizeCookiePath(cookie.Path),
		})
	}
}

func normalizeCookiePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

func (t *Tool) fetchRenderedMarkdown(ctx context.Context, rawURL string) (string, error) {
	browser, launch, err := launchRodBrowser(ctx, true)
	if err != nil {
		return "", err
	}
	defer browser.Close()
	defer launch.Kill()

	page := browser.MustPage().Context(ctx)
	defer page.Close()
	page = page.Timeout(45 * time.Second)
	if err := page.Navigate(rawURL); err != nil {
		return "", fmt.Errorf("failed to navigate: %w", err)
	}
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("failed to wait for load: %w", err)
	}

	title := ""
	pageURL := strings.TrimSpace(rawURL)
	if info, err := page.Info(); err == nil && info != nil {
		title = info.Title
		if strings.TrimSpace(info.URL) != "" {
			pageURL = strings.TrimSpace(info.URL)
		}
	}
	html := ""
	if body, err := page.HTML(); err == nil {
		html = body
	}
	if cloudflare.DetectFromPageContent(title, pageURL, html) {
		return "", &cloudflare.ChallengeError{
			URL:    pageURL,
			Reason: "cloudflare challenge detected",
		}
	}

	markdown, err := evaluateReadabilityMarkdown(page)
	if err != nil {
		var cfErr *cloudflare.ChallengeError
		if errors.As(err, &cfErr) {
			return "", err
		}
		fallbackCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		htmlContent, htmlErr := t.fetchHTML(fallbackCtx, rawURL)
		if htmlErr != nil {
			return "", err
		}
		text, textErr := ExtractReadableText(htmlContent, rawURL)
		if textErr != nil {
			return "", err
		}
		return text, nil
	}
	return markdown, nil
}

func launchRodBrowser(ctx context.Context, headless bool) (*rod.Browser, *launcher.Launcher, error) {
	launch := launcher.New().Context(ctx).Headless(headless)
	controlURL, err := launch.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser: %w", err)
	}
	browser := rod.New().ControlURL(controlURL).Context(ctx)
	if err := browser.Connect(); err != nil {
		launch.Kill()
		return nil, nil, fmt.Errorf("failed to connect browser: %w", err)
	}
	return browser, launch, nil
}

func evaluateReadabilityMarkdown(page *rod.Page) (string, error) {
	const readabilityURL = "https://cdn.jsdelivr.net/npm/@mozilla/readability@0.5.0/Readability.js"
	const turndownURL = "https://cdn.jsdelivr.net/npm/turndown@7.1.2/dist/turndown.js"

	script := fmt.Sprintf(`async () => {
  const loadScript = (src) => new Promise((resolve, reject) => {
    const existing = document.querySelector('script[data-zw-src="' + src + '"]');
    if (existing) { resolve(); return; }
    const el = document.createElement('script');
    el.src = src;
    el.async = true;
    el.dataset.zwSrc = src;
    el.onload = () => resolve();
    el.onerror = () => reject(new Error('failed to load ' + src));
    document.head.appendChild(el);
  });

  await loadScript('%s');
  await loadScript('%s');

  if (typeof Readability === 'undefined' || typeof TurndownService === 'undefined') {
    throw new Error('Readability or TurndownService not available');
  }

  const doc = document.cloneNode(true);
  const reader = new Readability(doc);
  const article = reader.parse();
  if (!article || !article.content) {
    throw new Error('empty content');
  }

  const turndownService = new TurndownService();
  const markdown = turndownService.turndown(article.content);
  return { title: article.title || '', markdown: markdown || '' };
};`, readabilityURL, turndownURL)

	result, err := page.Evaluate(rod.Eval(script).ByPromise())
	if err != nil {
		return "", fmt.Errorf("failed to evaluate readability: %w", err)
	}
	if result == nil || result.Value.Nil() {
		return "", fmt.Errorf("empty readability result")
	}

	data := result.Value.Map()
	markdown := strings.TrimSpace(data["markdown"].Str())
	title := strings.TrimSpace(data["title"].Str())
	if markdown == "" {
		return "", fmt.Errorf("empty content")
	}
	if title != "" {
		return fmt.Sprintf("# %s\n\n%s", title, markdown), nil
	}
	return markdown, nil
}

func ExtractReadableText(htmlContent, pageURL string) (string, error) {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}

	article, err := readability.FromReader(strings.NewReader(htmlContent), parsedURL)
	if err != nil {
		return "", err
	}

	buf := shared.NewLimitedBuffer(10000)
	if err := article.RenderText(buf); err != nil {
		return "", err
	}
	text := strings.TrimSpace(buf.String())
	if text == "" {
		return "", fmt.Errorf("empty content")
	}
	if buf.Truncated() {
		return text + "... (truncated)", nil
	}
	return text, nil
}

func ShouldRetryWithRender(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "enable javascript") || strings.Contains(lower, "javascript is required") {
		return true
	}
	if strings.Contains(lower, "app") && strings.Contains(lower, "root") && len(trimmed) < 600 {
		return true
	}
	if strings.Contains(lower, "__next") || strings.Contains(lower, "id=\"app\"") || strings.Contains(lower, "id='app'") {
		return true
	}
	return false
}

func challengeURLFromError(err error, fallbackURL string) string {
	if err != nil {
		var challengeErr *cloudflare.ChallengeError
		if errors.As(err, &challengeErr) && strings.TrimSpace(challengeErr.URL) != "" {
			return strings.TrimSpace(challengeErr.URL)
		}
	}
	return strings.TrimSpace(fallbackURL)
}
