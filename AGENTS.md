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

## 经验教训：Issue #1 / #2

- 现象：`grep` 被构建产物超长行整次打死；`exec` 跟 `$SHELL` 读 rc，特殊字符和后台任务不可用。
- 根因：Walk 把读文件错误当致命错误；exec 把用户交互 shell 当工具运行时，且 `Wait` 会因后台子进程占用 stdout 管道而挂死，超时再 `Kill(-pgid)`。
- 排查路径：先在干净 worktree 用失败测试复现，不要拿主工作区未提交的 `eval "$1"` 当修复。
- 修复要点：walk skip 用目录名 + gitignore 通用策略；exec 固定非交互 bash；`WaitDelay` + `ErrWaitDelay` 表示 shell 已退出；超时只看 `timeout_ms`，禁止 `strings.Contains(command)` 特例。
- 验证方式：`go test ./grep ./exec ./internal/fsutil -count=1`
