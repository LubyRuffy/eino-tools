package read

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/text/transform"
)

const ToolName = "read"

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
		Desc: "Read a text file with paging and basic encoding detection.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"file_path": {
				Type:     schema.String,
				Desc:     "File path to read (relative to base_dir unless absolute).",
				Required: true,
			},
			"offset": {
				Type:     schema.Number,
				Desc:     "Start line number (1-based).",
				Required: false,
			},
			"limit": {
				Type:     schema.Number,
				Desc:     "Max lines to read.",
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

	filePath := shared.GetStringParam(params, "file_path")
	if filePath == "" {
		return "", fmt.Errorf("file_path is required")
	}
	absPath, err := fsutil.ResolvePathWithin(baseDir, filePath)
	if err != nil {
		return "", err
	}

	offset := shared.GetIntParam(params, "offset", 1)
	if offset <= 0 {
		offset = 1
	}
	limit := shared.GetIntParam(params, "limit", 200)
	if limit <= 0 {
		limit = 200
	}

	f, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	sample := make([]byte, 4096)
	n, _ := f.Read(sample)
	sample = sample[:n]

	encodingName, decoder := detectTextDecoder(sample)
	skip := int64(0)
	if encodingName == "utf-8" {
		skip = int64(utf8BOMLen(sample))
	}
	if _, err := f.Seek(skip, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file: %w", err)
	}

	var reader io.Reader = f
	if decoder != nil {
		reader = transform.NewReader(f, decoder)
	}

	br := bufio.NewReader(reader)
	currentLine := 0
	readLines := 0
	var out bytes.Buffer

	writeLine := func(lineNo int, line string) {
		fmt.Fprintf(&out, "%d|%s\n", lineNo, line)
	}

	for {
		lineBytes, readErr := br.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return "", fmt.Errorf("failed to read file: %w", readErr)
		}
		if len(lineBytes) == 0 && readErr == io.EOF {
			break
		}
		line := string(bytes.TrimRight(lineBytes, "\r\n"))
		currentLine++
		if currentLine < offset {
			if readErr == io.EOF {
				break
			}
			continue
		}
		writeLine(currentLine, line)
		readLines++
		if readLines >= limit {
			break
		}
		if readErr == io.EOF {
			break
		}
	}

	header := fmt.Sprintf("encoding=%s path=%s offset=%d limit=%d\n", encodingName, filePath, offset, limit)
	return header + out.String(), nil
}
