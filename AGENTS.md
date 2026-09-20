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

## 经验教训：Issue #4

- 现象：`read` 把无 BOM 的合法 UTF-8 中文标成 `encoding=gb18030`，正文是误解码乱码。
- 根因：4096 字节样本切在多字节 rune 中间导致 `utf8.Valid` 失败；`golang.org/x/text` 的 GB18030 decoder 几乎不返回 error，用 U+FFFD 吞非法字节，`transform.Bytes` 成功就被当成 GB18030。
- 排查路径：用 `4095` 个 ASCII + CJK 构造切点；对 `0xE9`/`0xFF` 等非中文页字节打印 `transform.Bytes` 的 error（通常是 nil）。
- 修复要点：先按 `utf8.FullRune` 允许末尾截断再判 UTF-8；GB18030 必须解码且无替换符，必要时丢 1–3 个尾字节；UTF-8 BOM 读取时 skip 3 字节；给 InvokableRun 用新的 decoder，不要复用探测时已经跑过的 transformer。
- 验证方式：`go test ./read -count=1 -race`

## 经验教训：Issue #5

- 现象：`edit` 缺 `search_block` 或空 `replace_block` 都报同一句 `either search_block/replace_block or patch is required`。
- 根因：模式开关用两个非空字符串 AND；`GetStringParam` 对非 string 静默变 `""`；默认 `ToolInvokableDefer` 把 error 塞进返回字符串，测试若断言 `err != nil` 会假绿/假红。
- 排查路径：用 `json.Marshal` 区分 omitted 与 `""`；不要改 `GetStringParam` 的静默行为。
- 修复要点：新增 `LookupStringParam`；按键是否存在分流；空 replace 走删除；类型错误点名字段；缺载荷列出收到的键；search/replace 优先于 patch。
- 验证方式：`go test ./edit ./internal/shared -count=1 -race`
