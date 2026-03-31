package write

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

func TestTool_InvokableRun_WritesFile(t *testing.T) {
	tmpDir := t.TempDir()
	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	result, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"nested/demo.txt","content":"hello"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "Updated file nested/demo.txt")

	content, err := os.ReadFile(filepath.Join(tmpDir, "nested", "demo.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestTool_InvokableRun_WritesFileOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	targetPath := filepath.Join(outsideDir, "demo.txt")

	tl, err := New(Config{DefaultBaseDir: baseDir})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	result, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"`+targetPath+`","content":"hello outside"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "Updated file")

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "hello outside", string(content))
}
