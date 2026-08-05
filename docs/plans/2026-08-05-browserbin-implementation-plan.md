# browserbin + webfetch Bin 接线 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 新增公开 `browserbin` 包解析 Chromium 系浏览器路径，接线到 `webfetch` 默认 Rod launcher，使 `render=true` 能成功抓取 `https://www.sentinelone.com/vulnerability-database/cve-2026-50522/`。

**Architecture:** `browserbin.Resolve(preferred)` 负责显式路径校验与跨平台探测（Brave > Edge > Chrome > Chromium）；`webfetch.Config.BrowserBin` 注入 preferred；`newRodLauncher` 对非空结果调用 `launcher.Bin`。探测失败返回空路径以保持 Rod 默认兼容。

**Tech Stack:** Go, `os`/`runtime`/`os/exec.LookPath`, go-rod launcher, testify

**Design:** `docs/plans/2026-08-05-browserbin-design.md`

---

### Task 1: `browserbin` 失败测试（显式路径与未命中）

**Files:**
- Create: `browserbin/doc.go`
- Create: `browserbin/resolve.go`（可先空实现或最小 stub）
- Create: `browserbin/resolve_test.go`

**Step 1: Write the failing tests**

```go
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
}

func TestResolve_PreferredExists_ReturnsPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-chrome")
	require.NoError(t, os.WriteFile(bin, []byte("x"), 0o755))
	got, err := Resolve(bin)
	require.NoError(t, err)
	require.Equal(t, bin, got)
}

func TestResolve_EmptyPreferred_NoCandidates_ReturnsEmpty(t *testing.T) {
	// Use injectable lookup that finds nothing — after Options API exists.
	// Until then, this may be implemented via ResolveWith in Task 2.
}
```

先实现前两个测试；第三个在 Task 2 用可注入依赖完成。

**Step 2: Run tests to verify they fail**

Run: `go test ./browserbin -count=1`

Expected: FAIL（`Resolve` undefined 或未实现）

**Step 3: Minimal stub so package compiles**

```go
package browserbin

func Resolve(preferred string) (string, error) {
	return "", nil
}
```

**Step 4: Commit stub + failing preferred tests only after implementation in Task 2**

（本 Task 与 Task 2 可合并一次提交，若严格 TDD：先提交红测再实现）

---

### Task 2: 实现 `browserbin.Resolve` 与跨平台候选表

**Files:**
- Create: `browserbin/doc.go`
- Create: `browserbin/resolve.go`
- Create: `browserbin/candidates.go`（或同文件）
- Create: `browserbin/resolve_test.go`

**Step 1: API**

```go
package browserbin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type lookupFunc func(path string) bool

func Resolve(preferred string) (string, error) {
	return resolve(preferred, runtime.GOOS, defaultExists, exec.LookPath)
}

func resolve(preferred, goos string, exists lookupFunc, lookPath func(string) (string, error)) (string, error) {
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
```

**Step 2: `candidates(goos, lookPath)`**

按设计文档路径表生成有序列表：Brave → Edge → Chrome → Chromium。Linux 先 `lookPath` 再绝对路径；Windows 拼 `ProgramFiles` / `ProgramFiles(x86)`。

**Step 3: Tests**

- preferred 存在 / 不存在
- 空 preferred：注入 fake exists，断言 Brave 优先于 Chrome
- 空 preferred：全部不存在 → `("", nil)`
- `candidates` 对 darwin/linux/windows 非空且顺序正确（表驱动）

**Step 4: Run**

Run: `go test ./browserbin -count=1`

Expected: PASS

**Step 5: Commit**

```bash
git add browserbin/
git commit -m "$(cat <<'EOF'
feat: add browserbin package for Chromium-family path resolution

EOF
)"
```

---

### Task 3: `webfetch` 接线 `BrowserBin` + launcher Bin

**Files:**
- Modify: `webfetch/tool.go`（`Config`、`Tool`、`New`、`newRodLauncher`、`launchRodBrowser` / `fetchRenderedMarkdown` 传参）
- Modify: `webfetch/tool_test.go`
- Modify: `webfetch/sentinelone_test.go`（改为走默认 render 路径，去掉自定义 BrowserFetch 若不再需要）

**Step 1: Failing test**

```go
func TestNewRodLauncher_AppliesBrowserBin(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "Brave Browser")
	require.NoError(t, os.WriteFile(bin, []byte("x"), 0o755))

	launch, err := newRodLauncher(context.Background(), true, netproxy.Config{}, bin)
	require.NoError(t, err)
	require.Equal(t, bin, launch.Get(flags.Bin))
}

func TestNewRodLauncher_InvalidBrowserBin_ReturnsError(t *testing.T) {
	_, err := newRodLauncher(context.Background(), true, netproxy.Config{}, filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}
```

