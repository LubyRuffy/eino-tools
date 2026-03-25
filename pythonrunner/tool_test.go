package pythonrunner

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTool_InfoName(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ToolName, info.Name)
}

func TestTool_InvokableRun_EmptyCode(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	result, runErr := invokable.InvokableRun(context.Background(), `{"code":"  "}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, "code is required")
}

func TestTool_InvokableRun_UsesInjectedRunner(t *testing.T) {
	callCount := 0
	tl, err := New(Config{
		PythonResolver: func() (string, error) { return "/usr/bin/python3", nil },
		TempDirFactory: func() (string, error) { return t.TempDir(), nil },
		CommandRunner: func(ctx context.Context, maxOutputBytes int, dir string, execPath string, args []string, extraEnv []string, stdin string) (string, string, int, error) {
			callCount++
			if callCount == 1 {
				return "", "", 0, nil
			}
			return "done", "", 0, nil
		},
	})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	result, runErr := invokable.InvokableRun(context.Background(), `{"code":"print('ok')"}`)
	require.NoError(t, runErr)
	assert.Contains(t, result, `"stdout":"done"`)
}
