package ls

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "ls"

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
		Desc: "List files and directories.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "Directory path to list (relative to base_dir unless absolute).",
				Required: false,
			},
			"base_dir": {
				Type:     schema.String,
				Desc:     "Base directory for resolving path-like parameters.",
				Required: false,
			},
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

	pathValue := shared.GetStringParam(params, "path")
	if pathValue == "" {
		pathValue = "."
	}
	absPath, err := fsutil.ResolvePathWithin(baseDir, pathValue, t.allowedPaths)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", pathValue)
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var out strings.Builder
	for _, entry := range entries {
		entryPath := fsutil.DisplayPath(baseDir, filepathJoin(absPath, entry.Name()))
		out.WriteString(entryPath)
		if entry.IsDir() {
			out.WriteString(string(os.PathSeparator))
		}
		out.WriteString("\n")
	}
	return out.String(), nil
}

func filepathJoin(elem ...string) string {
	return strings.Join(elem, string(os.PathSeparator))
}
