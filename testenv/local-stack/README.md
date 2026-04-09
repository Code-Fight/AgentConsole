# Local Test Stack

这个目录提供一个本地联调环境，用来快速拉起：

- `gateway`
- `client`
- `console`

默认使用 `client` 的 `fake runtime`，适合做服务联调、页面回归、协议回归，不依赖容器内安装真实 `codex`。

## 用法

```bash
./testenv/local-stack/run.sh up
./testenv/local-stack/run.sh ps
./testenv/local-stack/run.sh logs
./testenv/local-stack/run.sh down
```

启动后默认访问地址：

- Gateway: `http://localhost:18080`
- Console: `http://localhost:14173`

## 可选环境变量

- `CAG_GATEWAY_PORT`
- `CAG_CONSOLE_PORT`
- `CAG_MACHINE_ID`
- `CAG_CLIENT_RUNTIME_MODE`
- `CAG_CODEX_BIN`
- `CAG_TESTENV_PROJECT`

说明：

- `CAG_CLIENT_RUNTIME_MODE` 默认是 `fake`
- 如果后续要验证真实 Codex，可以基于这个目录继续替换 `client` 镜像或扩展挂载策略
