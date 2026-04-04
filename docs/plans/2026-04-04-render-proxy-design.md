# render=true 代理统一设计

## 背景

当前 `cmd/mcpserver` 的代理配置已能覆盖 `web_search` 与 `web_fetch` 的普通 HTTP 抓取，但 `web_fetch` 的默认 `render=true` 路径走 Rod 启动的浏览器实例，不会复用 `HTTPClient` 上的代理配置。

这导致同一套代理配置只能覆盖部分网络路径，行为不一致。

## 目标

- 让 `web_fetch` 的默认 `render=true` 也能复用同一套代理配置
- 保持 `cmd/mcpserver` 的现有 CLI 参数不变
- 让 package 模式也能通过同一份配置同时覆盖：
  - `web_search`
  - `web_fetch` 普通抓取
  - `web_fetch` 默认渲染抓取

## 方案

### 方案 A：仅在 `internal/mcpserver` 内拼装浏览器代理参数

优点：

- 改动范围小

缺点：

- package 模式仍无法复用同一套配置
- 代理能力只在 `cmd/mcpserver` 有效

### 方案 B：新增公开代理配置包，工具配置直接接收同一份代理结构

优点：

- `cmd/mcpserver` 和 package 模式都能复用
- 普通 HTTP 抓取与浏览器渲染可共用一套配置
- 文档表达清晰

缺点：

- 需要新增公开 package 和少量 API

## 选型

采用方案 B。

## 设计

新增公开 package：`netproxy`

对外提供：

- `Config`
  - `HTTPProxy`
  - `HTTPSProxy`
  - `NoProxy`
- `Resolve(...)`
  - 把显式配置与环境变量合并
- `NewHTTPClient(...)`
  - 从代理配置生成共享 `http.Client`
- `ChromiumConfig(...)`
  - 把同一份代理配置转换成浏览器可用的 `proxy-server` / `proxy-bypass-list`

工具侧改动：

- `websearch.Config` 新增 `ProxyConfig netproxy.Config`
- `webfetch.Config` 新增 `ProxyConfig netproxy.Config`
- 若未显式传入 `HTTPClient` 且 `ProxyConfig` 已启用：
  - 自动生成 `HTTPClient`
- `webfetch` 默认 `render=true` 路径把 `ProxyConfig` 映射到 Rod launcher 参数

## 优先级

package 模式：

1. `HTTPClient`
2. `ProxyConfig`
3. 默认直连

说明：

- 如果只给了 `HTTPClient`，普通抓取可走代理，但默认浏览器渲染无法可靠推断代理
- 若希望三条链路统一，宿主应显式提供 `ProxyConfig`

`cmd/mcpserver` 模式：

1. CLI 参数
2. 环境变量
3. 直连

构造出的结果同时注入 `HTTPClient` 和 `ProxyConfig`

## 测试

- `netproxy`
  - 环境变量解析
  - HTTP client 代理行为
  - Chromium 代理配置映射
- `webfetch`
  - 默认 render 路径会把代理配置写入 launcher
- `internal/mcpserver`
  - CLI 优先于环境变量
  - toolset 能把统一代理配置注入网络工具

## 文档

更新：

- `README.md`
- `ARCHITECTURE.md`
- `docs/CONFIG.md`
- `docs/TESTING.md`
- `CHANGELOG.md`
