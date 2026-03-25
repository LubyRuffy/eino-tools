package editutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeNewlines(t *testing.T) {
	assert.Equal(t, "a\nb\n", NormalizeNewlines("a\r\nb\r"))
}

func TestApplyReplaceBlockOnce(t *testing.T) {
	next, count := ApplyReplaceBlockOnce("hello world", "world", "codex")
	assert.Equal(t, "hello codex", next)
	assert.Equal(t, 1, count)
}

func TestParseApplyPatchText(t *testing.T) {
	patch := "*** Begin Patch\n*** Update File: demo.txt\n@@\n line1\n-line2\n+line2x\n line3\n*** End Patch\n"

	replacements, err := ParseApplyPatchText(patch, "demo.txt")
	require.NoError(t, err)
	require.Len(t, replacements, 1)
	assert.Contains(t, replacements[0].Search, "line2")
	assert.Contains(t, replacements[0].Replace, "line2x")
}
