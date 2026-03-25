# Generic Tools Extraction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 `zwell_template` 的 12 个通用型内置工具下沉到 `eino-tools`，并让 `zwell_template` 直接通过 module 依赖这些工具实现。

**Architecture:** `eino-tools` 负责承载通用工具和共享 helper，命名以当前工具体系为准；`zwell_template` 只保留装配器职责，向 `eino-tools` 注入缓存、路径白名单、Cloudflare 回调、浏览器抓取和命令执行能力。整个迁移按 TDD 推进：先在 `eino-tools` 写失败测试，再实现工具，再切换宿主接入。

**Tech Stack:** Go 1.26、CloudWeGo Eino、DuckDuckGo、Rod、Readability、testify

---

执行时必须遵循：@brainstorming、@writing-plans、@test-driven-development、@verification-before-completion。

### Task 1: 搭好 `eino-tools` 的当前命名结构与共享内部包

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/plans/2026-03-25-generic-tools-extraction-design.md`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/plans/2026-03-25-generic-tools-extraction-implementation-plan.md`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/fsutil/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/fsutil/path.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/editutil/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/editutil/patch.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/screenshotutil/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/internal/screenshotutil/region.go`

**Step 1: Write the failing test**

为 `internal/fsutil`、`internal/editutil`、`internal/screenshotutil` 分别添加测试，覆盖：
- `base_dir` 解析
- 路径逃逸阻断与白名单放行
- `apply_patch` 文本解析
- 截图区域解析和输出路径后缀规范化

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/... -count=1`
Expected: FAIL，提示新内部包和函数尚不存在。

**Step 3: Write minimal implementation**

把当前 `zwell_template/services/builtin_tools_path.go`、`builtin_tools_edit.go` 中的通用 helper 拆到 `eino-tools/internal/*`，去掉对宿主类型的引用。

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./internal/... -count=1`
Expected: PASS。

### Task 2: 先完成命名已存在的 `web_search`

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/websearch/tool.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/websearch/tool_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/go.mod`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_web_search.go`

**Step 1: Write the failing test**

补充 `websearch` 测试，确保工具名、空 `query` 错误、cache hit 与搜索 runner 透传符合当前约束；在 `zwell_template` 写 adapter 测试验证仍返回 `web_search`。

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./websearch -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**

保持 package 为 `websearch`，但确认对外工具名稳定为 `web_search`，并在 `zwell_template` 通过 `replace` 直接使用该实现。

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./websearch -count=1`
Expected: PASS。

### Task 3: 新增 `webfetch` 并切换 `web_fetch`

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/webfetch/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/webfetch/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/webfetch/tool_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_test.go`

**Step 1: Write the failing test**

覆盖：
- 工具名必须是 `web_fetch`
- 参数 `url/render`
- cache key 兼容
- Cloudflare challenge handler 触发
- 可选浏览器抓取回调优先级

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./webfetch -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**

从当前 `CreateFetchURLTool` / `ExecuteFetchURL` 抽出通用逻辑，命名改为当前体系 `web_fetch`，把宿主差异全部改成 config 注入。

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./webfetch -count=1`
Expected: PASS。

### Task 4: 新增 `exec` 并切换 `exec`

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/exec/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/exec/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/exec/tool_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_bash.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/cloudflare_domain_guard_test.go`

**Step 1: Write the failing test**

覆盖：
- 工具名必须是 `exec`
- JSON 返回 payload 与当前结构一致
- `cwd/base_dir` 路径约束
- 超时和输出截断
- Cloudflare 保护域名阻断与回调

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./exec -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**

从当前 bash 工具抽通用逻辑到 `exec` package；当前项目仅保留默认 `base_dir`、白名单和人工验证回调的装配。

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./exec -count=1`
Expected: PASS。

