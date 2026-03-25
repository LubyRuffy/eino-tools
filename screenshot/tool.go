package screenshot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/screenshotutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	ToolName              = "screenshot"
	defaultTimeoutMS      = 15000
	maxTimeoutMS          = 60000
	dataURLInlineSize int = 5 * 1024 * 1024
)

type Command struct {
	Name string
	Args []string
}

type CommandBuilder func(outputPath string, region *screenshotutil.Region) (*Command, error)
type CommandRunner func(ctx context.Context, name string, args ...string) error
type LookPath func(string) (string, error)

type Config struct {
	DefaultBaseDir         string
	AllowedPaths           []string
	RuntimeGOOS            string
	LookPath               LookPath
	CommandBuilder         CommandBuilder
	CommandRunner          CommandRunner
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	defaultBaseDir         string
	allowedPaths           []string
	runtimeGOOS            string
	lookPath               LookPath
	commandBuilder         CommandBuilder
	commandRunner          CommandRunner
	shouldPassthroughError shared.ErrorPassthrough
}

func New(cfg Config) (*Tool, error) {
	goos := cfg.RuntimeGOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	lookPath := cfg.LookPath
	if lookPath == nil {
		lookPath = osExec.LookPath
	}
	commandRunner := cfg.CommandRunner
	if commandRunner == nil {
		commandRunner = defaultCommandRunner
	}
	return &Tool{
		defaultBaseDir:         cfg.DefaultBaseDir,
		allowedPaths:           append([]string{}, cfg.AllowedPaths...),
		runtimeGOOS:            goos,
		lookPath:               lookPath,
		commandBuilder:         cfg.CommandBuilder,
		commandRunner:          commandRunner,
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Capture a screenshot and save it to a local image file.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"output_path": {Type: schema.String, Desc: "Optional output image path. Relative to base_dir unless absolute."},
			"region": {Type: schema.String, Desc: "Optional capture region in x,y,width,height format."},
			"include_data_url": {Type: schema.Boolean, Desc: "Optional: include base64 data URL in result (default false)."},
			"timeout_ms": {Type: schema.Number, Desc: "Optional timeout in milliseconds."},
			"base_dir": {Type: schema.String, Desc: "Base directory for resolving output path."},
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

	outputPath := shared.GetStringParam(params, "output_path")
	if outputPath == "" {
		outputPath = filepath.Join("screenshots", fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405")))
	}
	outputPath, err = screenshotutil.NormalizeOutputPath(outputPath)
	if err != nil {
		return "", err
	}
	absOutputPath, err := fsutil.ResolvePathWithin(baseDir, outputPath, t.allowedPaths)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(absOutputPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	region, err := screenshotutil.ParseRegion(shared.GetStringParam(params, "region"))
	if err != nil {
		return "", err
	}

	timeoutMs := shared.GetIntParam(params, "timeout_ms", defaultTimeoutMS)
	if timeoutMs <= 0 {
		timeoutMs = defaultTimeoutMS
	}
	if timeoutMs > maxTimeoutMS {
		timeoutMs = maxTimeoutMS
	}

	command, err := t.buildCommand(absOutputPath, region)
	if err != nil {
		return "", err
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	if err := t.commandRunner(ctxWithTimeout, command.Name, command.Args...); err != nil {
		return "", fmt.Errorf("failed to capture screenshot: %w", err)
	}

	fileInfo, err := os.Stat(absOutputPath)
	if err != nil {
		return "", fmt.Errorf("screenshot file not found after capture: %w", err)
	}

	mimeType := screenshotutil.MimeType(absOutputPath)
	payload := map[string]interface{}{
		"path":       absOutputPath,
		"image_path": absOutputPath,
		"preview":    fmt.Sprintf("![screenshot](%s)", absOutputPath),
		"mime_type":  mimeType,
		"size_bytes": fileInfo.Size(),
	}
	if region != nil {
		payload["region"] = region.String()
	}

	if shared.GetBoolParam(params, "include_data_url") {
		if err := attachDataURL(payload, absOutputPath, mimeType, fileInfo.Size()); err != nil {
			return "", err
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal screenshot result: %w", err)
	}
	return string(b), nil
}

func (t *Tool) buildCommand(outputPath string, region *screenshotutil.Region) (*Command, error) {
	if t.commandBuilder != nil {
		return t.commandBuilder(outputPath, region)
	}
	switch t.runtimeGOOS {
	case "darwin":
		args := []string{"-x"}
		if region != nil {
			args = append(args, "-R", region.String())
		}
		args = append(args, outputPath)
		return &Command{Name: "screencapture", Args: args}, nil
	case "linux":
		return t.buildLinuxCommand(outputPath, region)
	default:
		return nil, fmt.Errorf("screenshot is not supported on %s", t.runtimeGOOS)
	}
}

func (t *Tool) buildLinuxCommand(outputPath string, region *screenshotutil.Region) (*Command, error) {
	if _, err := t.lookPath("grim"); err == nil {
		if region == nil {
			return &Command{Name: "grim", Args: []string{outputPath}}, nil
		}
		return &Command{Name: "grim", Args: []string{"-g", region.String(), outputPath}}, nil
	}
	if _, err := t.lookPath("scrot"); err == nil {
		args := []string{"--silent"}
		if region != nil {
			args = append(args, "-a", region.String())
		}
		args = append(args, outputPath)
		return &Command{Name: "scrot", Args: args}, nil
	}
	if _, err := t.lookPath("import"); err == nil {
		args := []string{"-window", "root"}
		if region != nil {
			args = append(args, "-crop", fmt.Sprintf("%dx%d+%d+%d", region.Width, region.Height, region.X, region.Y))
		}
		args = append(args, outputPath)
		return &Command{Name: "import", Args: args}, nil
	}
	if _, err := t.lookPath("gnome-screenshot"); err == nil {
		if region != nil {
			return nil, fmt.Errorf("region capture requires grim/scrot/import on linux")
		}
		return &Command{Name: "gnome-screenshot", Args: []string{"-f", outputPath}}, nil
	}
	return nil, fmt.Errorf("no supported screenshot command found on linux (grim/scrot/import/gnome-screenshot)")
}

func attachDataURL(payload map[string]interface{}, imagePath string, mimeType string, sizeBytes int64) error {
	if sizeBytes > int64(dataURLInlineSize) {
		payload["data_url_omitted"] = true
		payload["data_url_omitted_reason"] = fmt.Sprintf("image too large: %d bytes", sizeBytes)
		return nil
	}
	content, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("failed to read screenshot file: %w", err)
	}
	payload["screenshot"] = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content))
	return nil
}

func defaultCommandRunner(ctx context.Context, name string, args ...string) error {
	cmd := osExec.CommandContext(ctx, name, args...)
	return cmd.Run()
}
