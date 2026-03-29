package mcpserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewToolset_UsesCanonicalToolNamesOnly(t *testing.T) {
	tools, err := NewToolset(context.Background(), Config{
		BaseDir: t.TempDir(),
		Name:    "eino-tools",
		Version: "dev",
	})
	require.NoError(t, err)

	names, err := ToolNames(context.Background(), tools)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"web_search",
		"web_fetch",
		"exec",
		"read",
		"write",
		"edit",
		"ls",
		"tree",
		"glob",
		"grep",
		"python_runner",
		"screenshot",
	}, names)
	require.NotContains(t, names, "fetchurl")
	require.NotContains(t, names, "bashcmd")
}
