package exec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/LubyRuffy/eino-tools/internal/cloudflare"
	"github.com/LubyRuffy/eino-tools/internal/fsutil"
	"github.com/LubyRuffy/eino-tools/internal/shared"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	ToolName                = "exec"
	defaultCommandTimeoutMS = 10000
	maxCommandTimeoutMS     = 300000
)

var bashLookPath = osExec.LookPath

type ProtectedDomains interface {
	Mark(input string)
	Contains(domain string) bool
}

type ChallengeRequest struct {
	ToolName  string
	URL       string
	TimeoutMS int
}

type ChallengeHandler func(ctx context.Context, req ChallengeRequest) error

type Config struct {
	DefaultBaseDir         string
	ProtectedDomains       ProtectedDomains
	ChallengeHandler       ChallengeHandler
	ChallengeTimeoutMS     int
	ShouldPassthroughError shared.ErrorPassthrough
	ShellPath              string
}

type Params struct {
	Command     string
	CWD         string
	Stdin       string
	TimeoutMS   int
	MaxOutputKB int
	Env         map[string]string
	BaseDir     string
}

type Tool struct {
	defaultBaseDir         string
	protectedDomains       ProtectedDomains
	challengeHandler       ChallengeHandler
	challengeTimeoutMS     int
	shouldPassthroughError shared.ErrorPassthrough
	shellPath              string
}

func New(cfg Config) (*Tool, error) {
	baseDir := strings.TrimSpace(cfg.DefaultBaseDir)
	if baseDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			baseDir = "."
		} else {
			baseDir = wd
		}
	}
	shellPath := strings.TrimSpace(cfg.ShellPath)
	if shellPath == "" {
		shellPath, _ = resolveShellPath()
	}
	timeoutMS := cfg.ChallengeTimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 120000
	}

	return &Tool{
		defaultBaseDir:         baseDir,
		protectedDomains:       cfg.ProtectedDomains,
		challengeHandler:       cfg.ChallengeHandler,
		challengeTimeoutMS:     timeoutMS,
		shouldPassthroughError: cfg.ShouldPassthroughError,
		shellPath:              shellPath,
	}, nil
}

func (t *Tool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: ToolName,
		Desc: "Execute bash commands with a non-interactive bash (no rc files, not $SHELL). Supports pipes, redirects, and chained commands.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "The full bash command string to execute. Example: \"ls -la | grep .txt\"",
				Required: true,
			},
			"cwd": {
				Type:     schema.String,
				Desc:     "Working directory.",
				Required: false,
			},
			"stdin": {
				Type:     schema.String,
				Desc:     "Text fed to stdin.",
				Required: false,
			},
			"timeout_ms": {
				Type:     schema.Number,
				Desc:     "Timeout in milliseconds for the still-running command process group. Default 10000. Background jobs already reparented after the shell exits are not reaped.",
				Required: false,
			},
			"max_output_kb": {
				Type:     schema.Number,
				Desc:     "Max stdout/stderr bytes captured (KB).",
				Required: false,
			},
			"env": {
				Type:     schema.Object,
				Desc:     "Environment variables map. Example: {\"KEY\":\"VALUE\"}",
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

	paramsMap, err := shared.ParseToolArgs(argumentsInJSON)
	if err != nil {
		return "", err
	}

	extraEnv := map[string]string{}
	if envRaw, ok := paramsMap["env"]; ok && envRaw != nil {
		if rawMap, ok := envRaw.(map[string]interface{}); ok {
			for key, value := range rawMap {
				text, ok := value.(string)
				if !ok || strings.TrimSpace(key) == "" {
					continue
				}
				extraEnv[strings.TrimSpace(key)] = text
			}
		}
	}

	payload, err := t.Execute(ctx, Params{
		Command:     strings.TrimSpace(shared.GetStringParam(paramsMap, "command")),
		CWD:         shared.GetStringParam(paramsMap, "cwd"),
		Stdin:       shared.GetStringParam(paramsMap, "stdin"),
		TimeoutMS:   shared.GetIntParam(paramsMap, "timeout_ms", 0),
		MaxOutputKB: shared.GetIntParam(paramsMap, "max_output_kb", 0),
		Env:         extraEnv,
		BaseDir:     shared.GetStringParam(paramsMap, "base_dir"),
	})
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(encoded), nil
}

