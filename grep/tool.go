package grep

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "grep"

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

type grepMatch struct {
	Path    string
	Line    int
	Content string
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
		Desc: "Search for a pattern in files.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {Type: schema.String, Desc: "Regular expression pattern to search for.", Required: true},
			"path": {Type: schema.String, Desc: "Base directory path (relative to base_dir unless absolute)."},
			"glob": {Type: schema.String, Desc: "Glob pattern to filter files."},
			"output_mode": {Type: schema.String, Desc: "Output mode: files_with_matches, content, or count."},
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
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
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

	globPattern := shared.GetStringParam(params, "glob")
	outputMode := shared.GetStringParam(params, "output_mode")
	if outputMode == "" {
		outputMode = "files_with_matches"
	}

	var matches []grepMatch
	if info.IsDir() {
		if globPattern != "" {
			files, err := filepath.Glob(filepath.Join(absBasePath, globPattern))
			if err != nil {
				return "", fmt.Errorf("invalid glob pattern: %w", err)
			}
			for _, file := range files {
				fileInfo, err := os.Stat(file)
				if err != nil || fileInfo.IsDir() {
					continue
				}
				fileMatches, err := t.searchFile(file, re)
				if err != nil {
					return "", err
				}
				matches = append(matches, fileMatches...)
			}
		} else {
			err = filepath.Walk(absBasePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					fileMatches, err := t.searchFile(path, re)
					if err != nil {
						return err
					}
					matches = append(matches, fileMatches...)
				}
				return nil
			})
			if err != nil {
				return "", fmt.Errorf("failed to walk directory: %w", err)
			}
		}
	} else {
		fileMatches, err := t.searchFile(absBasePath, re)
		if err != nil {
			return "", err
		}
		matches = append(matches, fileMatches...)
	}

	switch outputMode {
	case "count":
		return strconv.Itoa(len(matches)), nil
	case "content":
		var out strings.Builder
		for _, match := range matches {
			out.WriteString(fmt.Sprintf("%s:%d:%s\n", fsutil.DisplayPath(baseDir, match.Path), match.Line, match.Content))
		}
		return out.String(), nil
	default:
		seen := make(map[string]struct{})
		files := make([]string, 0)
		for _, match := range matches {
			if _, ok := seen[match.Path]; ok {
				continue
			}
			files = append(files, fsutil.DisplayPath(baseDir, match.Path))
			seen[match.Path] = struct{}{}
		}
		return strings.Join(files, "\n"), nil
	}
}

func (t *Tool) searchFile(filePath string, re *regexp.Regexp) ([]grepMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var matches []grepMatch
	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, grepMatch{Path: filePath, Line: lineNum, Content: line})
		}
		lineNum++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return matches, nil
}
