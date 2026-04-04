package mcpserver

import (
	"os"
	"strings"
)

const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
	TransportAll   = "all"

	DefaultAddr    = ":8080"
	DefaultName    = "eino-tools"
	DefaultVersion = "dev"
)

type Config struct {
	Name       string
	Version    string
	BaseDir    string
	Addr       string
	Transport  string
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

func (c Config) WithDefaults() Config {
	out := c
	if strings.TrimSpace(out.Name) == "" {
		out.Name = DefaultName
	}
	if strings.TrimSpace(out.Version) == "" {
		out.Version = DefaultVersion
	}
	if strings.TrimSpace(out.Addr) == "" {
		out.Addr = DefaultAddr
	}
	if strings.TrimSpace(out.Transport) == "" {
		out.Transport = TransportAll
	}
	if strings.TrimSpace(out.BaseDir) == "" {
		if wd, err := os.Getwd(); err == nil {
			out.BaseDir = wd
		} else {
			out.BaseDir = "."
		}
	}
	return out
}
