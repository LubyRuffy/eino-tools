package screenshotutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRegion(t *testing.T) {
	region, err := ParseRegion("10,20,300,180")
	require.NoError(t, err)
	require.NotNil(t, region)
	assert.Equal(t, "10,20,300,180", region.String())
}

func TestParseRegion_InvalidFormat(t *testing.T) {
	_, err := ParseRegion("10,20,30")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid region format")
}

func TestNormalizeOutputPath_AppendsPNGExt(t *testing.T) {
	pathValue, err := NormalizeOutputPath("shots/demo")
	require.NoError(t, err)
	assert.Equal(t, "shots/demo.png", pathValue)
}
