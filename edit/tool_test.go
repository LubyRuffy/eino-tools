package edit

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

func TestTool_InvokableRun_SearchReplace(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("Hello World\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	result, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"test.txt","search_block":"Hello World","replace_block":"Hi Universe"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "ok: replaced block")

	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "Hi Universe\n", string(content))
}

func TestTool_InvokableRun_Patch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	patch := `*** Begin Patch\n*** Update File: test.txt\n@@\n line1\n-line2\n+modified line2\n line3\n*** End Patch`
	result, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"test.txt","patch":"`+patch+`"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "ok: patched")
}
