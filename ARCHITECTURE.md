# ARCHITECTURE

## 系统整体架构

仓库按“导出工具 package + 内部共享能力”组织：

- `websearch`：搜索工具
- `webfetch`：当前命名的网页抓取与可读性提取工具
- `exec`：当前命名的命令执行工具
- `read` / `write` / `edit`：结构化文件读写工具
- `ls` / `tree` / `glob` / `grep`：文件系统检索工具
- `pythonrunner`：Python 隔离执行工具
- `screenshot`：截图工具
- `fetchurl` / `bashcmd`：旧命名兼容包
- `internal/shared`：参数解析、输出缓冲、工具错误处理
- `internal/cloudflare`：Cloudflare 检测与保护域名状态
- `internal/fsutil`：`base_dir`、白名单路径与显示路径处理
- `internal/editutil`：apply-patch 文本解析与替换
- `internal/screenshotutil`：截图路径、区域与 mime 处理

## 核心模块

### websearch

封装 DuckDuckGo 文本搜索，并支持缓存与 HTTP client 注入。

### webfetch / fetchurl

封装 HTTP 抓取、Readability 文本提取、可选渲染抓取与 Cloudflare 挑战回调。

宿主可以只注入 `HTTPClient`，也可以额外注入：

- `Cache`
- `HeaderProvider`
- `CookieProvider`
- `HTMLFetcher`
- `RenderFetcher`
  - `render=true` 的宿主覆盖点；若未注入则保持工具内建的 Rod 渲染实现，保证兼容
- `ChallengeDetector`
- `ChallengeHandler`

### exec / bashcmd

封装 `/bin/bash -c` 执行、路径限制、输出截断、超时与 Cloudflare 保护域名拦截。

宿主可通过 `Config` 注入：

- `DefaultBaseDir`
- `AllowedPaths`
- `ProtectedDomains`
- `ChallengeHandler`

### 文件与截图工具

`read/write/edit/ls/tree/glob/grep/screenshot` 共享 `internal/fsutil`、`internal/editutil` 与 `internal/screenshotutil`，把路径边界、patch 解析和平台差异统一收敛到内部 helper。

## 请求与数据流

1. 调用方构造 `Config`
2. 调用 `New(...)` 返回 Eino `tool.BaseTool`
3. 工具执行时通过共享 helper 解析参数与格式化结果
4. 需要宿主参与的行为通过 callback/interface 回调给宿主
