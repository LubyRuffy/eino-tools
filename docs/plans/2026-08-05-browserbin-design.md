# browserbin 浏览器路径解析设计

**日期：** 2026-08-05  
**状态：** approved

## Goal

为 Rod 启动链路提供可复用的 Chromium 系浏览器可执行文件解析能力，使 `web_fetch` 的 `render=true` 优先使用本机已安装浏览器（Brave 优先），并确保能成功抓取：

`https://www.sentinelone.com/vulnerability-database/cve-2026-50522/`

## 背景

当前 `webfetch.newRodLauncher` 只配置 `Headless` 与代理参数，不设置 `Bin`，依赖 Rod 默认浏览器行为。对部分受 Cloudflare 等保护的站点，本机真实浏览器（尤其 Brave）更可靠。

现有 `screenshot` 仍是 OS 级截屏，本轮不改其实现；共享 package 供其未来或其他 Rod 工具复用。

## 决策摘要

| 项 | 选择 |
|----|------|
| 配置面 | `webfetch.Config.BrowserBin`；不加 MCP CLI / 环境变量 |
| 实现位置 | 公开包 `browserbin`（与 `netproxy` 同级） |
| 探测优先级 | Brave > Edge > Chrome > Chromium |
| 平台 | macOS + Linux + Windows |
| 显式路径无效 | error，不回退探测 |
| 探测全部未命中 | 返回空路径，回退 Rod 默认 |
| 工具参数 | 不新增；仍仅 `url` / `render` |

## 架构

```text
webfetch.Config.BrowserBin
        │
        ▼
browserbin.Resolve(preferred)
        │
        ├─ preferred 非空且存在 → 返回该路径
        ├─ preferred 非空且不存在 → error
        ├─ preferred 空，探测命中 → 返回第一个候选
        └─ preferred 空，全部未命中 → ("", nil)

webfetch.newRodLauncher
        │
        ├─ Resolve 返回非空 → launcher.Bin(path)
        └─ Resolve 返回空   → 不设 Bin（Rod 默认）
```

### 职责边界

| 包 | 做 | 不做 |
|----|----|------|
| `browserbin` | 显式校验、跨平台候选表、按优先级探测 | 启动浏览器、代理、headless |
| `webfetch` | 调用 `Resolve` 并接线 `launcher.Bin` | 维护探测表 |
| `cmd/mcpserver` | 本轮不变 | — |

## API

```go
package browserbin

// Resolve returns an existing Chromium-family browser binary.
// If preferred is non-empty, it must exist or Resolve returns an error.
// If preferred is empty, candidates are tried in Brave > Edge > Chrome > Chromium order.
// If none exist, Resolve returns ("", nil) so callers can fall back to Rod defaults.
func Resolve(preferred string) (string, error)
```

测试可注入 `GOOS` / 文件存在性检查（具体签名实现时确定，保持包对外主入口为 `Resolve`）。

## 候选路径

### macOS

1. `/Applications/Brave Browser.app/Contents/MacOS/Brave Browser`
2. `/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge`
3. `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
4. `/Applications/Chromium.app/Contents/MacOS/Chromium`

### Linux

同优先级内先 `LookPath`，再常见绝对路径：

- Brave: `brave-browser`, `brave`
- Edge: `microsoft-edge`, `microsoft-edge-stable`
- Chrome: `google-chrome`, `google-chrome-stable`
- Chromium: `chromium`, `chromium-browser`

### Windows

在 `Program Files` / `Program Files (x86)` 下：

- Brave: `BraveSoftware\Brave-Browser\Application\brave.exe`
- Edge: `Microsoft\Edge\Application\msedge.exe`
- Chrome: `Google\Chrome\Application\chrome.exe`
- Chromium: `Chromium\Application\chrome.exe`

## 失败语义

| 场景 | 行为 |
|------|------|
| `BrowserBin` 路径不存在 | `Resolve` error → render 失败，错误含路径 |
| `BrowserBin` 路径存在 | 使用该 Bin |
| 未设置且探测命中 | 使用第一个命中路径 |
| 未设置且全部未命中 | `("", nil)` → Rod 默认 |

## 验收标准

```bash
go test ./webfetch -run TestFetch_SentinelOneCVE -count=1 -v
```

必须成功抓取目标 URL，结果非空，且包含 CVE / vulnerability 相关实质正文（非 Cloudflare 拦截页）。

## 非目标

- 不下载浏览器
- 不改 `web_fetch` 工具参数 schema
- 不加 MCP CLI / 环境变量
- 不改现有 OS 级 `screenshot` 实现
- 不把宿主业务类型引入本仓库

## 文档影响

实现时同步更新：

- `README.md`
- `docs/CONFIG.md`
- `docs/TESTING.md`
- `CHANGELOG.md`
- 若形成用户可见能力 / 稳定契约：`docs/FEATURES.md`、`docs/CONTRACTS.md`（若仓库已维护）
