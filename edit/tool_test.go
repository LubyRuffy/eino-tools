package edit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureContent = "alpha\nbeta\ngamma\n"
const catchAllErr = "either search_block/replace_block or patch is required"

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

func setupFixture(t *testing.T) (*Tool, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(path, []byte(fixtureContent), 0o644))
	tl, err := New(Config{DefaultBaseDir: dir})
	require.NoError(t, err)
	return tl, path
}

func fileContent(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func editJSON(t *testing.T, v map[string]any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func runEdit(t *testing.T, tl *Tool, args string) string {
	t.Helper()
	out, err := tl.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	return out
}

func TestTool_InvokableRun_MissingPayloadListsRequiredFields(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{"file_path": "x.txt"}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "search_block")
	assert.Contains(t, out, "replace_block")
	assert.Contains(t, out, "patch")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_OnlyReplaceRequiresSearch(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"replace_block": "beta2\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "search_block")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_OnlySearchRequiresReplace(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":    "x.txt",
		"search_block": "beta\n",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "replace_block")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_EmptyReplaceDeletes(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"search_block":  "beta\n",
		"replace_block": "",
	}))
	assert.Contains(t, out, "ok: replaced block")
	assert.Equal(t, "alpha\ngamma\n", fileContent(t, path))
}

func TestTool_InvokableRun_PairedSearchReplaceOnFixture(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"search_block":  "beta",
		"replace_block": "beta2",
	}))
	assert.Contains(t, out, "ok: replaced block")
	got := fileContent(t, path)
	assert.Contains(t, got, "beta2")
	assert.NotContains(t, got, "beta\n")
}

func TestTool_InvokableRun_NonStringBlocksAreTypeErrors(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"search_block":  []any{"beta"},
		"replace_block": []any{"beta2"},
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "must be a string")
	assert.Contains(t, out, "search_block")
	assert.Contains(t, out, "replace_block")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_UnknownFieldNamesListReceivedKeys(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":  "x.txt",
		"old_string": "beta",
		"new_string": "beta2",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "search_block")
	assert.Contains(t, out, "replace_block")
	assert.Contains(t, out, "patch")
	assert.Contains(t, out, "old_string")
	assert.Contains(t, out, "new_string")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_SearchNotFoundIsSpecific(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"search_block":  "missing-line",
		"replace_block": "other",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "search_block not found in file")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_EmptySearchIsRejected(t *testing.T) {
	tl, path := setupFixture(t)
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"search_block":  "",
		"replace_block": "beta2",
	}))
	assert.Contains(t, out, "error:")
	assert.Contains(t, out, "search_block must be non-empty")
	assert.NotContains(t, out, catchAllErr)
	assert.Equal(t, fixtureContent, fileContent(t, path))
}

func TestTool_InvokableRun_SearchReplaceWinsOverPatch(t *testing.T) {
	tl, path := setupFixture(t)
	patch := "*** Begin Patch\n*** Update File: x.txt\n@@\n alpha\n beta\n-gamma\n+gamma2\n*** End Patch"
	out := runEdit(t, tl, editJSON(t, map[string]any{
		"file_path":     "x.txt",
		"search_block":  "beta",
		"replace_block": "beta2",
		"patch":         patch,
	}))
	assert.Contains(t, out, "ok: replaced block")
	got := fileContent(t, path)
	assert.Contains(t, got, "beta2")
	assert.Contains(t, got, "gamma")
	assert.NotContains(t, got, "gamma2")
}

func TestTool_InfoDescribesPairedModes(t *testing.T) {
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

	assert.Contains(t, blob, "search_block")
	assert.Contains(t, blob, "replace_block")
	assert.Contains(t, blob, "patch")
	assert.Contains(t, blob, "empty")
	assert.NotContains(t, blob, "FEATURES.md")
	assert.NotContains(t, blob, "x.txt")
	assert.NotContains(t, blob, "old_string")
}
