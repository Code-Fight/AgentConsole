# Settings E2E

这套环境用于验证从 `console` 到 `gateway`，再到 `client` 本地 `~/.codex/config.toml` 的完整配置下发链路。

## 用法

```bash
./testing/environments/settings-e2e/run.sh
```

默认会：

1. 构建并启动 `gateway + client + console`
2. 通过 `Authorization: Bearer <GATEWAY_API_KEY>` 轮询 `/machines`，等待名为 `Settings E2E Client` 的测试机器注册成功
3. 运行 `testing/playwright/settings-e2e.spec.ts`
4. 自动清理容器

临时目录会放在：

- `testing/environments/settings-e2e/.tmp/<project>/client-home`
- `testing/environments/settings-e2e/.tmp/<project>/gateway-data`

## 可选环境变量

- `CAG_GATEWAY_API_KEY`（默认 `settings-e2e-key`，会传给 gateway 容器并作为 Playwright 的 `SETTINGS_E2E_GATEWAY_API_KEY`）
- `CAG_SETTINGS_E2E_GATEWAY_PORT`
- `CAG_SETTINGS_E2E_CONSOLE_PORT`
- `CAG_SETTINGS_E2E_PROJECT`
