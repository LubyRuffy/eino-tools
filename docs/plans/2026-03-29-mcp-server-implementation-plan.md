# MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `eino-tools` 增加基于官方 `github.com/modelcontextprotocol/go-sdk` 的 `cmd/mcpserver`，默认支持 `stdio`、`SSE`、`streamable HTTP`，并删除 `fetchurl` / `bashcmd` 别名包。

**Architecture:** 新增 `internal/mcpserver` 负责工具构造、Eino tool 到 MCP tool 的桥接、HTTP 端点挂载和传输启动；`cmd/mcpserver` 仅处理参数解析和进程生命周期。桥接层直接复用现有工具的 `Info()` 与 `InvokableRun()`，避免复制工具定义并保持行为稳定。实现按 TDD 推进，先锁定删除别名和 MCP server 的行为，再补代码与文档。

**Tech Stack:** Go 1.26、CloudWeGo Eino、Model Context Protocol Go SDK、testify

---

执行时必须遵循：@writing-plans、@test-driven-development、@verification-before-completion。

### Task 1: 写入计划文档并清点当前引用面

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/plans/2026-03-29-mcp-server-design.md`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/plans/2026-03-29-mcp-server-implementation-plan.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/README.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ARCHITECTURE.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/AGENTS.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/TESTING.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/CHANGELOG.md`

**Step 1: Write the failing check**

先检索仓库内所有 `fetchurl` / `bashcmd` / MCP 相关引用，形成明确的改动面清单。

**Step 2: Run check to verify current state**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && rg -n "fetchurl|bashcmd|mcpserver|modelcontextprotocol|go-sdk" -S`
Expected: 能看到旧别名仍存在，且当前没有 `cmd/mcpserver`。

**Step 3: Write minimal documentation**

把 MCP server 方案和删除别名策略写入 `docs/plans`，作为后续实现约束。

**Step 4: Verify documentation exists**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && test -f docs/plans/2026-03-29-mcp-server-design.md && test -f docs/plans/2026-03-29-mcp-server-implementation-plan.md`
Expected: PASS。

### Task 2: 先写 MCP server 工具注册与桥接失败测试

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/bridge_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/tools_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/http_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/server_test.go`

**Step 1: Write the failing test**

覆盖：
- 正式工具列表包含 12 个工具
- 不包含 `fetchurl` / `bashcmd`
- bridge 能把 arguments JSON 透传到底层 `InvokableRun`
- HTTP mux 暴露 `/sse` 与 `/mcp`
- `stdio` server 可完成基础构建

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/mcpserver -count=1`
Expected: FAIL，提示新包和函数不存在。

**Step 3: Write minimal implementation contract**

为每个测试明确需要的入口函数，例如：
- `NewToolSet`
- `BuildServer`
- `BuildHTTPHandler`
- `RegisterTools`

**Step 4: Re-run test to confirm failures are meaningful**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/mcpserver -count=1`
Expected: 仍 FAIL，但失败原因聚焦在未实现逻辑而非测试自身错误。

### Task 3: 先写 `cmd/mcpserver` 启动失败测试

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/cmd/mcpserver/main_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/cmd/mcpserver/testdata/.gitkeep`

**Step 1: Write the failing test**

覆盖：
- 默认配置为 `all`
- `http` 模式默认地址有效
- `stdio` / `http` / `all` 为合法传输
- 非法传输值返回错误

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./cmd/mcpserver -count=1`
Expected: FAIL。

**Step 3: Write minimal config contract**

定义 `main` 需要的解析函数与配置结构，先不实现全部启动逻辑。

**Step 4: Re-run test to confirm failures are meaningful**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./cmd/mcpserver -count=1`
Expected: FAIL，但失败原因集中在未实现的 main/config 逻辑。

