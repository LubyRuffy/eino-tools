# CONFIG

## `cmd/mcpserver` Flags

### `--transport`

- 默认值：`all`
- 可选值：`stdio`、`http`、`all`

说明：

- `stdio` 只启标准输入输出传输
- `http` 只启 HTTP 服务，并同时挂 `/sse` 与 `/mcp`
- `all` 同时启 `stdio` 和 HTTP

### `--addr`

- 默认值：`:8080`

说明：

- 仅在 `http` 或 `all` 模式下生效

### `--base-dir`

- 默认值：当前工作目录

说明：

- 统一传给 `exec`、`read`、`write`、`edit`、`ls`、`tree`、`glob`、`grep`、`screenshot` 等依赖文件系统路径的工具
- 这些工具都会用它解析相对路径，但不再要求最终路径必须位于 `base_dir` 内
- `base_dir` 现在主要用于给相对路径提供稳定锚点

### `--http-proxy`

- 默认值：空

说明：

- 仅影响 `web_search` 与 `web_fetch`
- 显式设置时优先于环境变量 `HTTP_PROXY`

### `--https-proxy`

- 默认值：空

说明：

- 仅影响 `web_search` 与 `web_fetch`
- 显式设置时优先于环境变量 `HTTPS_PROXY`

### `--no-proxy`

- 默认值：空

说明：

- 仅影响 `web_search` 与 `web_fetch`
- 用逗号分隔不走代理的 host / domain
- 显式设置时优先于环境变量 `NO_PROXY`

### `--name`

- 默认值：`eino-tools`

说明：

- MCP server 的实现名

### `--version`

- 默认值：`dev`

说明：

- MCP server 的实现版本

## 环境变量回退

当对应 CLI 参数未设置时，`cmd/mcpserver` 会按以下环境变量回退：

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `NO_PROXY`

这些变量同样只影响 `web_search` 与 `web_fetch`。

## 作为 Go Package 使用

`docs/CONFIG.md` 上面的配置项只适用于 `cmd/mcpserver`。

如果上层业务直接调用 `websearch.New(...)` 或 `webfetch.New(...)`：

- 推荐通过 `netproxy.Config` 作为统一代理配置入口
- 通过 `websearch.Config.ProxyConfig` 与 `webfetch.Config.ProxyConfig` 注入
- 这样普通 HTTP 抓取和默认 `render=true` 都会复用同一套代理配置
- 若宿主必须自定义 `HTTPClient`，建议同时把同一份 `ProxyConfig` 也传给工具
- 只有在宿主要完全接管浏览器实现时，才需要显式注入 `RenderFetcher` 或 `BrowserFetch`
