package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalkIgnore_DefaultSkipDirsExactNamesOnly(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "frontend", "dist")
	distributed := filepath.Join(root, "src", "distributed")
	worktrees := filepath.Join(root, ".worktrees", "issue-x")
	require.NoError(t, os.MkdirAll(dist, 0o755))
	require.NoError(t, os.MkdirAll(distributed, 0o755))
	require.NoError(t, os.MkdirAll(worktrees, 0o755))

	ignore := LoadWalkIgnore(root)
	assert.False(t, ignore.SkipDir(root, filepath.Base(root)))
	assert.True(t, ignore.SkipDir(dist, "dist"))
	assert.True(t, ignore.SkipDir(filepath.Join(root, ".worktrees"), ".worktrees"))
	assert.False(t, ignore.SkipDir(distributed, "distributed"))
	assert.True(t, ignore.SkipFile(filepath.Join(dist, "bundle.js")))
	assert.False(t, ignore.SkipFile(filepath.Join(distributed, "app.go")))
}

func TestWalkIgnore_ExplicitRootIsNeverSkipped(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "dist")
	require.NoError(t, os.MkdirAll(dist, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dist, "bundle.js"), []byte("x"), 0o644))

	ignore := LoadWalkIgnore(dist)
	assert.False(t, ignore.SkipDir(dist, "dist"))
	assert.False(t, ignore.SkipFile(filepath.Join(dist, "bundle.js")))
}

func TestWalkIgnore_GitignoreDirPatternAndNegation(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "hidden"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("hidden/\n*.tmp\n!keep.tmp\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "hidden", "secret.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "secret.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "drop.tmp"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "keep.tmp"), []byte("x"), 0o644))

	ignore := LoadWalkIgnore(root)
	assert.True(t, ignore.SkipDir(filepath.Join(root, "hidden"), "hidden"))
	assert.True(t, ignore.SkipFile(filepath.Join(root, "hidden", "secret.txt")))
	assert.False(t, ignore.SkipFile(filepath.Join(root, "src", "secret.txt")))
	assert.True(t, ignore.SkipFile(filepath.Join(root, "src", "drop.tmp")))
	assert.False(t, ignore.SkipFile(filepath.Join(root, "src", "keep.tmp")))
}

func TestWalkIgnore_NilReceiverAndGlobVariants(t *testing.T) {
	var ignore *WalkIgnore
	assert.True(t, ignore.SkipDir("/tmp/dist", "dist"))
	assert.False(t, ignore.SkipDir("/tmp/src", "src"))
	assert.False(t, ignore.SkipFile("/tmp/src/a.go"))

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("# comment\n\n/\n**/*.cache\nfile?.txt\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "foo.cache"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "fileA.txt"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src", "fileAB.txt"), []byte("x"), 0o644))

	loaded := LoadWalkIgnore(root)
	assert.False(t, loaded.SkipFile(root))
	assert.True(t, loaded.SkipFile(filepath.Join(root, "src", "foo.cache")))
	assert.True(t, loaded.SkipFile(filepath.Join(root, "src", "fileA.txt")))
	assert.False(t, loaded.SkipFile(filepath.Join(root, "src", "fileAB.txt")))
}

func TestWalkIgnore_DirOnlyPatternDoesNotMatchFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("logs/\nfoo**bar\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "logs"), []byte("not a dir\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "fooxbar"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "keep.txt"), []byte("x"), 0o644))

	ignore := LoadWalkIgnore(root)
	assert.False(t, ignore.SkipFile(filepath.Join(root, "logs")))
	assert.True(t, ignore.SkipFile(filepath.Join(root, "fooxbar")))
	assert.False(t, ignore.SkipFile(filepath.Join(root, "keep.txt")))
	assert.False(t, ignore.SkipFile(filepath.Join(t.TempDir(), "outside.txt")))
}
