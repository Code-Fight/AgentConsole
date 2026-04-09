---
mode: plan
cwd: /Users/zfcode/Documents/DEV/CodingAgentGateway
task: 为 Gateway/Client/Console 增加 Agent 配置下发能力，并补齐单元测试、系统测试和端到端测试门槛
complexity: complex
planning_method: builtin
created_at: 2026-04-10T00:00:00+08:00
---

# Plan: Agent Config Settings

🎯 任务概述
为现有 `console -> gateway -> client -> codex` 链路增加一套以 TOML 为中心的配置下发能力。Gateway 负责保存 `global default` 和可选 `machine override`，Console 负责编辑和下发，Client 负责按 Agent 实现把配置写入本地文件，同时补齐完整的单元测试、系统测试和端到端测试门槛。

📋 执行计划
1. 扩展共享模型和协议，新增 Agent 配置文档、settings 相关 northbound/southbound payload，并先写失败测试锁定接口形状。
2. 实现 `gateway/internal/settings` 存储抽象和本地文件实现，补 settings northbound API，并以 Go 单元测试覆盖 `store + handler`。
3. 在 `client/internal/agent/types` 和 `manager` 增加统一配置接口，补 `agent.config.apply` 命令链路和对应失败/成功测试。
4. 为 `codex` 实现 TOML 校验和 `~/.codex/config.toml` 原子写入，使用临时 HOME 目录做系统级配置写入测试。
5. 为 `console` 增加 `Settings` 页面、路由和交互，使用 TOML 文本域而不是 JSON 编辑器，补前端单测并启用覆盖率阈值。
6. 新增专用 Docker E2E 测试环境，挂载 client HOME，编写从 Settings 页面到本地 `config.toml` 的 Playwright 场景。
7. 加入统一测试脚本与覆盖率脚本，输出单元测试、系统测试、端到端测试三类结果和门槛检查。
8. 运行完整验证，修复剩余缺口，更新测试环境 README 和开发说明。

🧠 当前思考摘要
- 需求的关键不是“读机器当前配置”，而是“从 Gateway 保存的 default/override 下发到机器”，所以 southbound 最小命令只需要 `agent.config.apply`。
- 用 TOML 文本作为产品文档格式最贴近 Codex 真实配置，能避免 JSON/TOML 双向转换带来的错误和复杂度。
- `gateway` 存储必须先抽象，否则一旦引入数据库或 Redis，API 层和业务层会被迫一起重写。
- 测试不能只看“跑过没有”，需要把覆盖率和场景数量写成硬门槛，纳入脚本和 CI 入口。

⚠️ 风险与阻塞
- Go 标准库不支持 TOML，需要引入 TOML 解析库做服务端和 client 端校验。
- `agent.config.apply` 是否自动重启运行中的 Codex 需要保持谨慎，本期应先保证文件更新成功，避免误伤活跃会话。
- Docker E2E 必须能稳定检查 client 容器写出的配置文件，否则端到端验证会停留在 API 成功而不是真实落盘。
- 现有测试环境目录是联调用 fake runtime 的，需要另起一套 settings E2E 环境来验证真实文件写入。

📎 参考
- `/Users/zfcode/Documents/DEV/CodingAgentGateway/docs/superpowers/specs/2026-04-10-agent-config-settings-design.md:1`
- `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/types/interfaces.go:1`
- `/Users/zfcode/Documents/DEV/CodingAgentGateway/client/internal/agent/codex/environment.go:1`
- `/Users/zfcode/Documents/DEV/CodingAgentGateway/gateway/internal/api/server.go:1`
- `/Users/zfcode/Documents/DEV/CodingAgentGateway/console/src/app/router.tsx:1`