func (t *Tool) Execute(ctx context.Context, params Params) (map[string]interface{}, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context canceled before execution: %w", ctx.Err())
	}
	if params.Command == "" {
		return nil, fmt.Errorf("command is required")
	}
	if blockedDomain := t.BlockedProtectedDomainForCommand(params.Command); blockedDomain != "" {
		return nil, fmt.Errorf(
			"目标域名 %s 已命中过 Cloudflare 验证，禁止继续使用直连 HTTP 脚本（curl/wget/requests/httpx）以避免验证失效；请改用 browser 工具继续抓取",
			blockedDomain,
		)
	}

	timeoutMS := resolveCommandTimeoutMS(params.TimeoutMS)

	maxOutputKB := params.MaxOutputKB
	if maxOutputKB <= 0 {
		maxOutputKB = 256
	}
	if maxOutputKB > 1024 {
		maxOutputKB = 1024
	}

	baseDir, err := fsutil.ResolveBaseDir(t.defaultBaseDir, params.BaseDir)
	if err != nil {
		return nil, err
	}
	workDir := baseDir
	if strings.TrimSpace(params.CWD) != "" {
		resolved, err := fsutil.ResolvePathWithin(baseDir, params.CWD)
		if err != nil {
			return nil, fmt.Errorf("invalid cwd: %w", err)
		}
		workDir = resolved
	}

	ctx2, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()
	payload, err := runCommandOnce(ctx2, t.shellPath, params.Command, workDir, params.Stdin, params.Env, maxOutputKB*1024)
	if err != nil {
		return nil, err
	}

	stdoutText, _ := payload["stdout"].(string)
	stderrText, _ := payload["stderr"].(string)
	challengeURL, detected := cloudflare.DetectFromCommandOutput(params.Command, stdoutText, stderrText)
	if detected && t.challengeHandler != nil {
		if strings.TrimSpace(challengeURL) == "" {
			challengeURL = cloudflare.ExtractFirstURL(params.Command)
		}
		if err := t.challengeHandler(ctx, ChallengeRequest{
			ToolName:  ToolName,
			URL:       strings.TrimSpace(challengeURL),
			TimeoutMS: t.challengeTimeoutMS,
		}); err != nil {
			return nil, err
		}

		retryCtx, retryCancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
		defer retryCancel()
		retryPayload, retryErr := runCommandOnce(retryCtx, t.shellPath, params.Command, workDir, params.Stdin, params.Env, maxOutputKB*1024)
		if retryErr != nil {
			return nil, retryErr
		}
		retryPayload["cloudflare_manual_verified"] = true
		if strings.TrimSpace(challengeURL) != "" {
			retryPayload["cloudflare_manual_target_url"] = strings.TrimSpace(challengeURL)
		}
		retryStdout, _ := retryPayload["stdout"].(string)
		retryStderr, _ := retryPayload["stderr"].(string)
		if _, stillBlocked := cloudflare.DetectFromCommandOutput(params.Command, retryStdout, retryStderr); stillBlocked {
			return nil, fmt.Errorf("cloudflare challenge still exists after manual verification, please retry with browser/web_fetch: %s", strings.TrimSpace(challengeURL))
		}
		return retryPayload, nil
	}

	return payload, nil
}

func (t *Tool) BlockedProtectedDomainForCommand(command string) string {
	if t == nil || t.protectedDomains == nil {
		return ""
	}
	if !isLikelyDirectHTTPCommand(command) {
		return ""
	}
	for _, domain := range cloudflare.ExtractHTTPDomainsFromText(command) {
		if t.protectedDomains.Contains(domain) {
			return domain
		}
	}
	return ""
}

