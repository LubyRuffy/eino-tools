package mcpserver

import (
	"context"

	"github.com/LubyRuffy/eino-tools/edit"
	"github.com/LubyRuffy/eino-tools/exec"
	"github.com/LubyRuffy/eino-tools/glob"
	"github.com/LubyRuffy/eino-tools/grep"
	"github.com/LubyRuffy/eino-tools/ls"
	"github.com/LubyRuffy/eino-tools/pythonrunner"
	"github.com/LubyRuffy/eino-tools/read"
	"github.com/LubyRuffy/eino-tools/screenshot"
	"github.com/LubyRuffy/eino-tools/tree"
	"github.com/LubyRuffy/eino-tools/webfetch"
	"github.com/LubyRuffy/eino-tools/websearch"
	"github.com/LubyRuffy/eino-tools/write"
)

func NewToolset(ctx context.Context, cfg Config) ([]EinoTool, error) {
	cfg = cfg.WithDefaults()

	proxyCfg := resolveProxyConfig(cfg, nil)
	networkHTTPClient, err := newNetworkHTTPClient(cfg, nil)
	if err != nil {
		return nil, err
	}

	webSearchTool, err := websearch.New(ctx, websearch.Config{
		HTTPClient:  networkHTTPClient,
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		return nil, err
	}
	webFetchTool, err := webfetch.New(webfetch.Config{
		HTTPClient:  networkHTTPClient,
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		return nil, err
	}
	execTool, err := exec.New(exec.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	readTool, err := read.New(read.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	writeTool, err := write.New(write.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	editTool, err := edit.New(edit.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	lsTool, err := ls.New(ls.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	treeTool, err := tree.New(tree.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	globTool, err := glob.New(glob.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	grepTool, err := grep.New(grep.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}
	pythonTool, err := pythonrunner.New(pythonrunner.Config{})
	if err != nil {
		return nil, err
	}
	screenshotTool, err := screenshot.New(screenshot.Config{DefaultBaseDir: cfg.BaseDir})
	if err != nil {
		return nil, err
	}

	return []EinoTool{
		webSearchTool,
		webFetchTool,
		execTool,
		readTool,
		writeTool,
		editTool,
		lsTool,
		treeTool,
		globTool,
		grepTool,
		pythonTool,
		screenshotTool,
	}, nil
}
