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

如果需要统一配置网络代理，推荐同时引入 `netproxy`：

```bash
go get github.com/LubyRuffy/eino-tools/netproxy
```

### 启动 MCP Server

```bash
go run ./cmd/mcpserver
```

默认行为：

- `--transport=all`
- HTTP 监听 `:8080`
- 网络工具默认直连；可通过 `--http-proxy`、`--https-proxy`、`--no-proxy` 或对应环境变量启用代理
- 这套代理配置会同时覆盖 `web_search`、`web_fetch` 普通抓取和 `web_fetch` 默认 `render=true` 渲染抓取
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

通过显式参数配置网络代理：

```bash
go run ./cmd/mcpserver \
  --transport http \
  --addr :8080 \
  --http-proxy http://127.0.0.1:7890 \
  --https-proxy http://127.0.0.1:7890 \
  --no-proxy localhost,127.0.0.1,.internal
```

也可以直接复用环境变量：

```bash
HTTP_PROXY=http://127.0.0.1:7890 \
HTTPS_PROXY=http://127.0.0.1:7890 \
NO_PROXY=localhost,127.0.0.1,.internal \
go run ./cmd/mcpserver
```

优先级规则：

1. `--http-proxy` / `--https-proxy` / `--no-proxy`
2. `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`
3. 未设置时直连

## 使用示例

### 在宿主项目中注册工具

```go
fetchTool, err := webfetch.New(webfetch.Config{})
if err != nil {
	panic(err)
}
_ = fetchTool
```

### 在宿主项目中为 pkg 注入代理

业务系统直接使用 package 时，不走 `cmd/mcpserver` 的 CLI 参数。若希望普通 HTTP 抓取和默认 `render=true` 都复用同一套代理，推荐直接注入 `netproxy.Config`。

统一代理配置示例：

```go
package main

import (
	"context"

	"github.com/LubyRuffy/eino-tools/netproxy"
	"github.com/LubyRuffy/eino-tools/webfetch"
	"github.com/LubyRuffy/eino-tools/websearch"
)

func main() {
	proxyCfg := netproxy.Config{
		HTTPProxy:  "http://127.0.0.1:7890",
		HTTPSProxy: "http://127.0.0.1:7890",
		NoProxy:    "localhost,127.0.0.1,.internal",
	}

	searchTool, err := websearch.New(context.Background(), websearch.Config{
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		panic(err)
	}

	fetchTool, err := webfetch.New(webfetch.Config{
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		panic(err)
	}

	_, _ = searchTool, fetchTool
}
```

显式复用环境变量示例：

```go
package main

import (
	"context"
	"os"

	"github.com/LubyRuffy/eino-tools/netproxy"
	"github.com/LubyRuffy/eino-tools/webfetch"
	"github.com/LubyRuffy/eino-tools/websearch"
)

func main() {
	proxyCfg := netproxy.Resolve(netproxy.Config{}, os.Getenv)

	searchTool, err := websearch.New(context.Background(), websearch.Config{
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		panic(err)
	}

	fetchTool, err := webfetch.New(webfetch.Config{
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		panic(err)
	}

	_, _ = searchTool, fetchTool
}
```

如果宿主必须自己控制 HTTP transport，也可以继续直接注入 `HTTPClient`，但建议同时把同一份 `ProxyConfig` 也传进去：

```go
package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/LubyRuffy/eino-tools/netproxy"
	"github.com/LubyRuffy/eino-tools/webfetch"
	"github.com/LubyRuffy/eino-tools/websearch"
	"golang.org/x/net/http/httpproxy"
)

func main() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = (&httpproxy.Config{
		HTTPProxy:  os.Getenv("HTTP_PROXY"),
		HTTPSProxy: os.Getenv("HTTPS_PROXY"),
		NoProxy:    os.Getenv("NO_PROXY"),
	}).ProxyFunc()

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	proxyCfg := netproxy.Resolve(netproxy.Config{}, os.Getenv)

	searchTool, err := websearch.New(context.Background(), websearch.Config{
		HTTPClient:  httpClient,
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		panic(err)
	}

	fetchTool, err := webfetch.New(webfetch.Config{
		HTTPClient:  httpClient,
		ProxyConfig: proxyCfg,
	})
	if err != nil {
		panic(err)
	}

	_, _ = searchTool, fetchTool
}
```

注意：

1. 推荐给 `web_search` 和 `web_fetch` 传同一个 `netproxy.Config`
2. 这样普通 HTTP 抓取和默认 `render=true` 都会复用同一套代理配置
3. 如果只传 `HTTPClient` 而不传 `ProxyConfig`，默认浏览器渲染链路无法可靠推断代理
4. 如果业务需要完全接管浏览器实现，仍可显式注入 `RenderFetcher` 或 `BrowserFetch`

宿主若希望接管 `render=true` 的执行后端，可注入 `RenderFetcher`；未注入时工具会继续使用库内建的默认 render 实现。

`cmd/mcpserver` 会为 `web_search` 与 `web_fetch` 统一构造共享代理配置，并把它同时用于网络 `HTTPClient` 和默认 Rod 浏览器渲染链路。

### 作为 MCP HTTP 服务运行

```bash
go run ./cmd/mcpserver --transport http --addr :8080 --base-dir /workspace
```

然后让客户端连接：

- SSE: `http://127.0.0.1:8080/sse`
- Streamable HTTP: `http://127.0.0.1:8080/mcp`

## 常见使用方式

1. 直接把正式工具注册到 Eino agent。
2. 在业务仓库中包一层 adapter，注入缓存、cookie 源、路径策略、统一 `netproxy.Config` 与 challenge 回调。
3. 直接运行 `cmd/mcpserver`，把当前正式工具暴露给 MCP client。
4. 用宿主仓库的 `replace` 指向本地工作目录做联调，再切正式 tag。

## 常见排障方式

`cmd/mcpserver` 会把 MCP server 日志和工具调用日志写到 stderr。排障时优先保留：

- `session_id`
- `tool`
- 启动参数：`transport`、`addr`、`base-dir`
- 代理参数：`http-proxy`、`https-proxy`、`no-proxy`

拿到 `session_id` 后，可以快速对照同一会话内的连接、调用和失败日志。
