package tree

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "tree"

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
		Desc: "Display a directory tree with depth control and filtering.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {Type: schema.String, Desc: "Directory path to display (relative to base_dir unless absolute).", Required: true},
			"max_depth": {Type: schema.Number, Desc: "Max recursion depth (0 means only the root entry)."},
			"include": {Type: schema.String, Desc: "Comma-separated glob patterns to include (match base name)."},
			"exclude": {Type: schema.String, Desc: "Comma-separated glob patterns to exclude (match base name)."},
			"only_dirs": {Type: schema.Boolean, Desc: "Only show directories."},
			"max_entries": {Type: schema.Number, Desc: "Max number of entries to output."},
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

	pathValue := shared.GetStringParam(params, "path")
	if pathValue == "" {
		return "", fmt.Errorf("path is required")
	}
	root, err := fsutil.ResolvePathWithin(baseDir, pathValue, t.allowedPaths)
	if err != nil {
		return "", err
	}

	maxDepth := shared.GetIntParam(params, "max_depth", 4)
	if maxDepth < 0 {
		maxDepth = 0
	}
	maxEntries := shared.GetIntParam(params, "max_entries", 2000)
	if maxEntries <= 0 {
		maxEntries = 2000
	}
	onlyDirs := shared.GetBoolParam(params, "only_dirs")
	includePatterns := splitPatterns(shared.GetStringParam(params, "include"))
	excludePatterns := splitPatterns(shared.GetStringParam(params, "exclude"))

	var out strings.Builder
	entries := 0

	rootInfo, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}
	displayRoot := fsutil.DisplayPath(baseDir, root)
	if rootInfo.IsDir() {
		out.WriteString(displayRoot)
		out.WriteString(string(os.PathSeparator))
	} else {
		out.WriteString(displayRoot)
	}
	out.WriteString("\n")

	if !rootInfo.IsDir() {
		return out.String(), nil
	}

	err = t.walkDir(&out, root, baseDir, 0, maxDepth, includePatterns, excludePatterns, onlyDirs, &entries, maxEntries)
	if err != nil {
		return "", err
	}
	if entries >= maxEntries {
		out.WriteString("... (truncated)\n")
	}
	return out.String(), nil
}

func (t *Tool) walkDir(out *strings.Builder, dir string, baseDir string, depth int, maxDepth int, include []string, exclude []string, onlyDirs bool, entries *int, maxEntries int) error {
	if *entries >= maxEntries || depth >= maxDepth {
		return nil
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read dir: %w", err)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir() != items[j].IsDir() {
			return items[i].IsDir()
		}
		return strings.ToLower(items[i].Name()) < strings.ToLower(items[j].Name())
	})

	for _, item := range items {
		if *entries >= maxEntries {
			return nil
		}
		name := item.Name()
		if shouldExcludeName(name, include, exclude) {
			continue
		}

		full := filepath.Join(dir, name)
		if onlyDirs && !item.IsDir() {
			continue
		}

		indent := strings.Repeat("  ", depth+1)
		out.WriteString(indent)
		out.WriteString(fsutil.DisplayPath(baseDir, full))
		if item.IsDir() {
			out.WriteString(string(os.PathSeparator))
		}
		out.WriteString("\n")
		*entries++

		if item.IsDir() {
			if err := t.walkDir(out, full, baseDir, depth+1, maxDepth, include, exclude, onlyDirs, entries, maxEntries); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitPatterns(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func shouldExcludeName(name string, include []string, exclude []string) bool {
	if len(include) > 0 {
		ok := false
		for _, pattern := range include {
			if matched, _ := filepath.Match(pattern, name); matched {
				ok = true
				break
			}
		}
		if !ok {
			return true
		}
	}
	for _, pattern := range exclude {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
