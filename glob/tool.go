package glob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "glob"

type Config struct {
	DefaultBaseDir         string
	AllowedPaths           []string
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	defaultBaseDir         string
	allowedPaths           []string
	shouldPassthroughError shared.ErrorPassthrough
}

func New(cfg Config) (*Tool, error) {
	return &Tool{
		defaultBaseDir:         cfg.DefaultBaseDir,
		allowedPaths:           append([]string{}, cfg.AllowedPaths...),
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Match files using glob patterns.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {Type: schema.String, Desc: "Glob pattern to match files.", Required: true},
			"path": {Type: schema.String, Desc: "Base directory path (relative to base_dir unless absolute)."},
			"base_dir": {Type: schema.String, Desc: "Base directory for resolving path-like parameters."},
		}),
	}, nil
}

func (t *Tool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (result string, err error) {
	defer shared.ToolInvokableDefer(&result, &err, t.shouldPassthroughError)

	params, err := shared.ParseToolArgs(argumentsInJSON)
	if err != nil {
		return "", err
	}
	baseDir, err := fsutil.ResolveBaseDir(t.defaultBaseDir, shared.GetStringParam(params, "base_dir"))
	if err != nil {
		return "", err
	}

	pattern := shared.GetStringParam(params, "pattern")
	if pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	pathValue := shared.GetStringParam(params, "path")
	if pathValue == "" {
		pathValue = "."
	}
	absBasePath, err := fsutil.ResolvePathWithin(baseDir, pathValue, t.allowedPaths)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absBasePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat base path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("base path is not a directory: %s", pathValue)
	}

	matches, err := filepath.Glob(filepath.Join(absBasePath, pattern))
	if err != nil {
		return "", fmt.Errorf("failed to execute glob pattern: %w", err)
	}

	resultLines := make([]string, 0, len(matches))
	for _, match := range matches {
		resultLines = append(resultLines, fsutil.DisplayPath(baseDir, match))
	}
	return strings.Join(resultLines, "\n"), nil
}
