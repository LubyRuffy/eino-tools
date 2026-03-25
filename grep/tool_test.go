package grep

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_InvokableRun_FilesWithMatches(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "sub", "a.txt"), []byte("hello\ngrep me\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"path":"sub","pattern":"grep","output_mode":"files_with_matches"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, filepath.Join("sub", "a.txt"))
}

func TestTool_InvokableRun_Count(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("x\nx\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"path":"a.txt","pattern":"x","output_mode":"count"}`)
	require.NoError(t, runErr)
	assert.Equal(t, "2", out)
}
