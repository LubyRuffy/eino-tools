package write

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fixtureName    = "notes.txt"
	fixtureContent = "alpha\n"
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
	tl := newTool(t, tmpDir)

	result := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": "nested/demo.txt",
		"content":   "hello",
	}))
	assert.Contains(t, result, "Updated file")
	assert.NotContains(t, result, "error:")

	content, err := os.ReadFile(filepath.Join(tmpDir, "nested", "demo.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))
}

func TestTool_InvokableRun_WritesFileOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	targetPath := filepath.Join(outsideDir, "demo.txt")
	tl := newTool(t, baseDir)

	result := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": targetPath,
		"content":   "hello outside",
	}))
	assert.Contains(t, result, "Updated file")
	assert.NotContains(t, result, "error:")

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	assert.Equal(t, "hello outside", string(content))
}

func TestTool_InvokableRun_OmittedContentDoesNotEmptyFile(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{"file_path": fixtureName}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "content")
	assert.Contains(t, out, "file_path")
	assert.NotContains(t, out, "must be a string")
	assert.NotContains(t, out, "use content not contents")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestTool_InvokableRun_ContentsKeyIsNotContent(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"contents":  "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "content")
	assert.Contains(t, out, "contents")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestTool_InvokableRun_NonStringContentIsTypeError(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"content":   []any{"beta"},
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "must be a string")
	assert.Contains(t, out, "content")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestTool_InvokableRun_EmptyContentWritesEmptyFile(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"content":   "",
	}))
	assert.Contains(t, out, "Updated file")
	assert.NotContains(t, out, "error:")
	assert.Equal(t, "", fileContent(t, absPath))
}

func TestTool_InvokableRun_SuccessRereadsExactBytes(t *testing.T) {
	tmpDir := t.TempDir()
	tl := newTool(t, tmpDir)
	body := "line 1\nline \"2\"\n"
	absPath := filepath.Join(tmpDir, fixtureName)

	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"content":   body,
	}))
	assert.Contains(t, out, "Updated file")
	assert.NotContains(t, out, "error:")
	got, err := os.ReadFile(absPath)
	require.NoError(t, err)
	assert.Equal(t, []byte(body), got)
	assert.True(t, strings.Contains(out, absPath) || strings.Contains(out, "bytes"),
		"success must include resolved path or byte count, got %q", out)
}

func TestTool_InvokableRun_RelativePathUsesBaseDirNotCwd(t *testing.T) {
	baseDir := t.TempDir()
	cwd := t.TempDir()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(orig))
	})

	tl := newTool(t, baseDir)
	body := "beta\n"
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"content":   body,
	}))
	assert.Contains(t, out, "Updated file")
	assert.NotContains(t, out, "error:")

	absPath := filepath.Join(baseDir, fixtureName)
	assert.Equal(t, body, fileContent(t, absPath))
	_, cwdErr := os.Stat(filepath.Join(cwd, fixtureName))
	assert.Error(t, cwdErr)
	assert.Contains(t, out, absPath)
	assert.NotContains(t, out, filepath.Join(cwd, fixtureName))
}

func TestTool_InvokableRun_MissingFilePathFails(t *testing.T) {
	tmpDir := t.TempDir()
	tl := newTool(t, tmpDir)
	out := runWrite(t, tl, writeJSON(t, map[string]any{"content": "beta\n"}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "file_path")
	assert.NotContains(t, out, "Updated file")
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestTool_InvokableRun_PathAliasHintsFilePath(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"path":    fixtureName,
		"content": "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "file_path")
	assert.Contains(t, out, "use file_path not path")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestConfirmBytesRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, fixtureName)
	require.NoError(t, os.WriteFile(path, []byte(fixtureContent), 0o644))
	err := confirmBytes(path, []byte("beta\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do not match")
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestConfirmBytesMissingFile(t *testing.T) {
	err := confirmBytes(filepath.Join(t.TempDir(), fixtureName), []byte("beta\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify written file")
}

func TestTool_InvokableRun_EmptyFilePathFails(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": "",
		"path":      fixtureName,
		"content":   "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "file_path")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestTool_InvokableRun_NonStringFilePathIsTypeError(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": []any{fixtureName},
		"content":   "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "must be a string")
	assert.Contains(t, out, "file_path")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestTool_InvokableRun_InvalidJSONFails(t *testing.T) {
	tl := newTool(t, t.TempDir())
	out := runWrite(t, tl, `{`)
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "failed to parse arguments")
	assert.NotContains(t, out, "Updated file")
}

func TestTool_InvokableRun_InvalidBaseDirFails(t *testing.T) {
	tl := newTool(t, t.TempDir())
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"content":   "beta\n",
		"base_dir":  filepath.Join(t.TempDir(), "missing-base"),
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "base_dir")
	assert.NotContains(t, out, "Updated file")
}

func TestTool_InvokableRun_DirectoryTargetFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, fixtureName)
	require.NoError(t, os.Mkdir(target, 0o755))
	tl := newTool(t, dir)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"content":   "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "failed to write file")
	assert.NotContains(t, out, "Updated file")
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestTool_InvokableRun_ParentFileBlocksWrite(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Join(dir, "blocked")
	require.NoError(t, os.WriteFile(parent, []byte(fixtureContent), 0o644))
	tl := newTool(t, dir)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": "blocked/child.txt",
		"content":   "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, parent))
}

func TestTool_InvokableRun_UnknownBodyKeyDoesNotSuggestContents(t *testing.T) {
	tl, absPath := setupFixture(t)
	out := runWrite(t, tl, writeJSON(t, map[string]any{
		"file_path": fixtureName,
		"body":      "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "content")
	assert.Contains(t, out, "body")
	assert.NotContains(t, out, "contents")
	assert.NotContains(t, out, "Updated file")
	assert.Equal(t, fixtureContent, fileContent(t, absPath))
}

func TestTool_InfoDescribesRequiredWriteContract(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	info, err := tl.Info(context.Background())
	require.NoError(t, err)

	blob := info.Desc
	if info.ParamsOneOf != nil {
		params, paramsErr := info.ParamsOneOf.ToJSONSchema()
		require.NoError(t, paramsErr)
		encoded, marshalErr := json.Marshal(params)
		require.NoError(t, marshalErr)
		blob += string(encoded)
	}

	assert.Contains(t, blob, "file_path")
	assert.Contains(t, blob, "content")
	assert.Contains(t, blob, "contents")
	assert.True(t, strings.Contains(blob, "required") || strings.Contains(blob, "必填"))
	assert.True(t,
		strings.Contains(strings.ToLower(blob), "success") ||
			strings.Contains(blob, "可读") ||
			strings.Contains(strings.ToLower(blob), "read"),
		"desc must say success means bytes are readable back, got %q", blob)
	for _, leak := range []string{fixtureName, "alpha", "beta", "Datakanban", "zwai", "eino-swarm"} {
		assert.NotContains(t, blob, leak)
	}
}

func newTool(t *testing.T, baseDir string) *Tool {
	t.Helper()
	tl, err := New(Config{DefaultBaseDir: baseDir})
	require.NoError(t, err)
	return tl
}

func setupFixture(t *testing.T) (*Tool, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, fixtureName)
	require.NoError(t, os.WriteFile(path, []byte(fixtureContent), 0o644))
	return newTool(t, dir), path
}

func fileContent(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func writeJSON(t *testing.T, v map[string]any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func runWrite(t *testing.T, tl *Tool, args string) string {
	t.Helper()
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)
	out, err := invokable.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	return out
}
