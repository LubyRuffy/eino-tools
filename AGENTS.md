# AGENTS.md

本文档描述 AI 编码代理在 `eino-tools` 仓库中的开发约定。

## 项目开发规则

- 所有新增和修改的导出能力必须带测试
- 对外 API 以稳定、可复用、低宿主耦合为第一优先级
- 不允许把宿主项目的业务类型直接引入本仓库
- 保持工具名、参数名和关键返回结构稳定
- 单文件不超过 1000 行

## 构建、运行、测试命令

```bash
go test ./... -count=1
go test ./websearch -count=1
go test ./internal/mcpserver ./cmd/mcpserver -count=1
go build ./cmd/mcpserver
```

## 项目开发约定

- 共享逻辑优先进入 `internal/shared` 或 `internal/cloudflare`
- MCP server 相关桥接、传输和工具注册统一放在 `internal/mcpserver`
- 宿主差异通过 interface / callback 注入
- 文档是实现的一部分，变更时同步更新 `README.md`、`ARCHITECTURE.md`、`CHANGELOG.md`、`docs/TESTING.md`、`docs/CLI.md`、`docs/CONFIG.md`
