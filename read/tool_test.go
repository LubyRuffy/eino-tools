package read

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

func TestTool_InvokableRun_ReadsPagedContent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "x.txt"), []byte("l1\nl2\nl3\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"x.txt","offset":2,"limit":2}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "encoding=")
	assert.Contains(t, out, "2|l2")
	assert.Contains(t, out, "3|l3")
}

func TestTool_InvokableRun_ReadsFileOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	targetPath := filepath.Join(outsideDir, "x.txt")
	require.NoError(t, os.WriteFile(targetPath, []byte("external"), 0o644))

	tl, err := New(Config{DefaultBaseDir: baseDir})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"`+targetPath+`"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "external")
}
