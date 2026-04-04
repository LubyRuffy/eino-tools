package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	internalmcp "github.com/LubyRuffy/eino-tools/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string) error {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := internalmcp.BuildServer(ctx, cfg)
	if err != nil {
		return err
	}

	switch cfg.Transport {
	case internalmcp.TransportStdio:
		return internalmcp.RunStdio(ctx, server)
	case internalmcp.TransportHTTP:
		return runHTTP(ctx, server, cfg.Addr)
	case internalmcp.TransportAll:
		return runAll(ctx, server, cfg.Addr)
	default:
		return fmt.Errorf("unsupported transport: %s", cfg.Transport)
	}
}

func parseConfig(args []string) (internalmcp.Config, error) {
	fs := flag.NewFlagSet("mcpserver", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := internalmcp.Config{}
	fs.StringVar(&cfg.Transport, "transport", internalmcp.TransportAll, "stdio, http, or all")
	fs.StringVar(&cfg.Addr, "addr", internalmcp.DefaultAddr, "HTTP listen address")
	fs.StringVar(&cfg.BaseDir, "base-dir", "", "base directory passed to filesystem-oriented tools")
	fs.StringVar(&cfg.HTTPProxy, "http-proxy", "", "HTTP proxy URL for network tools")
	fs.StringVar(&cfg.HTTPSProxy, "https-proxy", "", "HTTPS proxy URL for network tools")
	fs.StringVar(&cfg.NoProxy, "no-proxy", "", "comma-separated hosts that bypass the proxy for network tools")
	fs.StringVar(&cfg.Name, "name", internalmcp.DefaultName, "MCP server name")
	fs.StringVar(&cfg.Version, "version", version, "MCP server version")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}

	cfg = cfg.WithDefaults()
	switch cfg.Transport {
	case internalmcp.TransportStdio, internalmcp.TransportHTTP, internalmcp.TransportAll:
		return cfg, nil
	default:
		return cfg, fmt.Errorf("invalid transport %q", cfg.Transport)
	}
}

func runHTTP(ctx context.Context, server *mcp.Server, addr string) error {
	httpServer := internalmcp.NewHTTPServer(server, addr)

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		errCh <- httpServer.Shutdown(context.Background())
	}()

	listenErr := httpServer.ListenAndServe()
	shutdownErr := <-errCh
	if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
		return listenErr
	}
	if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
		return shutdownErr
	}
	return nil
}

func runAll(ctx context.Context, server *mcp.Server, addr string) error {
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)
	go func() {
		errCh <- internalmcp.RunStdio(ctx2, server)
	}()
	go func() {
		errCh <- runHTTP(ctx2, server, addr)
	}()

	err := <-errCh
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return err
}
