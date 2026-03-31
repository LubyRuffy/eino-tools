# ARCHITECTURE

## 系统整体架构

仓库按“导出工具 package + MCP server + 内部共享能力”组织：

- `websearch`：搜索工具
- `webfetch`：当前命名的网页抓取与可读性提取工具
- `exec`：当前命名的命令执行工具
- `read` / `write` / `edit`：结构化文件读写工具
- `ls` / `tree` / `glob` / `grep`：文件系统检索工具
- `pythonrunner`：Python 隔离执行工具
- `screenshot`：截图工具
- `cmd/mcpserver`：可直接启动的 MCP server 入口
- `internal/mcpserver`：Eino tools 到 MCP tools 的桥接、HTTP 端点挂载与 server 构造
- `internal/shared`：参数解析、输出缓冲、工具错误处理
- `internal/cloudflare`：Cloudflare 检测与保护域名状态
- `internal/fsutil`：`base_dir`、白名单路径与显示路径处理
- `internal/editutil`：apply-patch 文本解析与替换
- `internal/screenshotutil`：截图路径、区域与 mime 处理

## 核心模块

### websearch

封装 DuckDuckGo 文本搜索，并支持缓存与 HTTP client 注入。

### webfetch

封装 HTTP 抓取、Readability 文本提取、可选渲染抓取与 Cloudflare 挑战回调。

宿主可以只注入 `HTTPClient`，也可以额外注入：

- `Cache`
- `HeaderProvider`
- `CookieProvider`
- `HTMLFetcher`
- `RenderFetcher`
  - `render=true` 的宿主覆盖点；若未注入则保持工具内建的 Rod 渲染实现，保证兼容
- `ChallengeDetector`
- `ChallengeHandler`

### exec

封装 `/bin/bash -c` 执行、工作目录解析、输出截断、超时与 Cloudflare 保护域名拦截。

宿主可通过 `Config` 注入：

- `DefaultBaseDir`
- `AllowedPaths`
- `ProtectedDomains`
- `ChallengeHandler`

### 文件与截图工具

`read/write/edit/ls/tree/glob/grep/screenshot` 共享 `internal/fsutil`、`internal/editutil` 与 `internal/screenshotutil`，把相对路径解析、patch 解析和平台差异统一收敛到内部 helper。

### MCP Server

`cmd/mcpserver` 使用官方 `github.com/modelcontextprotocol/go-sdk` 启动 MCP server。

`internal/mcpserver` 负责：

- 构造当前正式工具集
- 读取每个工具的 `Info()` 并转成 MCP tool schema
- 调用每个工具的 `InvokableRun()` 执行实际逻辑
- 挂载 `/sse` 与 `/mcp`
- 启动 `stdio` 传输

## 请求与数据流

1. 调用方构造 `Config`
2. 调用 `New(...)` 返回 Eino `tool.BaseTool`
3. 工具执行时通过共享 helper 解析参数与格式化结果
4. 需要宿主参与的行为通过 callback/interface 回调给宿主

## MCP 请求流

1. `cmd/mcpserver` 解析启动参数并构造 `internal/mcpserver.Config`
2. `internal/mcpserver.BuildServer(...)` 创建官方 MCP server
3. `internal/mcpserver.NewToolset(...)` 构造 12 个正式工具
4. `internal/mcpserver.ToMCPTool(...)` 把 Eino tool 转成 MCP tool handler
5. 传输层通过 `stdio`、`/sse` 或 `/mcp` 接收请求
6. handler 将 MCP `arguments` 原样序列化后调用对应工具的 `InvokableRun(...)`
7. 工具结果作为 MCP text content 返回；若结果是 JSON object，会同时填入 `StructuredContent`
