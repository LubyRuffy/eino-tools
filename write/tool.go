package write

import (
	"context"
	"fmt"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "write"

type Config struct {
	DefaultBaseDir         string
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	defaultBaseDir         string
	shouldPassthroughError shared.ErrorPassthrough
}

func New(cfg Config) (*Tool, error) {
	return &Tool{
		defaultBaseDir:         cfg.DefaultBaseDir,
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Write the complete file body to disk. Both file_path and content are required. content is the full file text; the field name is content, not contents. Success means those bytes are readable back from the resolved path. Relative file_path is resolved against base_dir (or DefaultBaseDir), not the process working directory.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "Target file path. Relative paths resolve against base_dir, not the process working directory.",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "Complete file body as a string. Field name is content, not contents. An empty string writes an empty file; omitting content is an error.",
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

	payload, err := resolveWritePayload(params)
	if err != nil {
		return "", err
	}

	absPath, err := fsutil.ResolvePathWithin(baseDir, payload.filePath)
	if err != nil {
		return "", err
	}
	content := []byte(payload.content)
	if err := writeVerified(absPath, content); err != nil {
		return "", err
	}
	return fmt.Sprintf("Updated file %s (%d bytes)", absPath, len(content)), nil
}
