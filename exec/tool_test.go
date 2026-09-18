package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/LubyRuffy/eino-tools/internal/cloudflare"
	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_InfoName(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ToolName, info.Name)
}

func TestTool_InvokableRun_ValidCommandWithSpace(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"echo hello world"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "hello world")
	assert.Contains(t, result, `"exit_code":0`)
}

func TestTool_InvokableRun_ResolvesCWDWithinBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	workDir := filepath.Join(baseDir, "sub")
	require.NoError(t, os.MkdirAll(workDir, 0o755))

	tl, err := New(Config{
		DefaultBaseDir: baseDir,
	})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"pwd","cwd":"sub"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, workDir)
}

func TestTool_InvokableRun_AllowsCWDOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	workDir := t.TempDir()

	tl, err := New(Config{
		DefaultBaseDir: baseDir,
	})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"pwd","cwd":"`+workDir+`"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, workDir)
}

func TestTool_InvokableRun_BlocksProtectedDomain(t *testing.T) {
	store := cloudflare.NewProtectedDomains(0)
	store.Mark("https://www.dogster.com/")

	tl, err := New(Config{ProtectedDomains: store})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"curl -sS https://www.dogster.com/"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "禁止继续使用直连 HTTP 脚本")
	assert.Contains(t, result, "www.dogster.com")
	assert.Contains(t, result, "browser 工具")
}

func TestTool_ImplementsBaseTool(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	var baseTool tool.BaseTool = tl
	require.NotNil(t, baseTool)
}

func TestRunCommandOnce_UsesEnvShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	gotShell, gotArgs := buildShellInvocation("/bin/zsh", "echo hello")
	assert.Equal(t, "/bin/zsh", gotShell)
	assert.Equal(t, []string{"-f", "-c", "echo hello"}, gotArgs)
}

func TestRunCommandOnce_FallbackToBashWhenShellMissing(t *testing.T) {
	t.Setenv("SHELL", "")
	toolPath, err := resolveShellPath()
	require.NoError(t, err)
	assert.Equal(t, "bash", strings.ToLower(filepath.Base(toolPath)))

	gotShell, gotArgs := buildShellInvocation("", "echo hello")
	assert.Equal(t, "/bin/bash", gotShell)
	assert.Equal(t, []string{"--noprofile", "--norc", "-c", "echo hello"}, gotArgs)
}

