package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPHandler_MountsSSEAndMCPRoutes(t *testing.T) {
	server, err := BuildServer(context.Background(), Config{
		BaseDir: t.TempDir(),
		Name:    "eino-tools",
		Version: "dev",
	})
	require.NoError(t, err)

	handler := NewHTTPHandler(server)

	sseReq := httptest.NewRequest(http.MethodPut, "/sse", nil)
	sseRec := httptest.NewRecorder()
	handler.ServeHTTP(sseRec, sseReq)
	require.NotEqual(t, http.StatusNotFound, sseRec.Code)

	mcpReq := httptest.NewRequest(http.MethodPut, "/mcp", nil)
	mcpReq.Header.Set("Accept", "application/json, text/event-stream")
	mcpRec := httptest.NewRecorder()
	handler.ServeHTTP(mcpRec, mcpReq)
	require.NotEqual(t, http.StatusNotFound, mcpRec.Code)
}
