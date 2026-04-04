package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/LubyRuffy/eino-tools/netproxy"
	duckduckgo "github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "web_search"

type Cache interface {
	Get(query string) (string, bool, error)
	Set(query string, result string) error
}

type SearchRunner interface {
	InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error)
}

type Config struct {
	Cache                  Cache
	HTTPClient             *http.Client
	ProxyConfig            netproxy.Config
	SearchRunner           SearchRunner
	MaxResults             int
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	cache                  Cache
	searchRunner           SearchRunner
	shouldPassthroughError shared.ErrorPassthrough
}

func New(ctx context.Context, cfg Config) (*Tool, error) {
	runner := cfg.SearchRunner
	if runner == nil {
		client := cfg.HTTPClient
		if client == nil && cfg.ProxyConfig.Enabled() {
			var err error
			client, err = netproxy.NewHTTPClient(cfg.ProxyConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create proxy HTTP client: %w", err)
			}
		}
		maxResults := cfg.MaxResults
		if maxResults <= 0 {
			maxResults = 10
		}
		searchCfg := &duckduckgo.Config{
			Region:     duckduckgo.RegionWT,
			Timeout:    30 * time.Second,
			MaxResults: maxResults,
		}
		if client != nil {
			searchCfg.HTTPClient = client
		}
		searchTool, err := duckduckgo.NewTextSearchTool(ctx, searchCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create web search tool: %w", err)
		}
		runner = searchTool
	}
	return &Tool{
		cache:                  cfg.Cache,
		searchRunner:           runner,
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Search the web for information",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "search query",
				Required: true,
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
	query := shared.GetStringParam(params, "query")
	return t.Search(ctx, query, opts...)
}

func (t *Tool) Search(ctx context.Context, query string, opts ...tool.Option) (string, error) {
	if t == nil {
		return "", fmt.Errorf("tool is nil")
	}
	if t.searchRunner == nil {
		return "", fmt.Errorf("search runner is required")
	}
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	if t.cache != nil {
		cached, ok, err := t.cache.Get(query)
		if err != nil {
			return "", fmt.Errorf("failed to get cache: %w", err)
		}
		if ok {
			return cached, nil
		}
	}

	payload, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}
	result, err := t.searchRunner.InvokableRun(ctx, string(payload), opts...)
	if err != nil {
		return "", err
	}
	if t.cache != nil && result != "" {
		if err := t.cache.Set(query, result); err != nil {
			return "", fmt.Errorf("failed to set cache: %w", err)
		}
	}
	return result, nil
}