### Task 5: 新增文件系统工具 `read/write/edit/ls/tree/glob/grep`

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/read/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/read/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/read/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/write/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/write/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/write/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/edit/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/edit/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/edit/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ls/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ls/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ls/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/tree/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/tree/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/tree/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/glob/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/glob/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/glob/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/grep/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/grep/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/grep/tool_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_read.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_write.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_edit_simple.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_ls.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_tree.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_glob.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_grep.go`

**Step 1: Write the failing test**

每个工具至少覆盖一条当前行为主链路：
- `read`: 编码探测与分页
- `write`: 自动建目录并写文件
- `edit`: search-replace 与 patch
- `ls`: 相对路径输出
- `tree`: depth / include / exclude / truncation
- `glob`: 基于 base path 匹配
- `grep`: `files_with_matches` / `content` / `count`

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./read ./write ./edit ./ls ./tree ./glob ./grep -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**

把当前文件系统工具逐个迁到 `eino-tools`，统一依赖 `internal/fsutil` 和 `internal/editutil`。

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./read ./write ./edit ./ls ./tree ./glob ./grep -count=1`
Expected: PASS。

### Task 6: 新增 `pythonrunner` 与 `screenshot`

**Files:**
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/pythonrunner/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/pythonrunner/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/pythonrunner/tool_test.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/screenshot/doc.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/screenshot/tool.go`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/screenshot/tool_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_python.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_screenshot.go`

**Step 1: Write the failing test**

覆盖：
- `python_runner`：空代码、requirements 类型校验、命令执行 payload
- `screenshot`：工具名、区域校验、输出 payload、data URL 省略逻辑、注入 command builder / runner

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./pythonrunner ./screenshot -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**

把当前 Python 与截图工具迁到 `eino-tools`，把外部命令和临时目录相关能力抽成可注入依赖。

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./pythonrunner ./screenshot -count=1`
Expected: PASS。

### Task 7: 在 `zwell_template` 切换到 `eino-tools`

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/go.mod`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_registry.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/chat_service_tools.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/tool_service.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_screenshot_test.go`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/services/builtin_tools_edit_simple_test.go`

**Step 1: Write the failing test**

补 adapter 测试，确认当前工具 ID、`Info().Name`、路径注入、浏览器抓取注入、Cloudflare 回调注入与截图 runner 注入仍正常。

**Step 2: Run test to verify it fails**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template && go test ./services -run 'TestBuiltinToolsManager_|TestScreenshotTool_|TestEditFileSimpleTool_' -count=1`
Expected: FAIL。

**Step 3: Write minimal implementation**

1. 在 `go.mod` 增加 `github.com/LubyRuffy/eino-tools`，本地联调用 `replace => ../eino-tools`
2. 让 `CreateXxxTool` 直接构造 `eino-tools` 对应 package
3. 清理不再需要的本地实现与 helper，仅保留宿主 adapter

**Step 4: Run test to verify it passes**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template && go test ./services -run 'TestBuiltinToolsManager_|TestScreenshotTool_|TestEditFileSimpleTool_' -count=1`
Expected: PASS。

### Task 8: 文档、变更记录和完整验证

**Files:**
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/README.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/ARCHITECTURE.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/AGENTS.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/CHANGELOG.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/docs/TESTING.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/README.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/ARCHITECTURE.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/AGENTS.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/CHANGELOG.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/docs/TESTING.md`
- Modify: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/docs/design.md`
- Create: `/Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template/docs/history/20260325+generic-tools-extract-to-eino-tools.md`

**Step 1: Run `eino-tools` full verification**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./... -count=1`
Expected: PASS。

**Step 2: Run `zwell_template` focused verification**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template && go test ./services -count=1`
Expected: PASS；如存在无关历史失败，必须明确列出。

**Step 3: Run current-repo full verification**

Run: `cd /Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template && go test ./... -count=1`
Expected: PASS；如存在无关历史失败，必须明确列出。

**Step 4: Update docs**

同步更新两边 README、ARCHITECTURE、AGENTS、TESTING、CHANGELOG，并在 `zwell_template/docs/design.md` 追加一句话、写历史记录。

**Step 5: Final verification**

重新运行：
- `cd /Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools && go test ./... -count=1`
- `cd /Users/zhaowu/go/src/github.com/LubyRuffy/zwell_template && go test ./services -count=1`

Expected: PASS。
