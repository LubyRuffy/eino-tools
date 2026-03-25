package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
)

func ResolveBaseDir(defaultBaseDir string, override string) (string, error) {
	baseDir := strings.TrimSpace(defaultBaseDir)
	if strings.TrimSpace(override) != "" {
		baseDir = strings.TrimSpace(override)
	}
	if baseDir == "" {
		baseDir = "."
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base_dir: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("base_dir not accessible: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("base_dir is not a directory")
	}
	return abs, nil
}

func ResolvePathWithin(baseDir string, inputPath string, allowedPaths []string) (string, error) {
	if inputPath == "" {
		return "", fmt.Errorf("path is required")
	}
	if baseDir == "" {
		baseDir = "."
	}
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base_dir: %w", err)
	}

	if filepath.IsAbs(inputPath) {
		pathAbs, err := filepath.Abs(filepath.Clean(inputPath))
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
		if !IsPathWithin(baseAbs, pathAbs) && !isPathInAllowedList(pathAbs, allowedPaths) {
			return "", fmt.Errorf("path is outside base_dir")
		}
		return pathAbs, nil
	}

	joined, err := securejoin.SecureJoin(baseAbs, inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}
	if !IsPathWithin(baseAbs, joinedAbs) && !isPathInAllowedList(joinedAbs, allowedPaths) {
		return "", fmt.Errorf("path is outside base_dir")
	}
	return joinedAbs, nil
}

func IsPathWithin(baseDir string, target string) bool {
	baseDir = filepath.Clean(baseDir)
	target = filepath.Clean(target)
	if baseDir == target {
		return true
	}
	if !strings.HasSuffix(baseDir, string(os.PathSeparator)) {
		baseDir += string(os.PathSeparator)
	}
	return strings.HasPrefix(target, baseDir)
}

func DisplayPath(baseDir string, targetPath string) string {
	if rel, err := filepath.Rel(baseDir, targetPath); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return targetPath
}

func isPathInAllowedList(targetPath string, allowedPaths []string) bool {
	targetPath = filepath.Clean(targetPath)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, allowedPath := range allowedPaths {
		expanded := allowedPath
		if strings.HasPrefix(expanded, "~/") {
			expanded = filepath.Join(homeDir, expanded[2:])
		}
		expanded = filepath.Clean(expanded)
		if targetPath == expanded {
			return true
		}
		if !strings.HasSuffix(expanded, string(os.PathSeparator)) {
			expanded += string(os.PathSeparator)
		}
		if strings.HasPrefix(targetPath, expanded) {
			return true
		}
	}
	return false
}
