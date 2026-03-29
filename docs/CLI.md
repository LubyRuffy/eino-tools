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
