# Dev Integration

这个目录提供一个本地联调环境，用来快速拉起：

- `gateway`
- `client`
- `console`

默认使用真实 `Codex App Server`，`client` 容器内会安装 Linux 版 `codex` CLI，并直接使用容器内的默认 Codex 配置目录，不再挂载宿主机 `~/.codex`。

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
