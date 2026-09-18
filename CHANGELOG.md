# CHANGELOG

## Added

- 新增公开 `browserbin` package，按 Brave > Edge > Chrome > Chromium 解析本机 Chromium 系浏览器路径
- `webfetch.Config.BrowserBin`：可选显式浏览器路径；未设置时默认 render 走 `browserbin` 自动探测，探测失败回退 Rod 默认
- 初始化 `eino-tools` 仓库骨架与文档
- 新增 `websearch`，封装 `web_search` Eino `tool.BaseTool`
- 新增 `fetchurl`，封装 `fetch_url` 的 HTTP 抓取、Readability 提取与 render 回退
- 新增 `bashcmd`，封装 `run_bash_command` 的 bash 执行、路径限制与 Cloudflare 保护域名拦截
- 新增 `internal/shared` 与 `internal/cloudflare` 共享能力及对应单元测试
- 新增当前命名版本的通用工具包：`webfetch`、`exec`、`read`、`write`、`edit`、`ls`、`tree`、`glob`、`grep`、`pythonrunner`、`screenshot`
- 新增 `internal/fsutil`、`internal/editutil`、`internal/screenshotutil` 共享 helper 及对应单元测试
- 新增 `internal/mcpserver` 与 `cmd/mcpserver`
- 新增 `docs/CLI.md` 与 `docs/CONFIG.md`
- 新增 `cmd/mcpserver` 网络工具代理配置：`--http-proxy`、`--https-proxy`、`--no-proxy`

## Changed

- `webfetch` 现支持注入 `HTMLFetcher`、`HeaderProvider`、`CookieProvider` 与 `ChallengeDetector`，便于宿主按需覆盖抓取实现、补充请求头、提供 cookie 源并桥接自定义 Cloudflare 错误
- `webfetch` 的 `RenderFetcher` 语义已固定为 `render=true` 的宿主 override 点；未注入时仍保留库内建 render 实现，兼容旧宿主
- `webfetch` / `fetchurl` 的默认抓取链路现内置浏览器风格请求头与 cookie provider 注入，cookie/header 语义不再要求宿主自己重写 `fetchHTML`
- 仓库主推荐入口已切到当前命名工具：`web_search`、`web_fetch`、`exec`、`read`、`edit`、`write`、`ls`、`tree`、`glob`、`grep`、`python_runner`、`screenshot`
- `cmd/mcpserver` 默认支持 `stdio`、`SSE` 与 `streamable HTTP`
- `cmd/mcpserver` 现把 `session_id` 与 `tool` 写入 stderr，便于按会话排障
- `cmd/mcpserver` 现按“CLI 参数优先、环境变量兜底”的规则为 `web_search` 与 `web_fetch` 注入共享代理 `HTTPClient`
- README / ARCHITECTURE / CONFIG 文档现明确区分 CLI 代理配置与 package 注入方式，并补充 `render=true` 代理边界说明
- 新增公开 `netproxy` package，统一承载 `web_search`、`web_fetch` 普通抓取与默认 `render=true` 的代理配置
- `web_fetch` 默认 `render=true` 现可复用与普通 HTTP 抓取一致的代理配置
- `exec` 以及 `read`、`write`、`edit`、`ls`、`tree`、`glob`、`grep`、`screenshot` 的路径参数现仅把 `base_dir` 作为相对路径解析锚点，不再要求最终路径位于 `base_dir` 内
- `exec` 默认改为非交互 bash（`--noprofile --norc -c`），不再跟随 `$SHELL`、不再 `source` 用户 rc；`timeout_ms` 只描述仍在运行的命令进程组
- `grep` 仓库级搜索默认跳过构建/依赖/VCS 目录，尊重 `.gitignore`，并对不可读/超长/二进制文件跳过而不是整次失败；命中数量设上限
- 各工具 `Config` 移除无效的 `AllowedPaths` 字段，`internal/fsutil.ResolvePathWithin` 移除 `allowedPaths` 参数（自路径限制放宽后即无实际作用）
- `webfetch.Fetch` 缓存按模式严格隔离：`render=false` 不再命中此前 `render=true` 写入的缓存条目

## Fixed

- `grep` 裸 `Walk` 撞上 `dist` / `.worktrees` 或超长行 `token too long` 时整次搜索失败
- `exec` 在 macOS 上跟 `$SHELL` 走 zsh `eval`，特殊字符、无匹配 glob、多行 `python -c` 和刚拉起的后台任务不可用；并删除按命令字符串改超时的宿主特例
- `internal/editutil` 遇到 `*** End of File` 时继续解析后续行，不再提前结束整个 patch

- 删除兼容别名包 `fetchurl` 与 `bashcmd`，仓库对外只保留正式工具名
