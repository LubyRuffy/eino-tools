# CHANGELOG

## Added

- 初始化 `eino-tools` 仓库骨架与文档
- 新增 `websearch`，封装 `web_search` Eino `tool.BaseTool`
- 新增 `fetchurl`，封装 `fetch_url` 的 HTTP 抓取、Readability 提取与 render 回退
- 新增 `bashcmd`，封装 `run_bash_command` 的 bash 执行、路径限制与 Cloudflare 保护域名拦截
- 新增 `internal/shared` 与 `internal/cloudflare` 共享能力及对应单元测试
- 新增当前命名版本的通用工具包：`webfetch`、`exec`、`read`、`write`、`edit`、`ls`、`tree`、`glob`、`grep`、`pythonrunner`、`screenshot`
- 新增 `internal/fsutil`、`internal/editutil`、`internal/screenshotutil` 共享 helper 及对应单元测试

## Changed

- `webfetch` 现支持注入 `HTMLFetcher`、`HeaderProvider`、`CookieProvider` 与 `ChallengeDetector`，便于宿主按需覆盖抓取实现、补充请求头、提供 cookie 源并桥接自定义 Cloudflare 错误
- `webfetch` 的 `RenderFetcher` 语义已固定为 `render=true` 的宿主 override 点；未注入时仍保留库内建 render 实现，兼容旧宿主
- `webfetch` / `fetchurl` 的默认抓取链路现内置浏览器风格请求头与 cookie provider 注入，cookie/header 语义不再要求宿主自己重写 `fetchHTML`
- 仓库主推荐入口已切到当前命名工具：`web_search`、`web_fetch`、`exec`、`read`、`edit`、`write`、`ls`、`tree`、`glob`、`grep`、`python_runner`、`screenshot`

## Fixed

- 暂无
