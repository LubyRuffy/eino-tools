package browserbin

import (
	"os"
	"path/filepath"
)

func candidates(goos string, lookPath func(string) (string, error)) []string {
	switch goos {
	case "darwin":
		return darwinCandidates()
	case "linux":
		return linuxCandidates(lookPath)
	case "windows":
		return windowsCandidates()
	default:
		return nil
	}
}

func darwinCandidates() []string {
	return []string{
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
}

func linuxCandidates(lookPath func(string) (string, error)) []string {
	var out []string
	out = appendLinuxBrowser(out, lookPath, "brave-browser", "brave")
	out = appendLinuxBrowser(out, lookPath, "microsoft-edge", "microsoft-edge-stable")
	out = appendLinuxBrowser(out, lookPath, "google-chrome", "google-chrome-stable")
	out = appendLinuxBrowser(out, lookPath, "chromium", "chromium-browser")
	return out
}

func appendLinuxBrowser(out []string, lookPath func(string) (string, error), names ...string) []string {
	if lookPath != nil {
		for _, name := range names {
			if path, err := lookPath(name); err == nil && path != "" {
				out = append(out, path)
			}
		}
	}
	for _, name := range names {
		out = append(out, filepath.Join("/usr/bin", name))
	}
	return out
}

func windowsCandidates() []string {
	roots := windowsProgramFilesRoots()
	relPaths := []string{
		filepath.Join("BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		filepath.Join("Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join("Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join("Chromium", "Application", "chrome.exe"),
	}

	var out []string
	for _, rel := range relPaths {
		for _, root := range roots {
			out = append(out, filepath.Join(root, rel))
		}
	}
	return out
}

func windowsProgramFilesRoots() []string {
	var roots []string
	for _, key := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(key); root != "" {
			roots = append(roots, root)
		}
	}
	return roots
}
