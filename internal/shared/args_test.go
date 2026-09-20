package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupStringParam(t *testing.T) {
	params := map[string]interface{}{
		"empty":  "",
		"text":   "ok",
		"nilval": nil,
		"arr":    []interface{}{"x"},
		"num":    float64(1),
	}

	value, present, err := LookupStringParam(nil, "text")
	require.NoError(t, err)
	assert.False(t, present)
	assert.Equal(t, "", value)

	value, present, err = LookupStringParam(params, "missing")
	require.NoError(t, err)
	assert.False(t, present)
	assert.Equal(t, "", value)

	value, present, err = LookupStringParam(params, "nilval")
	require.NoError(t, err)
	assert.False(t, present)
	assert.Equal(t, "", value)

	value, present, err = LookupStringParam(params, "empty")
	require.NoError(t, err)
	assert.True(t, present)
	assert.Equal(t, "", value)

	value, present, err = LookupStringParam(params, "text")
	require.NoError(t, err)
	assert.True(t, present)
	assert.Equal(t, "ok", value)

	_, present, err = LookupStringParam(params, "arr")
	require.Error(t, err)
	assert.True(t, present)
	assert.Contains(t, err.Error(), "arr must be a string")

	_, present, err = LookupStringParam(params, "num")
	require.Error(t, err)
	assert.True(t, present)
	assert.Contains(t, err.Error(), "num must be a string")
}

func TestGetStringParamStillSwallowsNonString(t *testing.T) {
	params := map[string]interface{}{
		"arr": []interface{}{"x"},
	}
	assert.Equal(t, "", GetStringParam(params, "arr"))
	assert.Equal(t, "", GetStringParam(params, "missing"))
}
