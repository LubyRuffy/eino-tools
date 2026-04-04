# render=true 代理统一 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让 `web_fetch` 的默认 `render=true` 与普通 HTTP 抓取复用同一套代理配置，并对 package 模式公开统一接入方式。

**Architecture:** 新增公开 `netproxy` package 负责解析代理配置、构造共享 `HTTPClient`、以及生成 Chromium 代理参数。`websearch` / `webfetch` 直接接收 `netproxy.Config`，`internal/mcpserver` 复用同一套能力把 CLI / 环境变量代理配置同时注入 HTTP 与浏览器链路。

**Tech Stack:** Go, `net/http`, `httpproxy`, Rod launcher, testify

---

### Task 1: 为统一代理配置写失败测试

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/netproxy/proxy_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/webfetch/tool_test.go`

**Step 1: Write the failing test**

- 测 `Resolve` / `NewHTTPClient` / Chromium 映射
- 测 `render=true` 默认 launcher 能吃到统一代理配置

**Step 2: Run test to verify it fails**

Run: `go test ./netproxy ./webfetch -count=1`
Expected: FAIL，提示缺少公开代理 package 或 render 代理未注入

**Step 3: Write minimal implementation**

- 新增 `netproxy` package
- 让 `webfetch` 默认 render 路径接线

**Step 4: Run test to verify it passes**

Run: `go test ./netproxy ./webfetch -count=1`
Expected: PASS

### Task 2: 把统一代理配置接到 `websearch` 与 `internal/mcpserver`

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/websearch/tool.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/network.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/tools.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/tools_test.go`

**Step 1: Write the failing test**

- 断言 `internal/mcpserver` 会注入统一代理配置

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcpserver -count=1`
Expected: FAIL

**Step 3: Write minimal implementation**

- `websearch` 支持 `ProxyConfig`
- `internal/mcpserver` 统一注入 `HTTPClient` 和 `ProxyConfig`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/mcpserver -count=1`
Expected: PASS

### Task 3: 更新文档

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/README.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ARCHITECTURE.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/CONFIG.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/TESTING.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/CHANGELOG.md`

**Step 1: Apply updates**

- 把 pkg 模式推荐方式切到统一 `ProxyConfig`
- 明确 `render=true` 已可复用同一套代理配置

**Step 2: Verify**

Run: `rg -n "ProxyConfig|render=true|netproxy|HTTPClient" README.md ARCHITECTURE.md docs/CONFIG.md docs/TESTING.md CHANGELOG.md`
Expected: 相关说明完整出现

### Task 4: 全量验证

**Files:**
- Modify: 无

**Step 1: Run targeted tests**

Run: `go test ./netproxy ./webfetch ./websearch ./internal/mcpserver ./cmd/mcpserver -count=1`
Expected: PASS

**Step 2: Run full suite**

Run: `go test ./... -count=1`
Expected: PASS

**Step 3: Build**

Run: `go build ./cmd/mcpserver`
Expected: PASS
