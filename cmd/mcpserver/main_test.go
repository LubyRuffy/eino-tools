package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig_DefaultsToAllTransport(t *testing.T) {
	cfg, err := parseConfig(nil)
	require.NoError(t, err)
	require.Equal(t, "all", cfg.Transport)
	require.Equal(t, ":8080", cfg.Addr)
	require.NotEmpty(t, cfg.Name)
	require.NotEmpty(t, cfg.Version)
}

func TestParseConfig_AcceptsKnownTransports(t *testing.T) {
	for _, transport := range []string{"stdio", "http", "all"} {
		cfg, err := parseConfig([]string{"--transport", transport})
		require.NoError(t, err)
		require.Equal(t, transport, cfg.Transport)
	}
}

func TestParseConfig_RejectsUnknownTransport(t *testing.T) {
	_, err := parseConfig([]string{"--transport", "invalid"})
	require.Error(t, err)
}

func TestParseConfig_AcceptsProxyFlags(t *testing.T) {
	cfg, err := parseConfig([]string{
		"--http-proxy", "http://127.0.0.1:7890",
		"--https-proxy", "http://127.0.0.1:7891",
		"--no-proxy", "localhost,127.0.0.1,.internal",
	})
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:7890", cfg.HTTPProxy)
	require.Equal(t, "http://127.0.0.1:7891", cfg.HTTPSProxy)
	require.Equal(t, "localhost,127.0.0.1,.internal", cfg.NoProxy)
}