### Task 4: 实现 `internal/mcpserver` 桥接层

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/config.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/bridge.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/tools.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/http.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/mcpserver/server.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/go.mod`

**Step 1: Write the minimal implementation**

1. 引入官方 `github.com/modelcontextprotocol/go-sdk`
2. 实现工具构造函数
3. 实现 Eino tool 到 MCP tool 的 schema / handler 映射
4. 实现 `/sse`、`/mcp` handler 构造
5. 实现 `stdio` / HTTP 启动器

**Step 2: Run package tests**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/mcpserver -count=1`
Expected: PASS。

**Step 3: Refactor for clarity**

把 schema 映射、tool 构造、transport 启动拆到独立文件，保持单文件不超过 1000 行。

**Step 4: Re-run package tests**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/mcpserver -count=1`
Expected: PASS。

### Task 5: 实现 `cmd/mcpserver`

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/cmd/mcpserver/main.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/cmd/mcpserver/main_test.go`

**Step 1: Write the minimal implementation**

实现：
- flags 解析
- 默认 `transport=all`
- `base_dir` 注入
- `stdio` / HTTP / all 启动调度
- 优雅退出

**Step 2: Run package tests**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./cmd/mcpserver -count=1`
Expected: PASS。

**Step 3: Run build verification**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go build ./cmd/mcpserver`
Expected: PASS。

**Step 4: Re-run package tests**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./cmd/mcpserver -count=1`
Expected: PASS。

### Task 6: 删除 `fetchurl` / `bashcmd` 及其测试

**Files:**
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/fetchurl/doc.go`
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/fetchurl/tool.go`
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/fetchurl/tool_test.go`
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/bashcmd/doc.go`
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/bashcmd/path.go`
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/bashcmd/tool.go`
- Delete: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/bashcmd/tool_test.go`

**Step 1: Delete the compatibility packages**

删除别名包与对应测试。

**Step 2: Run repository tests to surface stale references**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./... -count=1`
Expected: FAIL，如果仍有旧引用。

**Step 3: Remove stale references**

清理 README、ARCHITECTURE、AGENTS、docs/TESTING、CHANGELOG 与代码中的旧命名引用。

**Step 4: Re-run repository tests**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./... -count=1`
Expected: PASS 或只剩可定位的新问题。

### Task 7: 更新使用者与开发者文档

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/README.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ARCHITECTURE.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/AGENTS.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/TESTING.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/CHANGELOG.md`

**Step 1: Write the documentation changes**

README 至少补：
- 项目简介
- `cmd/mcpserver` 快速启动
- `stdio` / `SSE` / `streamable HTTP` 使用示例
- 当前正式工具列表

ARCHITECTURE 至少补：
- `internal/mcpserver`
- 请求流从 transport 到 tool handler 的路径

AGENTS / docs/TESTING 至少补：
- 新增构建和测试命令
- MCP server 验证方式

CHANGELOG 记录：
- Added
- Changed
- Fixed

**Step 2: Run focused validation**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && rg -n "fetchurl|bashcmd" README.md ARCHITECTURE.md AGENTS.md docs/TESTING.md CHANGELOG.md -S`
Expected: 无旧别名残留，或只有明确删除说明。

**Step 3: Review docs for consistency**

逐份对照当前代码结构，修正文档中的旧命名和旧测试命令。

**Step 4: Re-run focused validation**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && rg -n "fetchurl|bashcmd" README.md ARCHITECTURE.md AGENTS.md docs/TESTING.md CHANGELOG.md -S`
Expected: 仅保留“已删除别名”的说明。

### Task 8: 完整验证

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/go.mod`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/go.sum`

**Step 1: Run full test suite**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./... -count=1`
Expected: PASS。

**Step 2: Run build verification**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go build ./...`
Expected: PASS。

**Step 3: Run final focused smoke test**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/mcpserver ./cmd/mcpserver -count=1`
Expected: PASS。

**Step 4: Prepare completion summary**

记录：
- MCP server 支持的传输
- 正式工具列表
- 删除的别名包
- 已执行的测试与构建命令
