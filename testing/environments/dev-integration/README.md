# Dev Integration

这个目录提供一个本地联调环境，用来快速拉起：

- `gateway`
- `client-default`
- `client-hostname`
- `client-not-agent`
- `console`

默认使用真实 `Codex App Server`，3 个 `client` 容器内都会安装 Linux 版 `codex` CLI（默认 `0.120.0`）。
为了让真实 App Server 可以在容器里访问 OpenAI，`run.sh` 会在容器启动后把宿主机的 `~/.codex/auth.json` 和 `~/.codex/config.toml` 复制到每个 `client-*` 容器的 `/home/appuser/.codex/`，再由 client 在启动隔离 agent 时复制到各自的隔离目录。
如果宿主机已经导出了 `OPENAI_API_KEY`，3 个 `client` 容器都会自动继承它，managed agents 也会继续透传这个环境变量给 Codex。
如果宿主机没有登录过 Codex，且也没有导出 `OPENAI_API_KEY`，真实模式会得到 `401 Unauthorized`；这种情况下先在宿主机完成登录/导出 key，或临时切到 fake runtime。

## 用法

```bash
./testing/environments/dev-integration/run.sh up
./testing/environments/dev-integration/run.sh ps
./testing/environments/dev-integration/run.sh logs
./testing/environments/dev-integration/run.sh down
```

启动后默认访问地址：

- Gateway: `http://localhost:18080`
- Console: `http://localhost:14173`

默认会看到 3 台机器：

- `client-default`：`MACHINE_NAME=Dev Integration Client`
- `client-hostname`：不传 `MACHINE_NAME`，回退到 docker hostname `hostname-fallback-client`
- `client-not-agent`：`MACHINE_NAME=Not Agent`

## 可选环境变量

- `CAG_GATEWAY_PORT`
- `CAG_CONSOLE_PORT`
- `CAG_MACHINE_NAME_DEFAULT`
- `CAG_HOSTNAME_FALLBACK_NAME`
- `CAG_MACHINE_NAME_NOT_AGENT`
- `CAG_GATEWAY_API_KEY`（默认 `dev-integration-key`，会注入 gateway 并用于联调脚本的鉴权请求）
- `CAG_CLIENT_RUNTIME_MODE`
- `CAG_CODEX_BIN`
- `CAG_CODEX_NPM_VERSION`
- `CAG_TESTENV_PROJECT`

说明：

- 3 个 `client` 都会首次生成并持久化自己的 `machineId`，彼此状态目录独立，不共享机器身份
- `client-hostname` 的机器名来自 hostname 回退，不来自 `MACHINE_NAME`
- `CAG_MACHINE_NAME_DEFAULT` 和 `CAG_MACHINE_NAME_NOT_AGENT` 用于覆盖两个显式命名 client 的展示名
- `CAG_HOSTNAME_FALLBACK_NAME` 用于覆盖 hostname 回退 client 的 docker hostname
- `CAG_CLIENT_RUNTIME_MODE` 会作用到 3 个 client，默认是 `appserver`
- 访问受保护的 Gateway API 时需带 `Authorization: Bearer <CAG_GATEWAY_API_KEY>`，例如：

```bash
curl -H "Authorization: Bearer ${CAG_GATEWAY_API_KEY:-dev-integration-key}" http://localhost:18080/machines
```

- 如需回退 fake runtime，可以显式设置：

```bash
CAG_CLIENT_RUNTIME_MODE=fake ./testing/environments/dev-integration/run.sh up
```
