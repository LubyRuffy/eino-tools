package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func BuildServer(ctx context.Context, cfg Config) (*mcp.Server, error) {
	cfg = cfg.WithDefaults()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Name,
		Title:   cfg.Name,
		Version: cfg.Version,
	}, &mcp.ServerOptions{
		Logger: slog.Default(),
	})

	tools, err := NewToolset(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := RegisterTools(ctx, server, tools); err != nil {
		return nil, err
	}
	return server, nil
}

func RunStdio(ctx context.Context, server *mcp.Server) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}

func NewHTTPServer(server *mcp.Server, addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: NewHTTPHandler(server),
	}
}
