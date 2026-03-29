package mcpserver

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewHTTPHandler(server *mcp.Server) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/sse", mcp.NewSSEHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil))
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil))
	return mux
}
