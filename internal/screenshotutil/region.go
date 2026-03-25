package screenshotutil

import (
	"fmt"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
)

type Region struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (r *Region) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%d,%d,%d,%d", r.X, r.Y, r.Width, r.Height)
}

func ParseRegion(raw string) (*Region, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid region format, expected x,y,width,height")
	}

	values := make([]int, 4)
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid region value %q: %w", strings.TrimSpace(part), err)
		}
		values[i] = value
	}

	if values[0] < 0 || values[1] < 0 {
		return nil, fmt.Errorf("region x and y must be non-negative")
	}
	if values[2] <= 0 || values[3] <= 0 {
		return nil, fmt.Errorf("region width and height must be greater than 0")
	}

	return &Region{
		X:      values[0],
		Y:      values[1],
		Width:  values[2],
		Height: values[3],
	}, nil
}

func NormalizeOutputPath(outputPath string) (string, error) {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return "", fmt.Errorf("output_path is required")
	}
	ext := strings.ToLower(filepath.Ext(outputPath))
	if ext == "" {
		outputPath += ".png"
		ext = ".png"
	}
	if !IsSupportedExt(ext) {
		return "", fmt.Errorf("unsupported screenshot file extension: %s", ext)
	}
	return outputPath, nil
}

func MimeType(path string) string {
	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mimeType == "" {
		return "image/png"
	}
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		return mimeType[:idx]
	}
	return mimeType
}

func IsSupportedExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}
