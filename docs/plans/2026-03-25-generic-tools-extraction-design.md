# Generic Tools Extraction Design

**目标**

把 `zwell_template` 里的通用型内置工具下沉到 `github.com/LubyRuffy/eino-tools`，并让 `zwell_template` 直接依赖 `eino-tools` 提供的 `tool.BaseTool` 实现。工具名、参数名、关键返回结构继续以当前项目为准，不再回退到旧版 `fetch_url` / `run_bash_command` 命名。

**本次范围**

- `web_search`
- `web_fetch`
- `exec`
- `read`
- `edit`
- `write`
- `ls`
- `tree`
- `glob`
- `grep`
- `python_runner`
- `screenshot`

**非目标**

- 本次不迁移 `browser`
- 不迁移 `ask_for_clarification`
- 不迁移 `end_research`
- 不迁移 skills 管理工具
- 不顺手调整上层 `ToolService`、前端开关协议、历史记录结构

## 现状问题

1. 这些通用工具目前散落在 `zwell_template/services/builtin_tools_*.go`，但真正依赖宿主的部分很少，主要是：
   - `base_dir` 默认值
   - 路径白名单
   - `web_search` / `web_fetch` 缓存
   - `web_fetch` / `exec` 的 Cloudflare 人工验证回调
   - `screenshot` 的命令执行注入
2. 公共逻辑已经开始堆积在 `builtin_tools_common.go`、`builtin_tools_path.go`、`builtin_tools_edit.go` 中，但仍被 `BuiltinToolsManager` 包住，无法在其他项目直接复用。
3. `eino-tools` 现在只覆盖旧命名体系下的 3 个工具，不足以成为当前项目“直接依赖”的统一来源。

## 设计原则

1. 对外稳定：
   - 保持当前工具名：`web_search`、`web_fetch`、`exec`、`read`、`edit`、`write`、`ls`、`tree`、`glob`、`grep`、`python_runner`、`screenshot`
   - 保持当前参数名和关键返回结构
2. 对内解耦：
   - `eino-tools` 不引入 `zwell_template/models`、`BuiltinToolsManager`、`ChatHistoryDB`
   - 宿主差异只通过 `Config`、interface、callback 注入
3. 文件命名以当前体系为准：
   - 新 package 和工具说明围绕当前工具名组织，不再使用旧的 `fetchurl` / `bashcmd` 对外语义
4. 共享逻辑集中：
   - 参数解析、错误包装、输出缓冲、路径解析、Patch 解析、截图命令构造等进入 `internal/*`

## 包结构

建议把 `eino-tools` 调整为以下结构：

- `websearch/`
- `webfetch/`
- `exec/`
- `read/`
- `edit/`
- `write/`
- `ls/`
- `tree/`
- `glob/`
- `grep/`
- `pythonrunner/`
- `screenshot/`
- `internal/shared/`
- `internal/cloudflare/`
- `internal/fsutil/`
- `internal/editutil/`
- `internal/screenshotutil/`

说明：

1. package 目录直接对应当前工具语义，避免 adapter 层再做一轮重命名。
2. `internal/fsutil` 统一承接 `base_dir`、白名单、路径安全解析、相对路径展示等逻辑。
3. `internal/editutil` 承接换行归一化、`apply_patch` 文本解析、block replace helper。
4. `internal/screenshotutil` 承接区域解析、输出路径规范化、MIME/data URL helper、跨平台截图命令构造。

## 各工具职责

### 1. `websearch`

- 维持当前工具名 `web_search`
- 参数：`query`
- 配置注入：
  - `HTTPClient`
  - `Cache`
  - `SearchRunner`

### 2. `webfetch`

- 维持当前工具名 `web_fetch`
- 参数：`url`、`render`
- 配置注入：
  - `HTTPClient`
  - `Cache`
  - `RenderFetcher`
  - `ChallengeHandler`
  - `ProtectedDomains`
  - 可选 `BrowserFetch`，用于宿主优先走真实浏览器抓取

### 3. `exec`

