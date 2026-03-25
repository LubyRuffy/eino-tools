package screenshot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/LubyRuffy/eino-tools/internal/screenshotutil"
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

func TestTool_InvokableRun_Success(t *testing.T) {
	tmpDir := t.TempDir()
	tl, err := New(Config{
		DefaultBaseDir: tmpDir,
		CommandBuilder: func(outputPath string, region *screenshotutil.Region) (*Command, error) {
			require.NotNil(t, region)
			assert.Equal(t, "10,20,300,180", region.String())
			return &Command{Name: "fakecap", Args: []string{outputPath}}, nil
		},
		CommandRunner: func(ctx context.Context, name string, args ...string) error {
			require.Equal(t, "fakecap", name)
			return os.WriteFile(args[len(args)-1], tinyPNGBytes, 0o644)
		},
	})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"output_path":"shots/capture.png","include_data_url":true,"region":"10,20,300,180"}`)
	require.NoError(t, runErr)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	assert.Equal(t, filepath.Join(tmpDir, "shots", "capture.png"), payload["image_path"])
	assert.Equal(t, "10,20,300,180", payload["region"])
	assert.Contains(t, payload["preview"], "![screenshot](")
	assert.Contains(t, payload["screenshot"], "data:image/png;base64,")
}

func TestTool_InvokableRun_InvalidRegion(t *testing.T) {
	tl, err := New(Config{DefaultBaseDir: t.TempDir()})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"output_path":"capture.png","region":"10,20,30"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "invalid region format")
}

var tinyPNGBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xDE, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0x00, 0x00,
	0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D, 0x53,
	0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
	0xAE, 0x42, 0x60, 0x82,
}
