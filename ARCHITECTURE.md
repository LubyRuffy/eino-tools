# ARCHITECTURE

## 系统整体架构

仓库按“导出工具 package + MCP server + 内部共享能力”组织：

- `websearch`：搜索工具
- `webfetch`：当前命名的网页抓取与可读性提取工具
- `browserbin`：跨平台解析 Chromium 系浏览器可执行文件路径
- `exec`：当前命名的命令执行工具
- `read` / `write` / `edit`：结构化文件读写工具
- `ls` / `tree` / `glob` / `grep`：文件系统检索工具
- `pythonrunner`：Python 隔离执行工具
- `screenshot`：截图工具
- `cmd/mcpserver`：可直接启动的 MCP server 入口
- `internal/mcpserver`：Eino tools 到 MCP tools 的桥接、HTTP 端点挂载与 server 构造
- `internal/shared`：参数解析、输出缓冲、工具错误处理
- `internal/cloudflare`：Cloudflare 检测与保护域名状态
- `internal/fsutil`：`base_dir`、路径解析、显示路径，以及仓库级 walk 的默认 skip / gitignore 策略
- `internal/editutil`：apply-patch 文本解析与替换
- `internal/screenshotutil`：截图路径、区域与 mime 处理

## 核心模块

### websearch

封装 DuckDuckGo 文本搜索，并支持缓存与 HTTP client 注入。

当通过 `cmd/mcpserver` 运行时，代理配置会先在 `internal/mcpserver` 收敛，再把共享 `HTTPClient` 注入给该工具。

当作为 package 被宿主直接调用时，推荐宿主通过 `netproxy.Config` 注入统一代理配置；若未显式传 `HTTPClient`，工具会自动据此构造共享 client。

### browserbin

公开包，解析 Chromium 系浏览器可执行文件路径，供 Rod 等浏览器链路复用：

- 显式路径优先校验
- 未设置时按 Brave > Edge > Chrome > Chromium 跨平台探测
- 全部未命中返回空路径，由调用方回退 Rod 默认

### webfetch

封装 HTTP 抓取、Readability 文本提取、可选渲染抓取与 Cloudflare 挑战回调。

当通过 `cmd/mcpserver` 运行时，它与 `websearch` 共享同一个代理感知 `HTTPClient`，保证 `web_search` / `web_fetch` 的出网策略一致。

当通过 `cmd/mcpserver` 运行时，默认 `render=true` 也会把同一套代理配置映射到 Rod launcher 的浏览器参数。

当作为 package 被宿主直接调用时，若宿主提供 `netproxy.Config`，普通 HTTP 抓取链路与默认 `render=true` 都会复用同一套代理配置；若只给了自定义 `HTTPClient`，默认浏览器渲染链路仍无法可靠推断代理。

默认 `render=true` 通过 `browserbin.Resolve` 选择本机浏览器：`BrowserBin` 显式路径优先，否则自动探测；探测失败则不设 `Bin`，回退 Rod 默认。

宿主可以只注入 `HTTPClient`，也可以额外注入：

- `ProxyConfig`
  - 推荐的统一代理配置入口；可同时覆盖普通抓取与默认 `render=true`
- `BrowserBin`
  - 可选；显式指定 Chromium 系浏览器可执行文件。无效路径报错；空值走 `browserbin` 自动探测
- `Cache`
- `HeaderProvider`
- `CookieProvider`
- `HTMLFetcher`
- `RenderFetcher`
  - `render=true` 的宿主覆盖点；若未注入则保持工具内建的 Rod 渲染实现，保证兼容
- `ChallengeDetector`
- `ChallengeHandler`

### exec

封装非交互 bash 执行（`--noprofile --norc -c`，不跟随 `$SHELL`、不读 rc）、工作目录解析、输出截断、超时与 Cloudflare 保护域名拦截。

超时只作用于仍在运行的命令进程组；shell 已经退出后留下的后台任务不会再被收割。`WaitDelay` 避免后台子进程占用 stdout/stderr 管道导致 `Wait` 一直挂到超时。超时数值只由 `timeout_ms` 决定，不按命令文本做特例。

宿主可用 `WithOutputListener(ctx, fn)` 在 `Wait` 之前收到 stdout/stderr 分片。回调在进程的写路径上，不得久阻塞。每次回调拿到的是独立拷贝。未设置 listener 时行为与原来一致。

宿主可通过 `Config` 注入：

- `DefaultBaseDir`
- `ProtectedDomains`
- `ChallengeHandler`
- `ShellPath`（测试或明确覆盖时才需要；默认解析 bash）

`base_dir` 仅作为相对路径解析锚点，解析后的路径不要求位于 `base_dir` 内；历史上 `Config` 的 `AllowedPaths` 字段已随路径限制一起移除。

### 文件与截图工具

`read/write/edit/ls/tree/glob/grep/screenshot` 共享 `internal/fsutil`、`internal/editutil` 与 `internal/screenshotutil`，把相对路径解析、patch 解析和平台差异统一收敛到内部 helper。

`read` 用文件前 4096 字节探测编码：样本允许末尾被窗口截断的 UTF-8，合法 UTF-8（含 BOM，读取时跳过 BOM 字节）输出 `encoding=utf-8`；裁尾后仍非法 UTF-8 才尝试 GB18030，且解码结果不得含替换符；否则回退 `iso-8859-1`。探测结论用于整文件解码，不会把合法 UTF-8 再交给 GB18030。

`edit` 只有两种模式：`search_block` 与 `replace_block` 必须成对出现（`replace_block` 为空串表示删除），或提供 `patch`。两者都给时 search/replace 优先。缺字段、类型不是 string、search 找不到，错误文案可区分，并会列出实际收到的键。

`write` 的 `file_path` 与 `content` 都必填。省略 `content` 是缺参，空串是合法写空文件；非 string 报类型错误；收到 `contents` / `path` 这类近似键时点名正确字段。相对路径相对 `base_dir`/`DefaultBaseDir` 解析，不是进程 cwd。成功前会回读比对字节，成功句带解析后路径和字节数。

`grep` 的仓库级搜索会跳过 `.git` / `node_modules` / `.worktrees` / `dist` / `vendor` 这类目录名，尊重 `.gitignore`，并在超长行、二进制或不可读文件上跳过而不是让整次 Walk 失败。命中有上限。用户显式传入的搜索根（例如 `path=dist`）本身不会被默认 skip。

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
