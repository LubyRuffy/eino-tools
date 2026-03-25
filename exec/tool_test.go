package exec

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
