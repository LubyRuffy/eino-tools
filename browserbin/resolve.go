package browserbin

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Resolve returns an existing Chromium-family browser binary.
// If preferred is non-empty, it must exist or Resolve returns an error.
// If preferred is empty, candidates are tried in Brave > Edge > Chrome > Chromium order.
// If none exist, Resolve returns ("", nil) so callers can fall back to Rod defaults.
func Resolve(preferred string) (string, error) {
	return resolve(preferred, runtime.GOOS, defaultExists, exec.LookPath)
}

func resolve(preferred, goos string, exists func(string) bool, lookPath func(string) (string, error)) (string, error) {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		if !exists(preferred) {
			return "", fmt.Errorf("browser binary not found: %s", preferred)
		}
		return preferred, nil
	}

	for _, candidate := range candidates(goos, lookPath) {
		if exists(candidate) {
			return candidate, nil
		}
	}
	return "", nil
}

func defaultExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
