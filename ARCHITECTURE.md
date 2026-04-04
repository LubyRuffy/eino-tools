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

当通过 `cmd/mcpserver` 运行时，代理配置会先在 `internal/mcpserver` 收敛，再把共享 `HTTPClient` 注入给该工具。

当作为 package 被宿主直接调用时，推荐宿主通过 `netproxy.Config` 注入统一代理配置；若未显式传 `HTTPClient`，工具会自动据此构造共享 client。

### webfetch

封装 HTTP 抓取、Readability 文本提取、可选渲染抓取与 Cloudflare 挑战回调。

当通过 `cmd/mcpserver` 运行时，它与 `websearch` 共享同一个代理感知 `HTTPClient`，保证 `web_search` / `web_fetch` 的出网策略一致。

当通过 `cmd/mcpserver` 运行时，默认 `render=true` 也会把同一套代理配置映射到 Rod launcher 的浏览器参数。

当作为 package 被宿主直接调用时，若宿主提供 `netproxy.Config`，普通 HTTP 抓取链路与默认 `render=true` 都会复用同一套代理配置；若只给了自定义 `HTTPClient`，默认浏览器渲染链路仍无法可靠推断代理。

宿主可以只注入 `HTTPClient`，也可以额外注入：

- `ProxyConfig`
  - 推荐的统一代理配置入口；可同时覆盖普通抓取与默认 `render=true`
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
- 解析 CLI 与环境变量代理配置，并为网络工具构造共享 `ProxyConfig`
- 基于共享 `ProxyConfig` 构造 `HTTPClient`，并把浏览器代理参数注入默认 render 链路
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
3. `internal/mcpserver.NewToolset(...)` 按 CLI 参数优先、环境变量兜底的规则构造共享 `ProxyConfig`
4. `internal/mcpserver.NewToolset(...)` 基于共享 `ProxyConfig` 构造 HTTP client
5. `internal/mcpserver.NewToolset(...)` 构造 12 个正式工具，并把共享 client 和 `ProxyConfig` 注入 `web_search` / `web_fetch`
6. `internal/mcpserver.ToMCPTool(...)` 把 Eino tool 转成 MCP tool handler
7. 传输层通过 `stdio`、`/sse` 或 `/mcp` 接收请求
8. handler 将 MCP `arguments` 原样序列化后调用对应工具的 `InvokableRun(...)`
9. 工具结果作为 MCP text content 返回；若结果是 JSON object，会同时填入 `StructuredContent`
