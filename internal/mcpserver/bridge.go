package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	einotool "github.com/cloudwego/eino/components/tool"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EinoTool interface {
	einotool.BaseTool
	einotool.InvokableTool
}

func ToMCPTool(ctx context.Context, tl EinoTool) (*mcp.Tool, mcp.ToolHandler, error) {
	if tl == nil {
		return nil, nil, fmt.Errorf("tool is nil")
	}

	info, err := tl.Info(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("get tool info: %w", err)
	}

	inputSchema, err := toolInputSchema(info)
	if err != nil {
		return nil, nil, err
	}

	mcpTool := &mcp.Tool{
		Name:        info.Name,
		Description: info.Desc,
		InputSchema: inputSchema,
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := []byte("{}")
		sessionID := ""
		if req != nil && req.Session != nil {
			sessionID = req.Session.ID()
		}
		if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
			args = req.Params.Arguments
		}
		slog.Default().Info("mcp tool call", "session_id", sessionID, "tool", info.Name)

		result, runErr := tl.InvokableRun(ctx, string(args))
		if runErr != nil {
			slog.Default().Error("mcp tool call failed", "session_id", sessionID, "tool", info.Name, "error", runErr)
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{&mcp.TextContent{Text: runErr.Error()}},
			}, nil
		}

		res := &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}
		var structured map[string]any
		if err := json.Unmarshal([]byte(result), &structured); err == nil {
			res.StructuredContent = structured
		}
		slog.Default().Info("mcp tool call completed", "session_id", sessionID, "tool", info.Name)
		return res, nil
	}

	return mcpTool, handler, nil
}

func RegisterTools(ctx context.Context, server *mcp.Server, tools []EinoTool) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}
	for _, tl := range tools {
		mcpTool, handler, err := ToMCPTool(ctx, tl)
		if err != nil {
			return err
		}
		server.AddTool(mcpTool, handler)
	}
	return nil
}

func ToolNames(ctx context.Context, tools []EinoTool) ([]string, error) {
	names := make([]string, 0, len(tools))
	for _, tl := range tools {
		info, err := tl.Info(ctx)
		if err != nil {
			return nil, err
		}
		names = append(names, info.Name)
	}
	return names, nil
}

func toolInputSchema(info *einoschema.ToolInfo) (any, error) {
	if info == nil || info.ParamsOneOf == nil {
		return map[string]any{"type": "object"}, nil
	}

	schema, err := info.ParamsOneOf.ToJSONSchema()
	if err != nil {
		return nil, fmt.Errorf("convert %s params schema: %w", info.Name, err)
	}
	if schema == nil {
		return map[string]any{"type": "object"}, nil
	}
	return schema, nil
}
