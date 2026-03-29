package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestBuildServer_RegistersToolsForClientListing(t *testing.T) {
	server, err := BuildServer(context.Background(), Config{
		BaseDir: t.TempDir(),
		Name:    "eino-tools",
		Version: "dev",
	})
	require.NoError(t, err)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err = server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)

	client := mcp.NewClient(&mcp.Implementation{Name: "tester", Version: "dev"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer session.Close()

	res, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	names := make([]string, 0, len(res.Tools))
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}
	require.Contains(t, names, "web_search")
	require.Contains(t, names, "python_runner")
	require.NotContains(t, names, "fetchurl")
	require.NotContains(t, names, "bashcmd")
}
