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

func TestResolvePathWithin_AllowsAbsolutePathOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "random.txt")

	pathValue, err := ResolvePathWithin(baseDir, outsidePath, nil)
	require.NoError(t, err)
	assert.Equal(t, outsidePath, pathValue)
}

func TestResolvePathWithin_AllowsRelativeTraversalOutsideBaseDir(t *testing.T) {
	parentDir := t.TempDir()
	baseDir := filepath.Join(parentDir, "base")
	require.NoError(t, os.MkdirAll(baseDir, 0o755))

	pathValue, err := ResolvePathWithin(baseDir, "../target.txt", nil)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(parentDir, "target.txt"), pathValue)
}

func TestResolveBaseDir_UsesOverride(t *testing.T) {
	defaultDir := t.TempDir()
	override := t.TempDir()

	baseDir, err := ResolveBaseDir(defaultDir, override)
	require.NoError(t, err)
	assert.Equal(t, override, baseDir)
}
