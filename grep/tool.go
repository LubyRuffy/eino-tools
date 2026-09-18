package grep

import (
	"bufio"
	"bytes"
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

const (
	ToolName          = "grep"
	DefaultMaxMatches = 200
	TruncatedSuffix   = "[truncated]"
	maxScanTokenSize  = 1 << 20
	maxGrepFileBytes  = 8 << 20
	maxReportedLine   = 4096
)

type Config struct {
	DefaultBaseDir         string
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	defaultBaseDir         string
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
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Search for a pattern in files. Skips VCS/dependency/build directories, gitignored paths, binaries, and unreadable or overlong files instead of failing the whole search.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern":     {Type: schema.String, Desc: "Regular expression pattern to search for.", Required: true},
			"path":        {Type: schema.String, Desc: "Base directory path (relative to base_dir unless absolute)."},
			"glob":        {Type: schema.String, Desc: "Glob pattern to filter files."},
			"output_mode": {Type: schema.String, Desc: "Output mode: files_with_matches, content, or count."},
			"base_dir":    {Type: schema.String, Desc: "Base directory for resolving path-like parameters."},
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
	absBasePath, err := fsutil.ResolvePathWithin(baseDir, pathValue)
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

	ignore := fsutil.LoadWalkIgnore(absBasePath)
	var matches []grepMatch
	truncated := false
	limitReached := func() bool {
		if outputMode == "count" {
			return false
		}
		if outputMode == "content" {
			return len(matches) >= DefaultMaxMatches
		}
		seen := make(map[string]struct{})
		for _, match := range matches {
			seen[match.Path] = struct{}{}
		}
		return len(seen) >= DefaultMaxMatches
	}

	collect := func(path string) {
		if path != absBasePath && ignore.SkipFile(path) {
			return
		}
		matches = append(matches, t.searchFile(path, re)...)
	}

	if info.IsDir() {
		if globPattern != "" {
			files, globErr := filepath.Glob(filepath.Join(absBasePath, globPattern))
			if globErr != nil {
				return "", fmt.Errorf("invalid glob pattern: %w", globErr)
			}
			for _, file := range files {
				fileInfo, statErr := os.Stat(file)
				if statErr != nil || fileInfo.IsDir() {
					continue
				}
				collect(file)
				if limitReached() {
					truncated = true
					break
				}
			}
		} else {
			walkErr := filepath.Walk(absBasePath, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if info.IsDir() {
					if ignore.SkipDir(path, info.Name()) {
						return filepath.SkipDir
					}
					return nil
				}
				collect(path)
				if limitReached() {
					truncated = true
					return filepath.SkipAll
				}
				return nil
			})
			if walkErr != nil {
				return "", fmt.Errorf("failed to walk directory: %w", walkErr)
			}
		}
	} else {
		collect(absBasePath)
	}

	return formatGrepMatches(baseDir, outputMode, matches, truncated), nil
}

func formatGrepMatches(baseDir, outputMode string, matches []grepMatch, truncated bool) string {
	switch outputMode {
	case "count":
		return strconv.Itoa(len(matches))
	case "content":
		if len(matches) > DefaultMaxMatches {
			matches = matches[:DefaultMaxMatches]
			truncated = true
		}
		var out strings.Builder
		for _, match := range matches {
			content := match.Content
			if len(content) > maxReportedLine {
				content = content[:maxReportedLine] + "…"
			}
			out.WriteString(fmt.Sprintf("%s:%d:%s\n", fsutil.DisplayPath(baseDir, match.Path), match.Line, content))
		}
		if truncated {
			out.WriteString(TruncatedSuffix + "\n")
		}
		return out.String()
	default:
		seen := make(map[string]struct{})
		files := make([]string, 0)
		for _, match := range matches {
			if _, ok := seen[match.Path]; ok {
				continue
			}
			if len(files) >= DefaultMaxMatches {
				truncated = true
				break
			}
			files = append(files, fsutil.DisplayPath(baseDir, match.Path))
			seen[match.Path] = struct{}{}
		}
		out := strings.Join(files, "\n")
		if truncated {
			if out != "" {
				out += "\n"
			}
			out += TruncatedSuffix
		}
		return out
	}
}

func (t *Tool) searchFile(filePath string, re *regexp.Regexp) []grepMatch {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() || info.Size() > maxGrepFileBytes {
		return nil
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	head := make([]byte, 8000)
	n, _ := file.Read(head)
	if n > 0 {
		head = head[:n]
	} else {
		head = nil
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return nil
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)
	var matches []grepMatch
	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, grepMatch{Path: filePath, Line: lineNum, Content: line})
		}
		lineNum++
	}
	if scanner.Err() != nil {
		return nil
	}
	return matches
}
