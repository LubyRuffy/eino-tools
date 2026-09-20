package edit

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/editutil"
	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "edit"

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
		Desc: "Edit a file in one of two modes. Mode A: search_block and replace_block must appear together; replace_block may be an empty string to delete the matched text. Mode B: patch is apply_patch text. If both modes are provided, search_block/replace_block takes precedence and patch is ignored.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "Target file path (relative to base_dir unless absolute).",
				Required: true,
			},
			"search_block": {
				Type:     schema.String,
				Desc:     "Exact text to find. Must be paired with replace_block. Use original newlines.",
				Required: false,
			},
			"replace_block": {
				Type:     schema.String,
				Desc:     "Replacement text paired with search_block. Empty string deletes the matched search_block. Omitted is not the same as empty.",
				Required: false,
			},
			"patch": {
				Type:     schema.String,
				Desc:     "apply_patch-style text for mode B (*** Begin Patch / *** Update File / @@ / +/- lines / *** End Patch). Ignored when search_block and replace_block are both provided.",
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

	displayPath := shared.GetStringParam(params, "file_path")
	if displayPath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	absPath, err := fsutil.ResolvePathWithin(baseDir, displayPath)
	if err != nil {
		return "", err
	}

	oldBytes, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	payload, err := resolveEditPayload(params)
	if err != nil {
		return "", err
	}
	if payload.useSearch {
		return t.applySearchReplace(absPath, displayPath, string(oldBytes), payload.search, payload.replace)
	}
	return t.applyPatch(absPath, displayPath, string(oldBytes), payload.patch)
}

func (t *Tool) applySearchReplace(absPath, displayPath, oldContent, searchBlock, replaceBlock string) (string, error) {
	searchNorm := editutil.NormalizeNewlines(searchBlock)
	replaceNorm := editutil.NormalizeNewlines(replaceBlock)
	if !strings.Contains(oldContent, searchNorm) {
		return "", fmt.Errorf("search_block not found in file")
	}

	newContent := strings.Replace(oldContent, searchNorm, replaceNorm, 1)
	if newContent == oldContent {
		return "", fmt.Errorf("no changes made")
	}
	if err := os.WriteFile(absPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return fmt.Sprintf("ok: replaced block in %s", displayPath), nil
}

func (t *Tool) applyPatch(absPath, displayPath, oldContent, patchText string) (string, error) {
	repls, err := editutil.ParseApplyPatchText(patchText, displayPath)
	if err != nil {
		return "", err
	}
	if len(repls) == 0 {
		return "", fmt.Errorf("patch contains no hunks")
	}

	cur := editutil.NormalizeNewlines(oldContent)
	for _, repl := range repls {
		next, count := editutil.ApplyReplaceBlockOnce(cur, editutil.NormalizeNewlines(repl.Search), editutil.NormalizeNewlines(repl.Replace))
		if count == 0 {
			return "", fmt.Errorf("patch hunk not found in file")
		}
		if count > 1 {
			return "", fmt.Errorf("patch hunk matched multiple times (count=%d); make it more specific", count)
		}
		cur = next
	}

	if strings.HasSuffix(oldContent, "\n") && !strings.HasSuffix(cur, "\n") {
		cur += "\n"
	}
	if err := os.WriteFile(absPath, []byte(cur), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return fmt.Sprintf("ok: patched %s", displayPath), nil
}
