# ARCHITECTURE

## 系统整体架构

仓库按“导出工具 package + 内部共享能力”组织：

- `websearch`：搜索工具
- `fetchurl`：网页抓取与可读性提取工具
- `bashcmd`：bash 执行工具
- `internal/shared`：参数解析、输出缓冲、工具错误处理
- `internal/cloudflare`：Cloudflare 检测与保护域名状态

## 核心模块

### websearch

封装 DuckDuckGo 文本搜索，并支持缓存与 HTTP client 注入。

### fetchurl

封装 HTTP 抓取、Readability 文本提取、可选渲染抓取与 Cloudflare 挑战回调。

宿主可以只注入 `HTTPClient`，也可以额外注入：

- `Cache`
- `RenderFetcher`
- `ChallengeHandler`

### bashcmd

封装 `/bin/bash -c` 执行、路径限制、输出截断、超时与 Cloudflare 保护域名拦截。

宿主可通过 `Config` 注入：

- `DefaultBaseDir`
- `AllowedPaths`
- `ProtectedDomains`
- `ChallengeHandler`

## 请求与数据流

1. 调用方构造 `Config`
2. 调用 `New(...)` 返回 Eino `tool.BaseTool`
3. 工具执行时通过共享 helper 解析参数与格式化结果
4. 需要宿主参与的行为通过 callback/interface 回调给宿主
