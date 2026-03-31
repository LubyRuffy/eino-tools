# Eino Tools

`eino-tools` 是一个面向 Go / CloudWeGo Eino 的可复用工具仓库，提供两类能力：

- 可直接返回 Eino `tool.BaseTool` / `tool.InvokableTool` 的通用工具包
- 可直接启动的 `cmd/mcpserver`，基于官方 `github.com/modelcontextprotocol/go-sdk` 暴露 MCP tools

## 当前正式工具

- `web_search`
- `web_fetch`
- `exec`
- `read`
- `write`
- `edit`
- `ls`
- `tree`
- `glob`
- `grep`
- `python_runner`
- `screenshot`

兼容别名 `fetchurl` / `bashcmd` 已删除。

## 快速启动

### 作为 Go 模块使用

```bash
go get github.com/LubyRuffy/eino-tools
```

```go
searchTool, err := websearch.New(context.Background(), websearch.Config{})
if err != nil {
	panic(err)
}
_ = searchTool
```

```go
execTool, err := exec.New(exec.Config{
	DefaultBaseDir: ".",
})
if err != nil {
	panic(err)
}
_ = execTool
```

`base_dir` 现在统一只作为相对路径解析锚点使用；`exec` 和文件读写类工具都允许最终路径落在 `base_dir` 之外。

### 启动 MCP Server

```bash
go run ./cmd/mcpserver
```

默认行为：

- `--transport=all`
- HTTP 监听 `:8080`
- 同时提供：
  - `stdio`
  - `SSE` -> `/sse`
  - `streamable HTTP` -> `/mcp`

只启 `stdio`：

```bash
go run ./cmd/mcpserver --transport stdio
```

只启 HTTP：

```bash
go run ./cmd/mcpserver --transport http --addr :8080
```

## 使用示例

### 在宿主项目中注册工具

```go
fetchTool, err := webfetch.New(webfetch.Config{})
if err != nil {
	panic(err)
}
_ = fetchTool
```

宿主若希望接管 `render=true` 的执行后端，可注入 `RenderFetcher`；未注入时工具会继续使用库内建的默认 render 实现。

### 作为 MCP HTTP 服务运行

```bash
go run ./cmd/mcpserver --transport http --addr :8080 --base-dir /workspace
```

然后让客户端连接：

- SSE: `http://127.0.0.1:8080/sse`
- Streamable HTTP: `http://127.0.0.1:8080/mcp`

## 常见使用方式

1. 直接把正式工具注册到 Eino agent。
2. 在业务仓库中包一层 adapter，注入缓存、cookie 源、路径策略与 challenge 回调。
3. 直接运行 `cmd/mcpserver`，把当前正式工具暴露给 MCP client。
4. 用宿主仓库的 `replace` 指向本地工作目录做联调，再切正式 tag。

## 常见排障方式

`cmd/mcpserver` 会把 MCP server 日志和工具调用日志写到 stderr。排障时优先保留：

- `session_id`
- `tool`
- 启动参数：`transport`、`addr`、`base-dir`

拿到 `session_id` 后，可以快速对照同一会话内的连接、调用和失败日志。