同步更新现有 `TestNewRodLauncher_AppliesProxyConfig` 的 `newRodLauncher` 签名。

**Step 2: Run to fail**

Run: `go test ./webfetch -run 'TestNewRodLauncher_' -count=1`

Expected: FAIL（签名/未接线）

**Step 3: Implement**

```go
// Config
BrowserBin string

// Tool 字段 browserBin string
// New: t.browserBin = strings.TrimSpace(cfg.BrowserBin)

func newRodLauncher(ctx context.Context, headless bool, proxyCfg netproxy.Config, browserBin string) (*launcher.Launcher, error) {
	launch := launcher.New().Context(ctx).Headless(headless)
	bin, err := browserbin.Resolve(browserBin)
	if err != nil {
		return nil, err
	}
	if bin != "" {
		launch = launch.Bin(bin)
	}
	// existing proxy flags...
	return launch, nil
}
```

`launchRodBrowser` / `fetchRenderedMarkdown` 传入 `t.browserBin`。

**Step 4: Run unit tests**

Run: `go test ./webfetch -count=1 -short`

Expected: PASS（跳过集成）

**Step 5: Commit**

```bash
git add webfetch/tool.go webfetch/tool_test.go
git commit -m "$(cat <<'EOF'
feat: wire webfetch Rod launcher to browserbin.Resolve

EOF
)"
```

---

### Task 4: SentinelOne 集成验收

**Files:**
- Modify or keep: `webfetch/sentinelone_test.go`

**Step 1: 简化集成测试**

走真实默认路径（探测本机 Brave 等），不要再注入自定义 `BrowserFetch`，除非默认 render 已足够：

```go
func TestFetch_SentinelOneCVE(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tl, err := New(Config{}) // BrowserBin 空 → 自动探测 Brave 优先
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result, err := tl.Fetch(ctx, "https://www.sentinelone.com/vulnerability-database/cve-2026-50522/", true)
	require.NoError(t, err)
	require.NotEmpty(t, result)
	lower := strings.ToLower(result)
	require.True(t, strings.Contains(lower, "cve-2026-50522") || strings.Contains(lower, "cve"))
	// vulnerability-related content + len > 100
}
```

若默认 `fetchRenderedMarkdown` 在 Cloudflare 下仍不够（挑战未等完），允许**最小**增强默认 render 等待逻辑，但优先验证 Bin=Brave 是否已足够；避免为单站硬编码域名特例。

**Step 2: Run goal verification**

Run: `go test ./webfetch -run TestFetch_SentinelOneCVE -count=1 -v`

Expected: PASS，日志可见实质 CVE 正文

**Step 3: Commit**

```bash
git add webfetch/sentinelone_test.go webfetch/
git commit -m "$(cat <<'EOF'
test: verify web_fetch render against SentinelOne CVE page

EOF
)"
```

---

### Task 5: 文档与全量回归

**Files:**
- Modify: `README.md`
- Modify: `docs/CONFIG.md`
- Modify: `docs/TESTING.md`
- Modify: `CHANGELOG.md`
- Modify: `ARCHITECTURE.md`（简述 `browserbin`）

**Step 1: 文档更新要点**

- `Config.BrowserBin` 用法与探测优先级
- 未设置时自动探测；显式无效路径报错；探测失败回退 Rod
- 测试说明含 `browserbin` 与 SentinelOne 集成（非 short）

**Step 2: 回归**

```bash
go test ./browserbin ./webfetch ./netproxy -count=1 -short
go test ./... -count=1 -short
go build ./cmd/mcpserver
```

Expected: PASS / 构建成功

**Step 3: Commit**

```bash
git add README.md ARCHITECTURE.md docs/CONFIG.md docs/TESTING.md CHANGELOG.md
git commit -m "$(cat <<'EOF'
docs: document browserbin and webfetch BrowserBin config

EOF
)"
```

---

### Task 6: 最终 goal 复核

**Step 1:** 再跑一次非 short 集成测试

Run: `go test ./webfetch -run TestFetch_SentinelOneCVE -count=1 -v`

Expected: PASS

**Step 2:** 确认本轮生产代码文件均 < 1000 行

Run: `wc -l browserbin/*.go webfetch/tool.go`

**Done when:** 上述命令通过，且结果含目标 CVE 页实质内容。
