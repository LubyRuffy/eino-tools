package editutil

import (
	"fmt"
	"strings"
)

type PatchReplacement struct {
	Search  string
	Replace string
}

func NormalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

func ApplyReplaceBlockOnce(text string, search string, replace string) (string, int) {
	if search == "" {
		return text, 0
	}
	count := 0
	start := 0
	var out strings.Builder
	for {
		idx := strings.Index(text[start:], search)
		if idx < 0 {
			out.WriteString(text[start:])
			break
		}
		idx += start
		out.WriteString(text[start:idx])
		out.WriteString(replace)
		start = idx + len(search)
		count++
	}
	return out.String(), count
}

func ParseApplyPatchText(patchText string, expectedDisplayPath string) ([]PatchReplacement, error) {
	patchText = NormalizeNewlines(patchText)
	lines := strings.Split(patchText, "\n")
	var replacements []PatchReplacement

	inUpdateFile := false
	seenUpdateFile := false
	targetOK := expectedDisplayPath == ""

	var oldLines []string
	var newLines []string
	inHunk := false

	flushHunk := func() {
		if !inHunk {
			return
		}
		replacements = append(replacements, PatchReplacement{
			Search:  strings.Join(oldLines, "\n"),
			Replace: strings.Join(newLines, "\n"),
		})
		oldLines = nil
		newLines = nil
		inHunk = false
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Update File:"):
			seenUpdateFile = true
			inUpdateFile = true
			targetPath := strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))
			if targetPath == expectedDisplayPath {
				targetOK = true
			}
			flushHunk()
			continue
		case strings.HasPrefix(line, "*** Begin Patch"):
			continue
		case strings.HasPrefix(line, "*** End Patch"):
			flushHunk()
			break
		case strings.HasPrefix(line, "*** Add File:"), strings.HasPrefix(line, "*** Delete File:"):
			return nil, fmt.Errorf("patch operation not supported")
		case strings.HasPrefix(line, "@@"):
			flushHunk()
			inHunk = true
			oldLines = nil
			newLines = nil
			continue
		case !inUpdateFile && strings.HasPrefix(line, "--- "):
			continue
		case !inUpdateFile && strings.HasPrefix(line, "+++ "):
			continue
		case len(line) == 0 && !inHunk:
			continue
		}

		if !inHunk {
			continue
		}
		if line == "*** End of File" {
			continue
		}
		if len(line) == 0 {
			oldLines = append(oldLines, "")
			newLines = append(newLines, "")
			continue
		}

		switch line[0] {
		case ' ':
			oldLines = append(oldLines, line[1:])
			newLines = append(newLines, line[1:])
		case '-':
			oldLines = append(oldLines, line[1:])
		case '+':
			newLines = append(newLines, line[1:])
		default:
			return nil, fmt.Errorf("invalid patch line: %q", line)
		}
	}

	if seenUpdateFile && !targetOK {
		return nil, fmt.Errorf("patch Update File does not match file_path")
	}
	return replacements, nil
}
