package websearch

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSearchRunner struct {
	calls int
	out   string
	err   error
}

func (f *fakeSearchRunner) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	f.calls++
	return f.out, f.err
}

type fakeCache struct {
	values map[string]string
}

func (f *fakeCache) Get(query string) (string, bool, error) {
	if f.values == nil {
		return "", false, nil
	}
	value, ok := f.values[query]
	return value, ok, nil
}

func (f *fakeCache) Set(query string, result string) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[query] = result
	return nil
}

func TestNewTool_InfoName(t *testing.T) {
	tl, err := New(context.Background(), Config{
		SearchRunner: &fakeSearchRunner{out: "ok"},
	})
	require.NoError(t, err)

	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ToolName, info.Name)
}

func TestTool_Search_EmptyQuery(t *testing.T) {
	tl, err := New(context.Background(), Config{
		SearchRunner: &fakeSearchRunner{out: "ok"},
	})
	require.NoError(t, err)

	_, searchErr := tl.Search(context.Background(), "")
	require.Error(t, searchErr)
	assert.Contains(t, searchErr.Error(), "query is required")
}

func TestTool_Search_UsesCache(t *testing.T) {
	cache := &fakeCache{
		values: map[string]string{
			"golang": "cached",
		},
	}
	runner := &fakeSearchRunner{out: "network"}
	tl, err := New(context.Background(), Config{
		Cache:        cache,
		SearchRunner: runner,
	})
	require.NoError(t, err)

	result, searchErr := tl.Search(context.Background(), "golang")
	require.NoError(t, searchErr)
	assert.Equal(t, "cached", result)
	assert.Equal(t, 0, runner.calls)
}
