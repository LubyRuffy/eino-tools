package fsutil

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultSkipDirNames are directory basenames skipped during repository-wide walks.
// The walk root itself is never skipped, so an explicit search of dist/ still works.
var DefaultSkipDirNames = map[string]struct{}{
	".git":         {},
	"node_modules": {},
	".worktrees":   {},
	"dist":         {},
	"vendor":       {},
}

// WalkIgnore combines default skip-dir names with gitignore rules rooted at a search path.
type WalkIgnore struct {
	root  string
	rules []ignoreRule
}

type ignoreRule struct {
	baseAbs string
	negated bool
	dirOnly bool
	re      *regexp.Regexp
}

// FindGitRoot walks up from start until it finds a .git file or directory.
func FindGitRoot(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		info, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// LoadWalkIgnore loads default skip rules plus .gitignore from the git root and walk root.
func LoadWalkIgnore(root string) *WalkIgnore {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	w := &WalkIgnore{root: filepath.Clean(abs)}
	seen := make(map[string]struct{})
	if gitRoot := FindGitRoot(w.root); gitRoot != "" {
		gitIgnore := filepath.Join(gitRoot, ".gitignore")
		w.loadFile(gitIgnore, gitRoot)
		seen[gitIgnore] = struct{}{}
	}
	rootIgnore := filepath.Join(w.root, ".gitignore")
	if _, ok := seen[rootIgnore]; !ok {
		w.loadFile(rootIgnore, w.root)
	}
	return w
}

// SkipDir reports whether a directory should be pruned from a walk.
func (w *WalkIgnore) SkipDir(absPath, name string) bool {
	if w == nil {
		_, skip := DefaultSkipDirNames[name]
		return skip
	}
	absPath = filepath.Clean(absPath)
	if absPath == w.root {
		return false
	}
	if _, ok := DefaultSkipDirNames[name]; ok {
		return true
	}
	return w.ignored(absPath, true)
}

// SkipFile reports whether a file should be skipped. The walk root itself is never skipped.
func (w *WalkIgnore) SkipFile(absPath string) bool {
	if w == nil {
		return false
	}
	absPath = filepath.Clean(absPath)
	if absPath == w.root {
		return false
	}
	dir := filepath.Dir(absPath)
	for dir != w.root && dir != "." && dir != string(filepath.Separator) {
		if w.SkipDir(dir, filepath.Base(dir)) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return w.ignored(absPath, false)
}

func (w *WalkIgnore) loadFile(path, baseAbs string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		rule, ok := parseIgnoreLine(scanner.Text(), baseAbs)
		if ok {
			w.rules = append(w.rules, rule)
		}
	}
}

func (w *WalkIgnore) ignored(absPath string, isDir bool) bool {
	ignored := false
	for _, rule := range w.rules {
		if rule.matches(absPath, isDir) {
			ignored = !rule.negated
		}
	}
	return ignored
}

func (r ignoreRule) matches(absPath string, isDir bool) bool {
	if r.dirOnly && !isDir {
		return false
	}
	rel, err := filepath.Rel(r.baseAbs, absPath)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	return r.re.MatchString(filepath.ToSlash(rel))
}

func parseIgnoreLine(line, baseAbs string) (ignoreRule, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return ignoreRule{}, false
	}
	rule := ignoreRule{baseAbs: baseAbs}
	if strings.HasPrefix(line, "!") {
		rule.negated = true
		line = line[1:]
	}
	if strings.HasSuffix(line, "/") {
		rule.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}
	anchored := strings.HasPrefix(line, "/") || strings.Contains(strings.TrimPrefix(line, "/"), "/")
	line = strings.TrimPrefix(line, "/")
	if line == "" {
		return ignoreRule{}, false
	}
	re, err := globPatternToRegexp(line, anchored)
	if err != nil {
		return ignoreRule{}, false
	}
	rule.re = re
	return rule, true
}

func globPatternToRegexp(pattern string, anchored bool) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	if !anchored {
		b.WriteString("(?:.*/)?")
	}
	for i := 0; i < len(pattern); {
		if i+1 < len(pattern) && pattern[i] == '*' && pattern[i+1] == '*' {
			if i+2 < len(pattern) && pattern[i+2] == '/' {
				b.WriteString("(?:.*/)?")
				i += 3
				continue
			}
			b.WriteString(".*")
			i += 2
			continue
		}
		switch pattern[i] {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(pattern[i])
		default:
			b.WriteByte(pattern[i])
		}
		i++
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