- 维持当前工具名 `exec`
- 参数：`command`、`cwd`、`stdin`、`timeout_ms`、`max_output_kb`、`env`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`
  - `ProtectedDomains`
  - `ChallengeHandler`
  - `ShellPath`

### 4. `read`

- 维持当前工具名 `read`
- 参数：`file_path`、`offset`、`limit`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 5. `write`

- 维持当前工具名 `write`
- 参数：`file_path`、`content`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 6. `edit`

- 维持当前工具名 `edit`
- 参数：`file_path`、`search_block`、`replace_block`、`patch`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 7. `ls`

- 维持当前工具名 `ls`
- 参数：`path`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 8. `tree`

- 维持当前工具名 `tree`
- 参数：`path`、`max_depth`、`include`、`exclude`、`only_dirs`、`max_entries`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 9. `glob`

- 维持当前工具名 `glob`
- 参数：`pattern`、`path`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 10. `grep`

- 维持当前工具名 `grep`
- 参数：`pattern`、`path`、`glob`、`output_mode`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`

### 11. `pythonrunner`

- 维持当前工具名 `python_runner`
- 参数：`code`、`requirements`、`timeout_ms`、`max_output_kb`
- 配置注入：
  - `PythonPathResolver`
  - `TempDirFactory`
  - `CommandRunner`

### 12. `screenshot`

- 维持当前工具名 `screenshot`
- 参数：`output_path`、`region`、`include_data_url`、`timeout_ms`、`base_dir`
- 配置注入：
  - `DefaultBaseDir`
  - `AllowedPaths`
  - `CommandBuilder`
  - `CommandRunner`

## 数据流

1. 宿主在 `zwell_template` 的 `CreateXxxTool` 中组装 `Config`
2. 调用 `eino-tools/<pkg>.New(...)`
3. `eino-tools` 在 `InvokableRun` 内解析 JSON 参数
4. 通过 `internal/shared` / `internal/fsutil` / `internal/*util` 执行共性逻辑
5. 需要宿主参与的行为通过 callback/interface 回调回宿主
6. 返回结果仍保持当前工具的字符串/JSON 字符串风格

## `zwell_template` 接入策略

`zwell_template` 保留 `BuiltinToolsManager`，但它只承担装配器角色：

1. 继续提供：
   - `baseDir`
   - 路径白名单
   - `historyDB` cache adapter
   - Cloudflare challenge adapter
   - 浏览器抓取 adapter
   - screenshot command runner adapter
2. 不再持有这些通用工具的核心实现。
3. `services/tool_service.go`、`services/chat_service_tools.go`、`services/builtin_tools_registry.go` 不改工具 ID 和调用路径，只切构造来源。

## 兼容性要求

1. 当前测试中已经锁定的名称必须保留：
   - `web_search`
   - `web_fetch`
   - `exec`
   - `read`
   - `edit`
   - `write`
   - `browser`
2. 通用工具的参数名必须保持不变。
3. `exec` 的 JSON 返回字段必须保持兼容，至少包含：
   - `exit_code`
   - `stdout`
   - `stderr`
   - `elapsed_ms`
   - `failed`
   - `error`（当有错误时）
4. `screenshot` 的 JSON 返回字段必须保持兼容。
5. `edit` 的 patch / search-replace 语义必须保持兼容。

## 测试策略

### `eino-tools`

1. 每个通用工具都要有独立单元测试。
2. 共享 helper 额外覆盖：
   - 路径逃逸阻断
   - 白名单路径放行
   - `apply_patch` 文本解析
   - 截图区域解析
   - Cloudflare 检测
3. 测试优先使用：
   - `httptest`
   - 临时目录
   - stub runner / fake command builder

### `zwell_template`

1. 保留现有工具行为测试。
2. 新增 adapter 测试，验证：
   - `CreateXxxTool` 仍返回当前工具名
   - cache / path / challenge / browser 注入生效
3. 确保 `go test ./services -count=1` 至少通过与通用工具相关的聚焦测试。

## 风险与控制

1. package 改名会带来较大 diff
   - 控制：优先迁共享 helper，再迁工具，再切宿主调用
2. `web_fetch` 和 `exec` 的 Cloudflare 逻辑最容易回归
   - 控制：先把现有测试搬到 `eino-tools`，再切 `zwell_template`
3. `screenshot` 跨平台命令存在环境差异
   - 控制：命令构造与执行解耦，单测只测 builder / parser / payload

## 验收标准

1. `../eino-tools` 能独立通过 `go test ./... -count=1`
2. `zwell_template` 通过 `replace github.com/LubyRuffy/eino-tools => ../eino-tools` 直接依赖新实现
3. 当前 12 个通用工具在 `zwell_template` 中的 ID、参数和主要返回结构不变
4. 两个仓库的 README、ARCHITECTURE、AGENTS、TESTING、CHANGELOG、历史文档同步更新
