# 网络工具代理配置设计

## 背景

当前仓库中的 `websearch` 与 `webfetch` 都支持注入 `HTTPClient`，但 `cmd/mcpserver` 和 `internal/mcpserver` 没有对外暴露代理配置入口，导致用户无法通过启动参数或环境变量统一控制网络工具走代理。

## 目标

- 为 `cmd/mcpserver` 增加网络代理配置能力
- 统一把代理配置注入 `web_search` 与 `web_fetch`
- 保持现有工具 schema 和默认行为兼容
- 让配置行为可测试、可文档化

## 方案

### 方案 A：仅支持显式 CLI 参数

- 新增 `--http-proxy`、`--https-proxy`、`--no-proxy`
- `internal/mcpserver` 基于这些字段构造共享 `http.Transport` / `http.Client`

优点：

- 行为清晰
- 测试稳定

缺点：

- 无法自动继承已有代理环境

### 方案 B：仅支持环境变量

- 内部固定走 `http.ProxyFromEnvironment`
- 不新增 CLI 参数

优点：

- 改动最小

缺点：

- 可发现性差
- 文档与测试表达不充分

### 方案 C：CLI 参数优先，环境变量兜底

- 新增 `--http-proxy`、`--https-proxy`、`--no-proxy`
- 若 CLI 未设置对应值，则回退读取 `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY`
- 统一构造共享 `http.Client` 注入 `websearch` 与 `webfetch`

优点：

- 兼容现有代理环境
- 使用方式明确
- 测试和文档都容易覆盖

缺点：

- 需要定义优先级规则

## 选型

采用方案 C。

## 配置优先级

按字段分别生效：

1. CLI 显式参数优先
2. 对应环境变量兜底
3. 都未设置时不配置代理，保持默认直连

说明：

- `--http-proxy` 优先于 `HTTP_PROXY`
- `--https-proxy` 优先于 `HTTPS_PROXY`
- `--no-proxy` 优先于 `NO_PROXY`
- 若只设置 `http-proxy`，HTTPS 请求没有显式代理时继续按 `https-proxy` / `HTTPS_PROXY` / 无代理规则处理

## 代码改动范围

### `cmd/mcpserver`

- 扩展 flag 解析
- 把代理配置写入 `internal/mcpserver.Config`

### `internal/mcpserver`

- `Config` 增加代理字段
- 新增共享 `http.Client` 构造逻辑
- `NewToolset` 注入 `websearch.Config.HTTPClient` 与 `webfetch.Config.HTTPClient`

### 文档

- 更新 `README.md`
- 更新 `ARCHITECTURE.md`
- 更新 `docs/CLI.md`
- 更新 `docs/CONFIG.md`
- 更新 `docs/TESTING.md`
- 更新 `CHANGELOG.md`

## 测试策略

### 单元测试

- `cmd/mcpserver/main_test.go`
  - 默认值不带代理
  - CLI 参数能正确写入 config
- `internal/mcpserver/tools_test.go`
  - 构造 toolset 时会为网络工具注入共享 client
  - CLI 配置优先于环境变量
  - 仅环境变量存在时也会生效

### 回归验证

- `go test ./internal/mcpserver ./cmd/mcpserver -count=1`
- `go test ./websearch ./webfetch -count=1`
- `go test ./... -count=1`
- `go build ./cmd/mcpserver`

## 兼容性

- 不修改 `web_search` / `web_fetch` 的 tool name、参数结构和返回结构
- 不影响未使用代理的现有宿主
