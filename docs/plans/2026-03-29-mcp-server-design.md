# MCP Server Design

**目标**

为 `eino-tools` 增加 `cmd/mcpserver` 启动入口，使用官方 [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) 暴露仓库内正式工具为 MCP tools。启动后默认支持三类传输：

- `stdio`
- `SSE`
- `streamable HTTP`

同时删除兼容别名包 `fetchurl` 与 `bashcmd`，仓库对外只保留当前正式工具名。

## 本次范围

- 新增 `cmd/mcpserver`
- 新增 `internal/mcpserver`
- 通过官方 `go-sdk` 适配当前 Eino 工具
- 支持 `stdio`、`SSE`、`streamable HTTP`
- 删除 `fetchurl` 与 `bashcmd`
- 更新使用者文档、开发者文档与变更记录

## 非目标

- 不新增 prompts、resources、sampling 等非工具能力
- 不重写现有工具返回结构
- 不为每个工具新增单独配置文件
- 不保留旧别名包的兼容层

## 现状问题

1. 仓库当前只提供一组可复用的 Eino tools，没有直接可运行的 MCP server 入口。
2. 当前存在 `fetchurl` / `bashcmd` 两套旧命名兼容包，会让 MCP 对外暴露面重复。
3. 现有工具已经有稳定的 `Info()` 和 `InvokableRun()`，但缺少统一桥接层把它们注册到 MCP server。

## 设计原则

1. 对外工具名稳定
   - 只暴露正式工具名，不暴露别名
2. 最小侵入
   - MCP 层只做桥接，不改动工具参数和返回语义
3. 单入口统一管理
   - 通过一个 `cmd/mcpserver` 管理工具注册、传输和配置
4. 默认同时支持 MCP 主流传输
   - `stdio`、`SSE`、`streamable HTTP`
5. 文档与测试同步
   - 新增能力必须有测试和文档，不允许只加代码

## 总体结构

- `cmd/mcpserver/main.go`
  - 解析命令行参数
  - 构造工具集
  - 启动 `stdio` / HTTP 传输
- `internal/mcpserver/config.go`
  - 统一 MCP server 运行配置
- `internal/mcpserver/tools.go`
  - 构造正式工具集
- `internal/mcpserver/bridge.go`
  - Eino tool 到 MCP tool 的桥接
- `internal/mcpserver/http.go`
  - HTTP mux 与 `/sse`、`/mcp` 挂载
- `internal/mcpserver/server.go`
  - MCP server 创建与传输启动

## 工具暴露范围

MCP server 只暴露以下工具：

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

删除以下兼容别名：

- `fetchurl`
- `bashcmd`

## 传输模型

### 1. stdio

- 适合本地 MCP client 拉起命令
- 由 `--transport=stdio` 或 `--transport=all` 启动
- 使用官方 `go-sdk` 的 `StdioTransport`

### 2. SSE

- 挂载在 HTTP server 的 `/sse`
- 用于需要传统 SSE MCP 兼容的客户端
- 与 streamable HTTP 共用同一个工具注册和配置

### 3. streamable HTTP

- 挂载在 HTTP server 的 `/mcp`
- 作为现代 HTTP MCP 接入方式
- 与 SSE 共用同一个 `http.Server`

### 默认行为

`cmd/mcpserver` 默认启动 `--transport=all`：

- 启动 `stdio`
- 启动 HTTP server
- 在 HTTP server 上同时挂 `/sse` 与 `/mcp`

如果宿主环境只希望单传输运行，可通过命令行显式收窄。

## 配置设计

第一版通过命令行参数配置，不引入 YAML 文件：

- `--transport`
- `--addr`
- `--base-dir`
- `--name`
- `--version`

说明：

1. `base_dir` 统一传给文件系统与命令执行相关工具。
2. 更细粒度的白名单和安全配置后续再扩展；本次不做过度设计。

## 桥接策略

桥接层直接复用每个工具的：

- `Info(ctx)`：生成 MCP tool 的名称、描述、参数 schema
- `InvokableRun(ctx, argumentsJSON)`：执行工具调用

桥接流程：

1. MCP client 发起 tool call
2. MCP handler 拿到 `arguments`
3. arguments 序列化为 JSON 字符串
4. 调用 Eino tool 的 `InvokableRun`
5. 结果作为 MCP text content 返回

这样有三个好处：

1. 不复制参数定义
2. 不改现有工具返回格式
3. 现有工具测试基本可复用，不需要重写行为层

## 数据流

1. `main` 解析 flags
2. `internal/mcpserver` 构造正式工具集
3. bridge 把工具注册到 MCP server
4. `stdio` 或 HTTP transport 接收 MCP 请求
5. 请求转给具体 tool handler
6. handler 调用对应 Eino tool
7. 结果通过 MCP 响应返回

## 删除别名策略

直接删除：

- `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/fetchurl`
- `/Users/zhaowu/go/src/github.com/LubyRuffy/eino-tools/bashcmd`

并同步清理：

- `README.md`
- `ARCHITECTURE.md`
- `AGENTS.md`
- `CHANGELOG.md`
- 测试命令与示例中的旧路径

## 测试策略

### 单元测试

- 工具注册列表只包含正式工具名
- 不包含 `fetchurl` / `bashcmd`
- bridge 能把 MCP arguments 透传给 Eino tool
- HTTP mux 同时挂载 `/sse` 与 `/mcp`
- `stdio` transport 可被创建并联通 server

### 集成验证

- `go test ./... -count=1`
- `go build ./cmd/mcpserver`

## 风险与控制

### 1. Eino schema 到 MCP schema 的映射差异

控制策略：

- 先以当前工具真实使用到的基础类型映射为主
- 复杂 schema 保持保守映射
- 通过测试锁定行为

### 2. 三种传输同时运行的生命周期管理

控制策略：

- `stdio` 和 HTTP 启动逻辑分开封装
- `all` 模式只负责并行拉起与统一退出

### 3. 删除别名造成外部引用中断

控制策略：

- 文档中明确声明别名删除
- CHANGELOG 中明确记录 breaking change
- 当前仓库内部同步清理所有旧引用
