package write

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func writeVerified(absPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return confirmBytes(absPath, content)
}

func confirmBytes(path string, want []byte) error {
	got, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to verify written file: %w", err)
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("written bytes do not match content at %s: expected %d bytes, got %d", path, len(want), len(got))
	}
	return nil
}
