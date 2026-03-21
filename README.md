# Eino Tools

`eino-tools` 是一个面向 Go / CloudWeGo Eino 的可复用工具仓库，提供可直接返回 `tool.BaseTool` 的中间模块。

## 当前包含

- `web_search`
- `fetch_url`
- `run_bash_command`

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

### `fetch_url`

```go
fetchTool, err := fetchurl.New(fetchurl.Config{})
if err != nil {
	panic(err)
}
_ = fetchTool
```

### `run_bash_command`

```go
bashTool, err := bashcmd.New(bashcmd.Config{
	DefaultBaseDir: ".",
})
if err != nil {
	panic(err)
}
_ = bashTool
```

## 常见使用方式

1. 直接把工具注册到 Eino agent
2. 在业务仓库中包一层 adapter，注入缓存、路径策略与挑战回调
3. 独立复用 `fetch_url` 或 `run_bash_command` 的能力模块
4. 用宿主仓库的 `replace` 指向本地工作目录做联调，再切正式 tag
