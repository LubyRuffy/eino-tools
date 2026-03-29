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

### `--name`

- 默认值：`eino-tools`

说明：

- MCP server 的实现名

### `--version`

- 默认值：`dev`

说明：

- MCP server 的实现版本