func runCommandOnce(
	ctx context.Context,
	shellPath string,
	command string,
	workDir string,
	stdinValue string,
	extraEnv map[string]string,
	maxOutputBytes int,
) (map[string]interface{}, error) {
	execPath, execArgs := buildShellInvocation(shellPath, command)
	cmd := osExec.CommandContext(ctx, execPath, execArgs...)
	cmd.Dir = workDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// Background children inherit the stdout/stderr pipes. Without WaitDelay,
	// cmd.Wait blocks until those children close the pipes (or until timeout
	// kills the whole process group). WaitDelay lets the shell exit while
	// detached jobs keep running.
	cmd.WaitDelay = 200 * time.Millisecond

	if stdinValue != "" {
		cmd.Stdin = strings.NewReader(stdinValue)
	}
	if len(extraEnv) > 0 {
		env := append([]string{}, os.Environ()...)
		for key, value := range extraEnv {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}

	stdoutBuf := shared.NewLimitedBuffer(maxOutputBytes)
	stderrBuf := shared.NewLimitedBuffer(maxOutputBytes)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	if fn := outputListenerFrom(ctx); fn != nil {
		cmd.Stdout = io.MultiWriter(stdoutBuf, liveWriter{stream: "stdout", fn: fn})
		cmd.Stderr = io.MultiWriter(stderrBuf, liveWriter{stream: "stderr", fn: fn})
	}
	start := time.Now()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
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

	elapsedMS := time.Since(start).Milliseconds()
	exitCode := 0
	if runErr != nil {
		if errors.Is(runErr, osExec.ErrWaitDelay) && cmd.ProcessState != nil {
			// Shell already exited; leftover pipe holders are background children.
			exitCode = cmd.ProcessState.ExitCode()
			runErr = nil
		} else if exitErr, ok := runErr.(*osExec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	} else if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	payload := map[string]interface{}{
		"exit_code":     exitCode,
		"stdout":        stdoutBuf.String(),
		"stderr":        stderrBuf.String(),
		"truncated":     stdoutBuf.Truncated() || stderrBuf.Truncated(),
		"elapsed_ms":    elapsedMS,
		"workdir":       workDir,
		"resolved_exec": shellPath,
		"full_command":  command,
		"failed":        runErr != nil || ctx.Err() != nil,
	}
	if runErr != nil {
		payload["error"] = runErr.Error()
	}
	if ctx.Err() != nil {
		payload["error"] = fmt.Sprintf("command timeout or canceled: %v", ctx.Err())
	}
	return payload, nil
}

func resolveShellPath() (string, error) {
	if path, err := bashLookPath("bash"); err == nil && strings.TrimSpace(path) != "" {
		return path, nil
	}
	return "/bin/bash", nil
}

func resolveCommandTimeoutMS(timeoutMS int) int {
	if timeoutMS <= 0 {
		timeoutMS = defaultCommandTimeoutMS
	}
	if timeoutMS > maxCommandTimeoutMS {
		timeoutMS = maxCommandTimeoutMS
	}
	return timeoutMS
}

func buildShellInvocation(shellPath string, command string) (string, []string) {
	shellPath = strings.TrimSpace(shellPath)
	if shellPath == "" {
		shellPath = "/bin/bash"
	}

	switch strings.ToLower(filepath.Base(shellPath)) {
	case "bash", "bash.exe":
		return shellPath, []string{"--noprofile", "--norc", "-c", command}
	case "zsh", "zsh.exe":
		return shellPath, []string{"-f", "-c", command}
	default:
		return shellPath, []string{"-c", command}
	}
}

func isLikelyDirectHTTPCommand(command string) bool {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if normalized == "" {
		return false
	}
	markers := []string{
		"curl ",
		"wget ",
		"requests.get(",
		"requests.post(",
		"requests.request(",
		"httpx.get(",
		"httpx.post(",
		"httpx.request(",
		"urllib.request",
		"invoke-webrequest",
		"invoke-restmethod",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
