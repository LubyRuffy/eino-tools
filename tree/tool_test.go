package tree

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_InvokableRun_ShowsTree(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sub", "b.txt"), []byte("b"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"path":".","max_depth":4,"max_entries":200}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "a.txt")
	assert.Contains(t, out, "sub"+string(os.PathSeparator))
	assert.Contains(t, out, filepath.Join("sub", "b.txt"))
}
