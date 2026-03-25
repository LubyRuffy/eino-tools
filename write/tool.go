package write

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "write"

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
		Desc: "Write content to a file.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "File path to write (relative to base_dir unless absolute).",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "Content to write to the file.",
				Required: true,
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

	filePath := shared.GetStringParam(params, "file_path")
	if filePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	content := shared.GetStringParam(params, "content")

	absPath, err := fsutil.ResolvePathWithin(baseDir, filePath, t.allowedPaths)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return fmt.Sprintf("Updated file %s", filePath), nil
}
