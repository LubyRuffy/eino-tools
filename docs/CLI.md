# CLI

## `cmd/mcpserver`

`cmd/mcpserver` 用于启动 `eino-tools` 的 MCP server。

## 用法

```bash
go run ./cmd/mcpserver [flags]
```

## 常见命令

启动全部传输：

```bash
go run ./cmd/mcpserver
```

只启 `stdio`：

```bash
go run ./cmd/mcpserver --transport stdio
```

只启 HTTP：

```bash
go run ./cmd/mcpserver --transport http --addr :8080
```

指定工具工作目录：

```bash
go run ./cmd/mcpserver --base-dir /workspace
```

为网络工具配置代理：

```bash
go run ./cmd/mcpserver \
  --http-proxy http://127.0.0.1:7890 \
  --https-proxy http://127.0.0.1:7890 \
  --no-proxy localhost,127.0.0.1,.internal
```

复用环境变量代理：

```bash
HTTP_PROXY=http://127.0.0.1:7890 \
HTTPS_PROXY=http://127.0.0.1:7890 \
NO_PROXY=localhost,127.0.0.1,.internal \
go run ./cmd/mcpserver
```

代理优先级：

1. CLI 参数 `--http-proxy`、`--https-proxy`、`--no-proxy`
2. 环境变量 `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY`
3. 未设置时直连

这套代理配置会同时覆盖 `web_search`、`web_fetch` 普通抓取和 `web_fetch` 默认 `render=true` 渲染抓取。

## 传输说明

- `stdio`：适合本地 MCP client 直接拉起命令
- `http`：同时暴露 `/sse` 与 `/mcp`
- `all`：同时启动 `stdio` 和 HTTP

## 排障

server 日志输出到 stderr。排障时优先记录：

- `session_id`
- `tool`
- `transport`
- `addr`
- `base-dir`
- `http-proxy`
- `https-proxy`
- `no-proxy`
