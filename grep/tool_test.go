package grep

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGrep(t *testing.T, baseDir, args string) (string, error) {
	t.Helper()
	tl, err := New(Config{DefaultBaseDir: baseDir})
	require.NoError(t, err)
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)
	return invokable.InvokableRun(context.Background(), args)
}

func TestTool_Info(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ToolName, info.Name)
	assert.Contains(t, info.Desc, "Skips")
}

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

func TestTool_InvokableRun_LongLineDoesNotAbortWalk(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("needle in source\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "blob.txt"), []byte(strings.Repeat("x", 70000)+"\n"), 0o644))

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, filepath.Join("src", "main.go"))
	assert.NotContains(t, out, "token too long")
}

func TestTool_InvokableRun_SkipsDefaultBuildDirs(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src", "distributed"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "frontend", "dist"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".worktrees", "issue-x", "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte("needle in source\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "distributed", "app.go"), []byte("needle in distributed pkg\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "frontend", "dist", "bundle.js"), []byte("needle in dist bundle\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".worktrees", "issue-x", "src", "copy.go"), []byte("needle in worktree copy\n"), 0o644))

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, filepath.Join("src", "main.go"))
	assert.Contains(t, out, filepath.Join("src", "distributed", "app.go"))
	assert.NotContains(t, out, "bundle.js")
	assert.NotContains(t, out, "copy.go")
}

func TestTool_InvokableRun_ExplicitPathStillSearchesSkipNamedDir(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "frontend", "dist")
	require.NoError(t, os.MkdirAll(distDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "bundle.js"), []byte("needle in dist bundle\n"), 0o644))

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle","path":"frontend/dist"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "bundle.js")
}

func TestTool_InvokableRun_RespectsGitignoreAndKeepsUnignoredTwin(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "hidden"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("hidden/\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "hidden", "secret.txt"), []byte("needle in ignored dir\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "src", "secret.txt"), []byte("needle in source twin\n"), 0o644))

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, filepath.Join("src", "secret.txt"))
	assert.NotContains(t, out, filepath.Join("hidden", "secret.txt"))
}

func TestTool_InvokableRun_GlobSkipsUnreadableInsteadOfAbort(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("needle here\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "blob.txt"), []byte(strings.Repeat("x", 70000)+"\n"), 0o644))

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle","glob":"*.txt"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "keep.txt")
}

func TestTool_InvokableRun_CapsReportedMatches(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < DefaultMaxMatches+20; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "hit-"+strconv.Itoa(i)+".txt"), []byte("needle\n"), 0o644))
	}

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, TruncatedSuffix)
	files := 0
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" || line == TruncatedSuffix {
			continue
		}
		files++
	}
	assert.Equal(t, DefaultMaxMatches, files)
}

func TestTool_InvokableRun_ContentModeAndBinarySkip(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "hit.txt"), []byte("needle here\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "blob.bin"), append([]byte{0x00, 0x01}, []byte("needle")...), 0o644))

	out, runErr := runGrep(t, tmpDir, `{"pattern":"needle","output_mode":"content"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "hit.txt:1:needle here")
	assert.NotContains(t, out, "blob.bin")
}

func TestFormatGrepMatches_ContentCapAndLongLine(t *testing.T) {
	long := strings.Repeat("x", maxReportedLine+8)
	out := formatGrepMatches("/base", "content", []grepMatch{{
		Path:    "/base/a.txt",
		Line:    1,
		Content: long,
	}}, false)
	assert.Contains(t, out, "a.txt:1:")
	assert.Contains(t, out, "…")

	many := make([]grepMatch, DefaultMaxMatches+3)
	for i := range many {
		many[i] = grepMatch{Path: "/base/a.txt", Line: i + 1, Content: "n"}
	}
	out = formatGrepMatches("/base", "content", many, false)
	assert.Contains(t, out, TruncatedSuffix)
	assert.Equal(t, strconv.Itoa(len(many)), formatGrepMatches("/base", "count", many, false))
	assert.Equal(t, TruncatedSuffix, formatGrepMatches("/base", "files_with_matches", nil, true))
}

func TestSearchFile_SkipsUnreadableAndOversize(t *testing.T) {
	tmpDir := t.TempDir()
	blocked := filepath.Join(tmpDir, "blocked.txt")
	require.NoError(t, os.WriteFile(blocked, []byte("needle\n"), 0o644))
	require.NoError(t, os.Chmod(blocked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o644) })

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)
	re := regexp.MustCompile("needle")
	assert.Empty(t, tl.searchFile(blocked, re))
	assert.Empty(t, tl.searchFile(tmpDir, re))

	big := filepath.Join(tmpDir, "big.txt")
	require.NoError(t, os.WriteFile(big, make([]byte, maxGrepFileBytes+1), 0o644))
	assert.Empty(t, tl.searchFile(big, re))

	empty := filepath.Join(tmpDir, "empty.txt")
	require.NoError(t, os.WriteFile(empty, nil, 0o644))
	assert.Empty(t, tl.searchFile(empty, re))
}

func TestTool_InvokableRun_InvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()
	out, err := runGrep(t, tmpDir, `{"pattern":"("}`)
	require.NoError(t, err)
	assert.Contains(t, out, "invalid regex pattern")
}
