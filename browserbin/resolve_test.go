package browserbin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolve_PreferredMissing_ReturnsError(t *testing.T) {
	_, err := Resolve(filepath.Join(t.TempDir(), "no-such-browser"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "browser binary not found")
}

func TestResolve_PreferredExists_ReturnsPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-chrome")
	require.NoError(t, os.WriteFile(bin, []byte("x"), 0o755))

	got, err := Resolve(bin)
	require.NoError(t, err)
	require.Equal(t, bin, got)
}

func TestResolve_EmptyPreferred_BravePreferredOverChrome(t *testing.T) {
	brave := "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
	chrome := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

	exists := func(path string) bool {
		return path == brave || path == chrome
	}

	got, err := resolve("", "darwin", exists, nil)
	require.NoError(t, err)
	require.Equal(t, brave, got)
}

func TestResolve_EmptyPreferred_NoCandidates_ReturnsEmpty(t *testing.T) {
	got, err := resolve("", "darwin", func(string) bool { return false }, nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestCandidates_OrderAndNonEmpty(t *testing.T) {
	tests := []struct {
		goos     string
		wantHead []string
	}{
		{
			goos: "darwin",
			wantHead: []string{
				"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
				"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
				"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
				"/Applications/Chromium.app/Contents/MacOS/Chromium",
			},
		},
		{
			goos: "linux",
			wantHead: []string{
				"/usr/bin/brave-browser",
				"/usr/bin/brave",
				"/usr/bin/microsoft-edge",
				"/usr/bin/microsoft-edge-stable",
			},
		},
		{
			goos: "windows",
			wantHead: []string{
				filepath.Join("C:\\Program Files", "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
				filepath.Join("C:\\Program Files (x86)", "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			var got []string
			switch tt.goos {
			case "windows":
				t.Setenv("ProgramFiles", `C:\Program Files`)
				t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)
				got = candidates(tt.goos, nil)
			default:
				got = candidates(tt.goos, func(string) (string, error) {
					return "", os.ErrNotExist
				})
			}

			require.NotEmpty(t, got)
			require.GreaterOrEqual(t, len(got), len(tt.wantHead))
			require.Equal(t, tt.wantHead, got[:len(tt.wantHead)])
		})
	}
}

func TestCandidates_Linux_LookPathBeforeAbsolute(t *testing.T) {
	const fromPath = "/opt/brave/brave-browser"

	got := candidates("linux", func(name string) (string, error) {
		if name == "brave-browser" {
			return fromPath, nil
		}
		return "", os.ErrNotExist
	})

	require.NotEmpty(t, got)
	require.Equal(t, fromPath, got[0])
	require.Contains(t, got, "/usr/bin/brave-browser")
}

func TestCandidates_Windows_BrowserOrder(t *testing.T) {
	t.Setenv("ProgramFiles", `C:\Program Files`)
	t.Setenv("ProgramFiles(x86)", `C:\Program Files (x86)`)

	got := candidates("windows", nil)
	require.NotEmpty(t, got)

	bravePF := filepath.Join(`C:\Program Files`, "BraveSoftware", "Brave-Browser", "Application", "brave.exe")
	edgePF := filepath.Join(`C:\Program Files`, "Microsoft", "Edge", "Application", "msedge.exe")
	chromePF := filepath.Join(`C:\Program Files`, "Google", "Chrome", "Application", "chrome.exe")
	chromiumPF := filepath.Join(`C:\Program Files`, "Chromium", "Application", "chrome.exe")

	idx := func(path string) int {
		for i, p := range got {
			if p == path {
				return i
			}
		}
		return -1
	}

	require.Less(t, idx(bravePF), idx(edgePF))
	require.Less(t, idx(edgePF), idx(chromePF))
	require.Less(t, idx(chromePF), idx(chromiumPF))
}
