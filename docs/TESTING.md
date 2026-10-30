# TESTING

## 目标

确保当前正式工具和 `cmd/mcpserver` 在代码、传输挂载和桥接层上保持稳定。

## 测试命令

```bash
go test ./... -count=1
go test ./browserbin -count=1
go test ./netproxy -count=1
go test ./websearch -count=1
go test ./webfetch -count=1 -race
go test ./webfetch -count=1 -coverprofile=/tmp/eino-tools-webfetch.cover
go tool cover -func=/tmp/eino-tools-webfetch.cover
go test ./webfetch ./exec ./read ./write ./edit ./ls ./tree ./glob ./grep ./pythonrunner ./screenshot -count=1
go test ./internal/mcpserver ./cmd/mcpserver -count=1
go build ./cmd/mcpserver
```

## 分层策略

1. `internal` 层测试通用 helper 与 challenge 检测
2. package 层测试参数校验、核心行为和错误处理
3. `internal/mcpserver` 测试工具注册、schema 桥接和传输挂载
4. 宿主仓库再补 adapter 回归测试

## 当前覆盖

- `netproxy`：显式配置优先级、HTTP client 代理行为、Chromium 代理参数映射
- `websearch`：空 query、缓存命中、工具名
- `browserbin`：显式路径存在/不存在、Brave 优先于 Chrome、探测未命中返回空、darwin/linux/windows 候选顺序
- `webfetch`：空 URL、默认请求头/cookie 注入、注入 HTML fetcher、默认 render 代理 launcher、`BrowserBin` 注入与无效路径、render/plain 缓存隔离、Cloudflare fallback、challenge handler 回调重试
- `exec`：正常执行、管道/换行保真、不跟随 `$SHELL`、不读 rc、`cwd` 相对 `base_dir` 解析且允许落在边界外、超时杀前台进程组但不误杀已返回的后台任务、受保护域名拦截、`WithOutputListener` 在 `Wait` 前收到 stdout/stderr 分片
- `read/write/edit/ls/tree/glob/grep`：路径解析允许落在 `base_dir` 边界外、基本文件操作和 patch/glob/grep 语义；`Config.AllowedPaths` / `ResolvePathWithin` 第三参已移除
- `read`：编码探测覆盖 UTF-8 窗口截断、无 BOM 短 CJK、UTF-8 BOM 不进正文、真 GB18030（含窗口切在多字节中间）、非中文页回退 iso-8859-1、ASCII / 空文件；工具描述不含测试夹具词
- `write`：缺 `content` 失败且不改原文件、空串写成空文件、非 string 类型错误、`contents`/`path` 近似键提示正确字段名、写后回读字节一致、相对路径落在 `DefaultBaseDir` 而非进程 cwd；成功句带解析后路径和字节数；工具描述不含测试夹具词
- `edit`：缺成对字段、空 replace 删除、非 string 类型错误、未知字段列出收到的键、search 找不到、search/replace 优先于 patch；Info 写清两种模式
- `internal/shared`：`LookupStringParam` 区分 omitted / 空串 / 非 string；`GetStringParam` 对非 string 仍静默变空
- `grep`：超长行不 abort、默认跳过 `dist` / `.worktrees` 等目录名、尊重 `.gitignore`、命中上限、显式 `path` 仍可搜 skip 目录本身
- `internal/fsutil`：walk skip 目录名精确匹配、gitignore 否定规则、显式搜索根不 skip
- `pythonrunner`：空代码、requirements、执行结果结构
- `screenshot`：路径规范化、区域解析、命令选择和 data URL
- `internal/mcpserver`：正式工具列表、Eino 到 MCP 的 arguments 透传、`/sse` 和 `/mcp` 挂载、server 对客户端的工具列表暴露
- `internal/mcpserver`：网络工具统一代理配置的 CLI 优先级与环境变量回退
- `cmd/mcpserver`：默认参数、合法传输值、非法传输值校验、代理 flag 解析

### web_fetch bot wall 回归（D1–D11）

测试位于 `webfetch/tool_test.go`，只使用 `httptest` 或注入 fetcher，不请求真实站点，也不启动外部浏览器：

旧的真实站点浏览器测试以 `integration` build tag 隔离，不参与默认测试命令及本次离线验收。

- D1–D3：DataDome 401/403 响应升级一次 `RenderFetcher`；仅 `X-DataDome` 而无 `Server` 也可识别。
- D4：无 DataDome 响应信号的普通 401/403 保留 HTTP 错误，渲染调用为零。
- D5–D6：自动升级失败说明 HTTP 被拦截且浏览器已尝试，只渲染一次；显式 `render=true` 直接渲染一次，不先走 HTTP、不重复升级。
- D7：具有稳定挑战资源特征的 HTML/Markdown 不能成功返回或缓存；普通文章提及厂商名字不误判。
- D8–D9：Cloudflare 403 保留注入 `BrowserFetch` fallback；HTTP 200 JavaScript 空壳保留 `ShouldRetryWithRender` 重试。
- D10：升级成功后相同 URL 的 `render=false` 可命中缓存；挑战页无成功缓存，显式渲染与默认模式缓存键仍隔离。
- D11：工具描述及 `render` 参数说明默认先 HTTP、识别 bot wall 后自动升级一次、显式 `render=true` 直接渲染；描述与错误保持泛化。

覆盖率验收关注新增响应识别、挑战页检测与「HTTP bot wall → 一次渲染」分支，新增语句覆盖率须达到 90%；不要用包含未启动 Rod 路径的全包覆盖率代替该分支的证据。

## 手工冒烟验证

### stdio

```bash
go run ./cmd/mcpserver --transport stdio
```

### HTTP

```bash
go run ./cmd/mcpserver --transport http --addr :8080
```

### 代理

```bash
HTTP_PROXY=http://127.0.0.1:7890 \
HTTPS_PROXY=http://127.0.0.1:7890 \
NO_PROXY=localhost \
go test ./cmd/mcpserver ./internal/mcpserver -count=1
```

检查端点：

- `http://127.0.0.1:8080/sse`
- `http://127.0.0.1:8080/mcp`

### 排障链路

`cmd/mcpserver` 会把 MCP server 事件和工具调用日志输出到 stderr。出现问题时优先保留：

- `session_id`
- `tool`
- 启动参数
