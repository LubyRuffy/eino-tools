# TESTING

## 目标

确保当前命名通用工具和旧命名兼容包在抽离后仍保持稳定。

## 测试命令

```bash
go test ./... -count=1
go test ./websearch -count=1
go test ./webfetch ./exec ./read ./write ./edit ./ls ./tree ./glob ./grep ./pythonrunner ./screenshot -count=1
go test ./fetchurl ./bashcmd -count=1
```

## 分层策略

1. `internal` 层测试通用 helper 与 challenge 检测
2. package 层测试参数校验、核心行为和错误处理
3. 宿主仓库再补 adapter 回归测试

## 当前覆盖

- `websearch`：空 query、缓存命中、工具名
- `webfetch`：空 URL、默认请求头/cookie 注入、注入 HTML fetcher、Cloudflare fallback、challenge handler 回调重试
- `fetchurl`：旧命名兼容包的默认请求头/cookie 注入与 challenge 回调
- `exec`：正常执行、`cwd/base_dir`、超时、受保护域名拦截
- `read/write/edit/ls/tree/glob/grep`：路径解析、基本文件操作和 patch/glob/grep 语义
- `pythonrunner`：空代码、requirements、执行结果结构
- `screenshot`：路径规范化、区域解析、命令选择和 data URL
- `bashcmd`：旧命名兼容包的基础行为
