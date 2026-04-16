# Dev Integration

这个目录提供一个本地联调环境，用来快速拉起：

- `gateway`
- `client-default`
- `client-hostname`
- `client-not-agent`
- `console`

默认使用真实 `Codex App Server`，3 个 `client` 容器内都会安装 Linux 版 `codex` CLI，并直接使用容器内的默认 Codex 配置目录，不再挂载宿主机 `~/.codex`。

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
- 如需回退 fake runtime，可以显式设置：

```bash
CAG_CLIENT_RUNTIME_MODE=fake ./testenv/dev-integration/run.sh up
```