func TestRunCommandOnce_UsesZshrcInRealExecution(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("skip real zsh execution test: zsh not found")
		}
		require.NoError(t, err)
	}

	baseHome := t.TempDir()
	t.Setenv("HOME", baseHome)
	zshrcPath := filepath.Join(baseHome, ".zshrc")
	require.NoError(t, os.WriteFile(zshrcPath, []byte("export TEST_ZSHRC_EXEC=from_zshrc\n"), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runCommandOnce(ctx, "/bin/zsh", "echo ${TEST_ZSHRC_EXEC:-unset}", "", "", nil, 64*1024)
	require.NoError(t, err)
	stdout, _ := result["stdout"].(string)
	assert.Equal(t, "unset", strings.TrimSpace(stdout))
}

func TestTool_InvokableRun_UsesAliasFromZshrc(t *testing.T) {
	baseHome := t.TempDir()
	t.Setenv("HOME", baseHome)
	t.Setenv("SHELL", "/bin/zsh")
	zshrcPath := filepath.Join(baseHome, ".zshrc")
	require.NoError(t, os.WriteFile(zshrcPath, []byte("alias mysql8='echo mysql8_alias_ok'\n"), 0o644))

	tl, err := New(Config{})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"mysql8"}`)
	require.NoError(t, runErr)
	assert.NotContains(t, result, "mysql8_alias_ok")
}

func TestResolveShellPath_IgnoresUserInteractiveShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	path, err := resolveShellPath()
	require.NoError(t, err)
	assert.Equal(t, "bash", strings.ToLower(filepath.Base(path)))
}

func TestResolveCommandTimeoutMS_DependsOnlyOnTimeout(t *testing.T) {
	assert.Equal(t, defaultCommandTimeoutMS, resolveCommandTimeoutMS(0))
	assert.Equal(t, 1500, resolveCommandTimeoutMS(1500))
	assert.Equal(t, maxCommandTimeoutMS, resolveCommandTimeoutMS(maxCommandTimeoutMS+1))
}

func TestExecuteSource_DoesNotSpecialCaseCommandTextForTimeout(t *testing.T) {
	src, err := os.ReadFile("tool.go")
	require.NoError(t, err)
	assert.NotContains(t, string(src), "strings.Contains(params.Command")
}

func TestBuildShellInvocation_Variants(t *testing.T) {
	shell, args := buildShellInvocation("/bin/bash.exe", "echo hi")
	assert.Equal(t, "/bin/bash.exe", shell)
	assert.Equal(t, []string{"--noprofile", "--norc", "-c", "echo hi"}, args)

	shell, args = buildShellInvocation("/bin/zsh.exe", "echo hi")
	assert.Equal(t, "/bin/zsh.exe", shell)
	assert.Equal(t, []string{"-f", "-c", "echo hi"}, args)

	shell, args = buildShellInvocation("/bin/dash", "echo hi")
	assert.Equal(t, "/bin/dash", shell)
	assert.Equal(t, []string{"-c", "echo hi"}, args)
}

func TestResolveShellPath_FallsBackWhenLookPathFails(t *testing.T) {
	orig := bashLookPath
	t.Cleanup(func() { bashLookPath = orig })
	bashLookPath = func(string) (string, error) { return "", errors.New("missing") }
	path, err := resolveShellPath()
	require.NoError(t, err)
	assert.Equal(t, "/bin/bash", path)
}

func TestNew_DoesNotSourceRcOrFollowShellEnv(t *testing.T) {
	baseHome := t.TempDir()
	t.Setenv("HOME", baseHome)
	t.Setenv("SHELL", "/bin/zsh")
	require.NoError(t, os.WriteFile(filepath.Join(baseHome, ".zshrc"), []byte("alias mysql8='echo mysql8_alias_ok'\n"), 0o644))

	tl, err := New(Config{})
	require.NoError(t, err)
	assert.Equal(t, "bash", strings.ToLower(filepath.Base(tl.shellPath)))

	payload, execErr := tl.Execute(context.Background(), Params{
		Command: "mysql8",
	})
	require.NoError(t, execErr)
	assert.NotEqual(t, 0, payload["exit_code"])
	stdout, _ := payload["stdout"].(string)
	assert.NotContains(t, stdout, "mysql8_alias_ok")
}

func TestTool_InvokableRun_SupportsPipes(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"printf 'a\\nb\\n' | wc -l"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, `"exit_code":0`)
	assert.Regexp(t, `"stdout":"[^"]*2`, result)
}

func TestTool_InvokableRun_PreservesPythonDashCNewlines(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("skip python -c test: python3 not found")
		}
		require.NoError(t, err)
	}

	tl, err := New(Config{})
	require.NoError(t, err)

	payload, execErr := tl.Execute(context.Background(), Params{
		Command: "python3 -c 'print(1)\nprint(2)'",
	})
	require.NoError(t, execErr)
	assert.Equal(t, 0, payload["exit_code"])
	assert.Equal(t, "1\n2\n", payload["stdout"])
}

func TestTool_InvokableRun_UnmatchedGlobStaysLiteral(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"ls unmatched_glob_should_stay_literal_*.xyz"}`)
	require.NoError(t, runErr)
	assert.NotContains(t, result, "no matches found")
}

func TestTool_InvokableRun_BackgroundJobSurvivesAfterShellExits(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	payload, execErr := tl.Execute(context.Background(), Params{
		Command:   "sleep 30 & echo $!",
		TimeoutMS: 1500,
	})
	require.NoError(t, execErr)
	require.Equal(t, 0, payload["exit_code"])
	pidStr := strings.TrimSpace(payload["stdout"].(string))
	pid, convErr := strconv.Atoi(pidStr)
	require.NoError(t, convErr)
	t.Cleanup(func() {
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Kill()
		}
	})

	proc, err := os.FindProcess(pid)
	require.NoError(t, err)
	require.NoError(t, proc.Signal(syscall.Signal(0)))
}

func TestTool_InvokableRun_TimeoutKillsForegroundProcess(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	start := time.Now()
	payload, execErr := tl.Execute(context.Background(), Params{
		Command:   "sleep 30",
		TimeoutMS: 800,
	})
	require.NoError(t, execErr)
	assert.True(t, time.Since(start) < 5*time.Second)
	assert.Equal(t, true, payload["failed"])
}
