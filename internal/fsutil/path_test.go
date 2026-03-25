package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPathWithin(t *testing.T) {
	tests := []struct {
		name     string
		baseDir  string
		target   string
		expected bool
	}{
		{name: "same path", baseDir: "/home/user", target: "/home/user", expected: true},
		{name: "target inside base", baseDir: "/home/user", target: "/home/user/docs", expected: true},
		{name: "target outside base", baseDir: "/home/user", target: "/home/other", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsPathWithin(tt.baseDir, tt.target))
		})
	}
}

func TestResolvePathWithin_AllowsConfiguredPaths(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	baseDir := t.TempDir()
	allowedPaths := []string{"~/.codex", "~/.claude"}

	pathValue, err := ResolvePathWithin(baseDir, filepath.Join(homeDir, ".codex", "skills"), allowedPaths)
	require.NoError(t, err)
	assert.NotEmpty(t, pathValue)
}

func TestResolvePathWithin_BlocksPathsOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()

	_, err := ResolvePathWithin(baseDir, "/tmp/random", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outside base_dir")
}

func TestResolveBaseDir_UsesOverride(t *testing.T) {
	defaultDir := t.TempDir()
	override := t.TempDir()

	baseDir, err := ResolveBaseDir(defaultDir, override)
	require.NoError(t, err)
	assert.Equal(t, override, baseDir)
}
