# Settings E2E

这套环境用于验证从 `console` 到 `gateway`，再到 `client` 本地 `~/.codex/config.toml` 的完整配置下发链路。

## 用法

```bash
./testing/environments/settings-e2e/run.sh
```

默认会：

1. 构建并启动 `gateway + client + console`
2. 等待名为 `Settings E2E Client` 的测试机器注册成功
3. 运行 `testing/playwright/settings-e2e.spec.ts`
4. 自动清理容器

临时目录会放在：

- `testing/environments/settings-e2e/.tmp/<project>/client-home`
- `testing/environments/settings-e2e/.tmp/<project>/gateway-data`
