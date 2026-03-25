package pythonrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const ToolName = "python_runner"

type PythonResolver func() (string, error)
type TempDirFactory func() (string, error)
type CommandRunner func(ctx context.Context, maxOutputBytes int, dir string, execPath string, args []string, extraEnv []string, stdin string) (string, string, int, error)

type Config struct {
	PythonResolver         PythonResolver
	TempDirFactory         TempDirFactory
	CommandRunner          CommandRunner
	ShouldPassthroughError shared.ErrorPassthrough
}

type Tool struct {
	pythonResolver         PythonResolver
	tempDirFactory         TempDirFactory
	commandRunner          CommandRunner
	shouldPassthroughError shared.ErrorPassthrough
}

func New(cfg Config) (*Tool, error) {
	pythonResolver := cfg.PythonResolver
	if pythonResolver == nil {
		pythonResolver = findPython
	}
	tempDirFactory := cfg.TempDirFactory
	if tempDirFactory == nil {
		tempDirFactory = func() (string, error) { return os.MkdirTemp("", "python_runner_*") }
	}
	commandRunner := cfg.CommandRunner
	if commandRunner == nil {
		commandRunner = runCmd
	}
	return &Tool{
		pythonResolver:         pythonResolver,
		tempDirFactory:         tempDirFactory,
		commandRunner:          commandRunner,
		shouldPassthroughError: cfg.ShouldPassthroughError,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Run Python code in an isolated venv with optional dependencies and timeout.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {
				Type:     schema.String,
				Desc:     "Python code to execute.",
				Required: true,
			},
			"requirements": {
				Type: schema.Array,
				Desc: "Optional pip requirement strings. Example: [\"requests==2.31.0\"]",
				ElemInfo: &schema.ParameterInfo{Type: schema.String},
			},
			"timeout_ms": {
				Type: schema.Number,
				Desc: "Timeout in milliseconds.",
			},
			"max_output_kb": {
				Type: schema.Number,
				Desc: "Max stdout/stderr capture size in KB.",
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

	code := shared.GetStringParam(params, "code")
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("code is required")
	}

	timeoutMs := shared.GetIntParam(params, "timeout_ms", 30_000)
	if timeoutMs <= 0 {
		timeoutMs = 30_000
	}
	maxOutputKB := shared.GetIntParam(params, "max_output_kb", 512)
	if maxOutputKB <= 0 {
		maxOutputKB = 512
	}
	maxOutputBytes := maxOutputKB * 1024

	var requirements []string
	if raw, ok := params["requirements"]; ok && raw != nil {
		if arr, ok := raw.([]interface{}); ok {
			requirements = make([]string, 0, len(arr))
			for _, v := range arr {
				s, ok := v.(string)
				if !ok {
					return "", fmt.Errorf("requirements must be an array of strings")
				}
				s = strings.TrimSpace(s)
				if s != "" {
					requirements = append(requirements, s)
				}
			}
		}
	}

	python, err := t.pythonResolver()
	if err != nil {
		return "", err
	}

	ctx2, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	workDir, err := t.tempDirFactory()
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	start := time.Now()
	venvDir := filepath.Join(workDir, "venv")
	_, stderr, exit, err := t.commandRunner(ctx2, maxOutputBytes, workDir, python, []string{"-m", "venv", venvDir}, nil, "")
	if err != nil {
		return marshalRunResult(exit, "", stderr, time.Since(start).Milliseconds(), true, err)
	}

	venvPython := filepath.Join(venvDir, "bin", "python")
	venvPip := filepath.Join(venvDir, "bin", "pip")

	if len(requirements) > 0 {
		reqPath := filepath.Join(workDir, "requirements.txt")
		if err := os.WriteFile(reqPath, []byte(strings.Join(requirements, "\n")+"\n"), 0o644); err != nil {
			return "", fmt.Errorf("failed to write requirements: %w", err)
		}
		_, pipStderr, pipExit, pipErr := t.commandRunner(ctx2, maxOutputBytes, workDir, venvPip, []string{"install", "--disable-pip-version-check", "--no-input", "-r", reqPath}, []string{"PIP_NO_CACHE_DIR=1"}, "")
		if pipErr != nil {
			return marshalRunResult(pipExit, "", pipStderr, time.Since(start).Milliseconds(), true, pipErr)
		}
	}

	codePath := filepath.Join(workDir, "main.py")
	if err := os.WriteFile(codePath, []byte(code), 0o644); err != nil {
		return "", fmt.Errorf("failed to write python file: %w", err)
	}

	stdout, stderr, exitCode, runErr := t.commandRunner(ctx2, maxOutputBytes, workDir, venvPython, []string{codePath}, []string{"PYTHONUNBUFFERED=1"}, "")
	elapsedMs := time.Since(start).Milliseconds()
	if runErr != nil {
		return marshalRunResult(exitCode, stdout, stderr, elapsedMs, true, runErr)
	}
	return marshalRunResult(exitCode, stdout, stderr, elapsedMs, false, nil)
}

func findPython() (string, error) {
	if p, err := osExec.LookPath("python3"); err == nil {
		return p, nil
	}
	if p, err := osExec.LookPath("python"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("python executable not found")
}

func runCmd(ctx context.Context, maxOutputBytes int, dir string, execPath string, args []string, extraEnv []string, stdin string) (string, string, int, error) {
	cmd := osExec.CommandContext(ctx, execPath, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	stdoutBuf := shared.NewLimitedBuffer(maxOutputBytes)
	stderrBuf := shared.NewLimitedBuffer(maxOutputBytes)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if startErr := cmd.Start(); startErr != nil {
		return "", "", -1, startErr
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var runErr error
	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
				_ = syscall.Kill(-pgid, syscall.SIGTERM)
			}
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
				if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
					_ = syscall.Kill(-pgid, syscall.SIGKILL)
				}
				_ = cmd.Process.Kill()
			}
		}
		runErr = ctx.Err()
	case runErr = <-done:
	}

	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*osExec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	} else if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode, runErr
}

func marshalRunResult(exitCode int, stdout string, stderr string, elapsedMs int64, failed bool, runErr error) (string, error) {
	payload := map[string]interface{}{
		"exit_code":  exitCode,
		"stdout":     stdout,
		"stderr":     stderr,
		"elapsed_ms": elapsedMs,
		"failed":     failed,
	}
	if runErr != nil {
		payload["error"] = runErr.Error()
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(b), nil
}
