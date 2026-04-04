# CHANGELOG

## Added

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

## Fixed

- 删除兼容别名包 `fetchurl` 与 `bashcmd`，仓库对外只保留正式工具名
