# TESTING

## 目标

确保当前正式工具和 `cmd/mcpserver` 在代码、传输挂载和桥接层上保持稳定。

## 测试命令

```bash
go test ./... -count=1
go test ./websearch -count=1
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

- `websearch`：空 query、缓存命中、工具名
- `webfetch`：空 URL、默认请求头/cookie 注入、注入 HTML fetcher、Cloudflare fallback、challenge handler 回调重试
- `exec`：正常执行、`cwd` 相对 `base_dir` 解析且允许落在边界外、超时、受保护域名拦截
- `read/write/edit/ls/tree/glob/grep`：路径解析允许落在 `base_dir` 边界外、基本文件操作和 patch/glob/grep 语义
- `pythonrunner`：空代码、requirements、执行结果结构
- `screenshot`：路径规范化、区域解析、命令选择和 data URL
- `internal/mcpserver`：正式工具列表、Eino 到 MCP 的 arguments 透传、`/sse` 和 `/mcp` 挂载、server 对客户端的工具列表暴露
- `cmd/mcpserver`：默认参数、合法传输值、非法传输值校验

## 手工冒烟验证

### stdio

```bash
go run ./cmd/mcpserver --transport stdio
```

### HTTP

```bash
go run ./cmd/mcpserver --transport http --addr :8080
```

检查端点：

- `http://127.0.0.1:8080/sse`
- `http://127.0.0.1:8080/mcp`

### 排障链路

`cmd/mcpserver` 会把 MCP server 事件和工具调用日志输出到 stderr。出现问题时优先保留：

- `session_id`
- `tool`
- 启动参数
