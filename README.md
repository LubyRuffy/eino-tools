# Eino Tools

`eino-tools` 是一个面向 Go / CloudWeGo Eino 的可复用工具仓库，提供可直接返回 `tool.BaseTool` 的中间模块。

## 当前包含

- `websearch` -> `web_search`
- `webfetch` -> `web_fetch`
- `exec` -> `exec`
- `read` / `write` / `edit`
- `ls` / `tree` / `glob` / `grep`
- `pythonrunner` -> `python_runner`
- `screenshot`

兼容保留：

- `fetchurl`
- `bashcmd`

## 快速开始

```bash
go get github.com/LubyRuffy/eino-tools
```

```go
searchTool, err := websearch.New(context.Background(), websearch.Config{})
if err != nil {
	panic(err)
}
_ = searchTool
```

### `web_fetch`

```go
fetchTool, err := webfetch.New(webfetch.Config{})
if err != nil {
	panic(err)
}
_ = fetchTool
```

宿主若希望接管 `render=true` 的执行后端，可注入 `RenderFetcher`；未注入时工具会继续使用库内建的默认 render 实现，保证兼容。

### `exec`

```go
execTool, err := exec.New(exec.Config{
	DefaultBaseDir: ".",
})
if err != nil {
	panic(err)
}
_ = execTool
```

## 常见使用方式

1. 直接把工具注册到 Eino agent
2. 在业务仓库中包一层 adapter，注入缓存、cookie 源、路径策略与挑战回调
3. 直接复用当前命名的结构化文件工具和运行工具
4. 用宿主仓库的 `replace` 指向本地工作目录做联调，再切正式 tag
