# TESTING

## 目标

确保 `web_search`、`fetch_url`、`run_bash_command` 的行为在抽离后仍保持稳定。

## 测试命令

```bash
go test ./... -count=1
go test ./websearch -count=1
go test ./fetchurl -count=1
go test ./bashcmd -count=1
```

## 分层策略

1. `internal` 层测试通用 helper 与 challenge 检测
2. package 层测试参数校验、核心行为和错误处理
3. 宿主仓库再补 adapter 回归测试

## 当前覆盖

- `websearch`：空 query、缓存命中、工具名
- `fetchurl`：空 URL、404、可读文本提取、render 回退、Cloudflare challenge 回调重试
- `bashcmd`：正常执行、`cwd/base_dir`、超时、受保护域名拦截
