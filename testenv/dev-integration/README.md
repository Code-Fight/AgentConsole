# Dev Integration

这个目录提供一个本地联调环境，用来快速拉起：

- `gateway`
- `client`
- `console`

默认使用真实 `Codex App Server`，`client` 容器内会安装 Linux 版 `codex` CLI（默认 `0.120.0`）。
为了让真实 App Server 可以在容器里访问 OpenAI，`run.sh` 会在容器启动后把宿主机的 `~/.codex/auth.json` 和 `~/.codex/config.toml` 复制到 `client` 容器的 `/home/appuser/.codex/`，再由 client 在启动隔离 agent 时复制到各自的隔离目录。
如果宿主机已经导出了 `OPENAI_API_KEY`，`client` 容器会自动继承它，managed agents 也会继续透传这个环境变量给 Codex。
如果宿主机没有登录过 Codex，且也没有导出 `OPENAI_API_KEY`，真实模式会得到 `401 Unauthorized`；这种情况下先在宿主机完成登录/导出 key，或临时切到 fake runtime。

## 用法

```bash
./testenv/dev-integration/run.sh up
./testenv/dev-integration/run.sh ps
./testenv/dev-integration/run.sh logs
./testenv/dev-integration/run.sh down
```

启动后默认访问地址：

- Gateway: `http://localhost:18080`
- Console: `http://localhost:14173`

## 可选环境变量

- `CAG_GATEWAY_PORT`
- `CAG_CONSOLE_PORT`
- `CAG_MACHINE_NAME`
- `CAG_CLIENT_RUNTIME_MODE`
- `CAG_CODEX_BIN`
- `CAG_CODEX_NPM_VERSION`
- `CAG_TESTENV_PROJECT`

说明：

- `client` 会首次生成并持久化自己的 `machineId`，不再依赖环境变量传入固定机器 ID
- `CAG_MACHINE_NAME` 用于设置 Console 里展示的友好机器名；不设置时，client 默认取 hostname
- `CAG_CLIENT_RUNTIME_MODE` 默认是 `appserver`
- 如需回退 fake runtime，可以显式设置：

```bash
CAG_CLIENT_RUNTIME_MODE=fake ./testenv/dev-integration/run.sh up
```
