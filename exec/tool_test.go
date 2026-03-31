package exec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
		AllowedPaths:   []string{"~/.codex", "~/.claude"},
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
		AllowedPaths:   []string{"~/.codex", "~/.claude"},
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
	baseHome := t.TempDir()
	t.Setenv("HOME", baseHome)
	zshrcPath := filepath.Join(baseHome, ".zshrc")
	require.NoError(t, os.WriteFile(zshrcPath, []byte("export TEST_ZSHRC_EXEC=1\n"), 0o644))

	gotShell, gotArgs := buildShellInvocation("/bin/zsh", "echo hello")
	assert.Equal(t, "/bin/zsh", gotShell)
	assert.Equal(t, []string{"-lc", "source " + strconv.Quote(zshrcPath) + "; eval " + strconv.Quote("echo hello")}, gotArgs)
}

func TestRunCommandOnce_FallbackToBashWhenShellMissing(t *testing.T) {
	t.Setenv("SHELL", "")
	toolPath, _ := resolveShellPath()
	assert.Equal(t, "/bin/bash", toolPath)

	gotShell, gotArgs := buildShellInvocation("", "echo hello")
	assert.Equal(t, "/bin/bash", gotShell)
	assert.Equal(t, []string{"-lc", "echo hello"}, gotArgs)
}

func TestRunCommandOnce_UsesZshrcInRealExecution(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("skip real zsh execution test: zsh not found")
		}
		require.NoError(t, err)
	}

	t.Setenv("SHELL", "/bin/zsh")
	baseHome := t.TempDir()
	t.Setenv("HOME", baseHome)
	zshrcPath := filepath.Join(baseHome, ".zshrc")
	require.NoError(t, os.WriteFile(zshrcPath, []byte("export TEST_ZSHRC_EXEC=from_zshrc\n"), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := runCommandOnce(ctx, "/bin/zsh", "echo $TEST_ZSHRC_EXEC", "", "", nil, 64*1024)
	require.NoError(t, err)
	stdout, _ := result["stdout"].(string)
	assert.Equal(t, "from_zshrc", strings.TrimSpace(stdout))
}

func TestTool_InvokableRun_UsesAliasFromZshrc(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("skip zsh alias execution test: zsh not found")
		}
		require.NoError(t, err)
	}

	baseHome := t.TempDir()
	t.Setenv("HOME", baseHome)
	t.Setenv("SHELL", "/bin/zsh")
	zshrcPath := filepath.Join(baseHome, ".zshrc")
	require.NoError(t, os.WriteFile(zshrcPath, []byte("alias mysql8='echo mysql8_alias_ok'\n"), 0o644))

	tl, err := New(Config{})
	require.NoError(t, err)

	result, runErr := tl.InvokableRun(context.Background(), `{"command":"mysql8"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, `"exit_code":0`)
	assert.Contains(t, result, "mysql8_alias_ok")
}
