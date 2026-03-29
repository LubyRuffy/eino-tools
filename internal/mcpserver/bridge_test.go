package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	einotool "github.com/cloudwego/eino/components/tool"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

type stubEinoTool struct {
	info     *einoschema.ToolInfo
	lastArgs string
	result   string
	err      error
}

func (s *stubEinoTool) Info(context.Context) (*einoschema.ToolInfo, error) {
	return s.info, nil
}

func (s *stubEinoTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	s.lastArgs = argumentsInJSON
	return s.result, s.err
}

func TestToMCPTool_MapsInfoAndArguments(t *testing.T) {
	tl := &stubEinoTool{
		info: &einoschema.ToolInfo{
			Name: "demo_tool",
			Desc: "demo description",
			ParamsOneOf: einoschema.NewParamsOneOfByParams(map[string]*einoschema.ParameterInfo{
				"query": {Type: einoschema.String, Desc: "search query", Required: true},
			}),
		},
		result: `{"ok":true}`,
	}

	mcpTool, handler, err := ToMCPTool(context.Background(), tl)
	require.NoError(t, err)
	require.Equal(t, "demo_tool", mcpTool.Name)
	require.Equal(t, "demo description", mcpTool.Description)
	require.NotNil(t, mcpTool.InputSchema)

	req := &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      "demo_tool",
			Arguments: json.RawMessage(`{"query":"golang"}`),
		},
	}
	res, err := handler(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, `{"query":"golang"}`, tl.lastArgs)
	require.Len(t, res.Content, 1)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	require.Equal(t, `{"ok":true}`, text.Text)
}
