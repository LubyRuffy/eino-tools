# 网络工具代理配置 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `cmd/mcpserver` 增加网络工具代理配置，并把代理统一注入 `web_search` 与 `web_fetch`

**Architecture:** 在 `cmd/mcpserver` 解析代理相关 flags，写入 `internal/mcpserver.Config`。`internal/mcpserver` 统一根据 CLI 参数和环境变量构造共享 `http.Client`，并注入到两个网络工具中，保证配置入口集中且默认行为兼容。

**Tech Stack:** Go, `net/http`, `flag`, testify

---

### Task 1: 补代理配置解析测试

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/cmd/mcpserver/main_test.go`

**Step 1: Write the failing test**

- 增加 CLI 参数解析用例，断言 `--http-proxy`、`--https-proxy`、`--no-proxy` 能写入 config

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/mcpserver -count=1`
Expected: FAIL，提示 config 缺少代理字段或 flag 未定义

**Step 3: Write minimal implementation**

- 在 `cmd/mcpserver/main.go` 增加 flag 解析
- 在 `internal/mcpserver.Config` 增加代理字段

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/mcpserver -count=1`
Expected: PASS

### Task 2: 补共享 HTTP client 代理优先级测试

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/tools_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/tools.go`

**Step 1: Write the failing test**

- 增加对共享代理 client 的测试
- 断言 CLI 配置优先于环境变量
- 断言仅环境变量存在时也能为网络工具生成代理 transport

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcpserver -count=1`
Expected: FAIL，提示缺少构造逻辑或优先级不符

**Step 3: Write minimal implementation**

- 提供共享 client 构造函数
- 把 client 注入 `websearch` 与 `webfetch`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/mcpserver -count=1`
Expected: PASS

### Task 3: 更新使用者与开发者文档

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/README.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ARCHITECTURE.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/CLI.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/CONFIG.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/TESTING.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/CHANGELOG.md`

**Step 1: Write the failing check**

- 先检查文档是否缺少代理配置说明

**Step 2: Apply minimal updates**

- 补 CLI 参数说明、环境变量兜底规则、网络工具注入说明、测试覆盖说明

**Step 3: Verify**

Run: `rg -n "proxy|HTTP_PROXY|HTTPS_PROXY|NO_PROXY" README.md ARCHITECTURE.md docs/CLI.md docs/CONFIG.md docs/TESTING.md CHANGELOG.md`
Expected: 所有相关文档都出现代理配置说明

### Task 4: 全量验证

**Files:**
- Modify: 无

**Step 1: Run targeted tests**

Run: `go test ./cmd/mcpserver ./internal/mcpserver ./websearch ./webfetch -count=1`
Expected: PASS

**Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS

**Step 3: Build server**

Run: `go build ./cmd/mcpserver`
Expected: PASS
