# Console Gateway Cookie Config 设计文档

## 1. 背景

当前 Console 存在三个结构性问题：

- 前端启动后会直接按当前 origin 或构建时默认值发起 HTTP 和 WebSocket 请求，没有“先配置 Gateway 再连接”的门槛。
- Settings 页面里的 `API Configuration` 虽然展示了 `Console URL` 和 `API Key`，但这些值当前只被保存，不参与真实请求构造，因此页面展示和运行行为不一致。
- Gateway 侧虽然已经暴露了一组可供 Console 使用的 HTTP 和 WebSocket API，但没有 Console 访问鉴权，也没有把 `api_key` 这类启动期配置纳入正式配置加载链路。

这会导致两个直接问题：

- 浏览器在没有有效 Gateway 连接配置时仍然尝试连接服务器。
- 只要能访问 Gateway 地址，就可以直接调用 Console 管理 API，不符合需要显式配置和认证的要求。

## 2. 目标

- Console 只能在浏览器 cookie 中同时拿到 `Gateway URL` 和 `API Key` 时才允许连接 Gateway。
- 如果 cookie 缺少任一项，Console 不得对任何服务器发起业务 HTTP 或 WebSocket 请求。
- Console 启动后要立即提示用户去 Settings 配置连接信息，而不是静默失败或尝试默认地址。
- Gateway 对 Console 暴露的 HTTP API 和 `/ws` 必须要求 `Authorization: Bearer <apiKey>` 或等价的 websocket 鉴权信息。
- Gateway 的启动配置支持从 TOML 配置文件和环境变量加载，`api_key` 属于启动期配置，而不是运行时 settings 数据。

## 3. 非目标

- 不改变 client 到 Gateway 的机器侧连接模型；`/ws/client` 仍沿用现有行为。
- 不设计多租户或多 API key 模型；本次只支持单个全局 API key。
- 不把浏览器中的 Gateway 引导配置写回 Gateway 持久化存储。
- 不引入 Gateway 运行时修改 `api_key` 的管理接口；更改密钥仍通过配置文件或环境变量完成。
- 不为浏览器提供任何默认 Gateway 地址，也不保留“无配置时按当前 origin 试连”的兼容路径。

## 4. 核心决策

### 4.1 浏览器连接引导信息只存 cookie

Console 本地引导配置只存浏览器 cookie，至少包含两个字段：

- `gatewayUrl`
- `apiKey`

建议 cookie 键名固定为：

- `cag_gateway_url`
- `cag_gateway_api_key`

cookie 属性要求：

- `Path=/`
- `SameSite=Lax`
- 在 `https` 下附加 `Secure`

说明：

- `HttpOnly` 不可用，因为前端需要读取这两个值并构造请求。
- 这是一个有意识的产品取舍：连接引导优先本地可控，而不是把引导信息交给远端接口管理。

### 4.2 无 cookie 时 fail-closed

若浏览器 cookie 中缺少 `gatewayUrl` 或 `apiKey`，Console 必须进入严格阻断状态：

- 不允许发起任何业务 HTTP 请求
- 不允许建立任何业务 WebSocket 连接
- 不允许根据当前页面 origin、构建时环境变量或其他默认值推导 Gateway 地址

应用启动后直接弹出提示框，明确告知用户当前未配置 Gateway 连接，并引导用户前往 Settings 页面配置。

### 4.3 Settings 拆成“本地连接引导”和“远端 Gateway 设置”

Settings 页面在职责上拆成两层：

- 本地连接引导：
  - 只读写 cookie
  - 表单项为 `Gateway URL` 和 `API Key`
  - 在无连接配置时仍可单独工作
- 远端 Gateway 设置：
  - 只有在本地连接引导配置完整后才允许加载
  - 继续承载 agent global default、machine override 等由 Gateway 提供的数据

这意味着 Settings 页面即使在“未连接”状态也能打开，但只能显示本地连接配置区域和阻断提示，不得去读取 `/settings/agents`、`/machines`、`/capabilities` 等远端接口。

### 4.4 HTTP 认证统一使用 Bearer Token

所有 Console 发往 Gateway 的 HTTP 请求统一附加：

```text
Authorization: Bearer <apiKey>
```

缺失、格式错误或 key 不匹配时，Gateway 返回 `401 Unauthorized`。

### 4.5 WebSocket 鉴权单独走查询参数

浏览器原生 WebSocket 不能稳定附加自定义 `Authorization` header，因此 `/ws` 的 Console 连接改为通过 URL 查询参数携带 API key：

```text
/ws?threadId=<id>&apiKey=<key>
```

Gateway 在 websocket 握手阶段验证 `apiKey`，不通过则拒绝升级。

说明：

- HTTP 仍然统一使用 `Authorization: Bearer <apiKey>`。
- 机器侧 `/ws/client` 不复用这个 Console key，也不参与这条认证链路。

### 4.6 Gateway 启动配置使用 TOML，环境变量覆盖文件

Gateway 启动期配置增加 TOML 文件读取能力，配置项先保持扁平：

```toml
host = "0.0.0.0"
port = 8080
settings_file = "data/settings.json"
api_key = "replace-me"
```

读取顺序固定为：

1. 内置默认值
2. Gateway TOML 配置文件
3. 环境变量覆盖

推荐新增环境变量：

- `GATEWAY_CONFIG_FILE`
- `GATEWAY_API_KEY`

保留现有环境变量：

- `HOST`
- `PORT`
- `SETTINGS_FILE`

### 4.7 `api_key` 缺失时 Gateway 启动失败

如果最终解析出的 `api_key` 为空，Gateway 应直接启动失败，而不是以“匿名可访问”或“启动后再拒绝请求”的方式继续运行。

这样可以确保：

- 部署阶段立即暴露配置错误
- Gateway 不会在错误配置下以不安全模式对外提供 Console API

## 5. Console 设计

### 5.1 本地 bootstrap 状态层

Console 新增一个本地 bootstrap 层，负责：

- 读取和解析 cookie 中的 `gatewayUrl` 与 `apiKey`
- 统一判断当前是否允许连接 Gateway
- 在保存配置后刷新全局连接状态
- 在认证失败时清理失效状态并重新进入阻断流程

该层是所有 API/WS 调用的前置依赖，其他模块不允许直接拼接 Gateway 地址或直接读 cookie。

### 5.2 HTTP 客户端改造

`console/src/common/api/http.ts` 的行为调整为：

- 删除 `import.meta.env.VITE_API_BASE_URL ?? ""` 的默认直连行为
- 每次请求前从 bootstrap 状态读取 `gatewayUrl` 和 `apiKey`
- 缺少任一值时直接抛出本地“未配置连接”错误，不调用 `fetch`
- 自动为请求添加：
  - `Accept: application/json`
  - `Authorization: Bearer <apiKey>`
- 请求地址改为 `<gatewayUrl><path>`

### 5.3 WebSocket 客户端改造

`console/src/common/api/ws.ts` 的行为调整为：

- 删除基于 `window.location` 的默认 websocket 地址推导
- 改为从 bootstrap 状态读取 `gatewayUrl`，并把其转换为 `ws/wss` 地址
- 连接 URL 附带 `apiKey` 查询参数
- 缺少连接配置时不创建 `WebSocket` 实例
- 鉴权失败或 `401/close` 时通过统一错误路径通知 UI，禁止无意义指数重连

### 5.4 启动阻断与提示框

Console 根层在启动时检查 bootstrap 状态：

- 若缺少 `gatewayUrl` 或 `apiKey`：
  - 立即弹出阻断提示框
  - 文案明确要求前往 Settings 配置 Gateway 连接
  - 全局视图进入“未配置连接”状态
- 用户确认后导航到 `/settings`

该提示框不是可忽略 toast，而是阻断式提示，直到用户进入 Settings 并完成保存。

### 5.5 Settings 页面拆分

Settings 页面调整为两个区块：

1. 本地连接配置区
   - `Gateway URL`
   - `API Key`
   - 保存后写入 cookie
   - 仅做本地校验，不依赖 Gateway

2. 远端 Gateway 设置区
   - Agent 列表
   - Global default TOML
   - Machine override TOML
   - Apply to machine
   - 其他依赖 Gateway 的只读/只写功能

只有 bootstrap 状态完整时才渲染或启用第 2 区块。

### 5.6 `/settings/console` 接口收缩

现有 `/settings/console` 不再承担 `consoleUrl` 和 `apiKey` 的存储职责。

保留该接口时，只允许它保存真正属于 Gateway 远端状态的数据，例如：

- `profile`
- `safetyPolicy`
- `lastThreadId`
- `threadTitles`

如果后续确认 `profile` 和 `safetyPolicy` 也应本地化，可在后续迭代继续收缩该接口；本次先至少把 `consoleUrl` 与 `apiKey` 从远端持久化中移除。

## 6. Gateway 设计

### 6.1 现有 API 能力继续保留

Gateway 当前已经具备通过 API 控制 Console 所需的核心能力：

- 线程列表与详情
- turn 启动、steer、interrupt
- approvals 响应
- environment 读写与同步
- settings 读写与应用
- machines 和 overview 指标

因此本次不重新设计 Console API，而是在现有 API 外层补齐鉴权和配置加载。

### 6.2 鉴权中间层

Gateway 新增 Console 鉴权中间层，用于保护：

- `GET /capabilities`
- `/threads*`
- `/machines*`
- `/overview/metrics`
- `/environment*`
- `/settings*`
- `/approvals*`
- `POST /threads/.../turns`
- `POST /threads/.../turns/.../steer`
- `POST /threads/.../turns/.../interrupt`
- Console websocket `/ws`

可匿名保留：

- `GET /health`
- 机器侧 websocket `/ws/client`

### 6.3 HTTP 认证规则

HTTP 认证规则固定为：

1. 读取 `Authorization` header
2. 要求格式必须是 `Bearer <token>`
3. `<token>` 必须与 Gateway 启动配置中的 `api_key` 完全匹配
4. 不匹配则返回 `401 Unauthorized`

不允许：

- 接受空 token
- 接受没有 `Bearer` 前缀的 header
- 在 HTTP API 上接受查询参数替代 header

### 6.4 WebSocket 认证规则

Console `/ws` 握手时：

1. 从查询参数读取 `apiKey`
2. 若缺失或与配置值不匹配，则拒绝升级
3. 若匹配，则允许 websocket 建立

`threadId` 继续作为可选过滤参数保留。

### 6.5 启动配置加载

`gateway/internal/config` 扩展为：

- 读取 TOML 配置文件
- 合并环境变量
- 校验最终配置

配置结构新增：

- `APIKey string`
- `ConfigFilePath string`

默认值策略：

- `host` 默认 `0.0.0.0`
- `port` 默认 `8080`
- `settings_file` 默认 `data/settings.json`
- `api_key` 无默认值，必须显式提供

### 6.6 运行时 settings 与启动配置解耦

现有 `settings.json` 继续只存运行时可写状态：

- agent global defaults
- machine overrides
- console 远端元数据

它不负责存放：

- `api_key`
- `host`
- `port`
- Gateway 配置文件路径

这样可以避免：

- Gateway 启动期配置被运行时 API 意外覆盖
- 认证边界依赖一个可被 Console 写回的状态文件

## 7. 数据流

### 7.1 首次打开且无 cookie

1. Console 根层读取 cookie
2. 发现 `gatewayUrl` 或 `apiKey` 缺失
3. 应用进入阻断态，不触发任何 HTTP/WS
4. 弹出提示框，引导用户进入 Settings
5. Settings 仅展示本地连接表单

### 7.2 用户保存本地连接配置

1. 用户输入 `Gateway URL` 和 `API Key`
2. 前端做本地校验
3. 写入 cookie
4. bootstrap 状态刷新
5. 应用解锁 Gateway 请求能力
6. Settings 开始加载 capabilities、agents、machines 等远端数据

### 7.3 正常远端请求

1. 调用 `http(path)`
2. bootstrap 提供 `gatewayUrl` 和 `apiKey`
3. 请求发送到 `<gatewayUrl><path>`
4. 自动携带 `Authorization: Bearer <apiKey>`
5. Gateway 验证通过后返回结果

### 7.4 认证失败

1. Gateway 返回 `401`
2. Console 统一识别为连接配置失效
3. 停止后续请求和 websocket 重连
4. 弹出提示框并引导用户重新进入 Settings 修正连接配置

## 8. 异常与边界处理

- cookie 中 `gatewayUrl` 非法：
  - 视为未配置
  - 不发请求
  - 要求用户重新保存
- cookie 中 `apiKey` 为空字符串：
  - 视为未配置
  - 不发请求
- Gateway URL 为 `http` 时：
  - HTTP 请求直接使用 `http`
  - WebSocket 使用 `ws`
- Gateway URL 为 `https` 时：
  - HTTP 请求直接使用 `https`
  - WebSocket 使用 `wss`
- HTTP 收到 `401`：
  - 前端进入“连接失效”状态
  - 不允许继续静默重试
- websocket 因鉴权失败关闭：
  - 前端进入与 HTTP `401` 相同的错误路径
  - 不启动指数退避重连
- Gateway 启动配置文件不存在但提供了环境变量：
  - 允许启动
  - 环境变量直接生效
- Gateway 启动配置文件存在但 TOML 非法：
  - 启动失败
- 最终 `api_key` 为空：
  - 启动失败

## 9. 测试策略

### 9.1 Console 单元测试

- 无 cookie 时 `http()` 不调用 `fetch`
- 无 cookie 时 `connectConsoleSocket()` 不创建 `WebSocket`
- Settings 在无 cookie 状态下可以渲染本地连接表单
- 保存 cookie 后，Console 才开始调用 Gateway API
- HTTP 请求自动附加 `Authorization: Bearer <apiKey>`
- websocket URL 正确附加 `apiKey`
- `401` 会触发重新配置引导
- `/settings/console` 不再写入 `consoleUrl` 和 `apiKey`

### 9.2 Gateway 单元测试

- TOML 配置文件可被正确读取
- 环境变量能覆盖 TOML 配置
- 缺少 `api_key` 时配置读取失败
- 正确 Bearer token 能访问受保护 API
- 缺失 header、错误格式、错误 token 返回 `401`
- `/ws` 在错误 `apiKey` 下拒绝升级
- `/health` 在无鉴权时仍可访问
- `/ws/client` 不受 Console 鉴权影响

### 9.3 集成验证

- 浏览器首次打开且无 cookie，不会触发任何业务请求
- 用户在 Settings 保存 `Gateway URL` 和 `API Key` 后可以正常连接 Gateway
- 修改为错误 key 后，HTTP 和 websocket 都会重新进入阻断态
- 远端 settings、threads、machines 在连接成功后恢复工作

## 10. 实施范围

预计会涉及以下区域：

- Console 本地 bootstrap 与 cookie 读写
- Console HTTP/WebSocket 客户端
- Console Settings 页面和启动阻断弹窗
- Console 测试
- Gateway 配置读取
- Gateway HTTP 鉴权与 websocket 鉴权
- Gateway 测试
- README 或运行文档中关于 `api_key` 与 TOML 配置的说明

## 11. 最终结论

本次方案把 Console 连接 Gateway 的前提收敛为两个本地 cookie 值：`gatewayUrl` 和 `apiKey`。没有它们时，前端不再尝试连接任何服务器，而是直接阻断并引导用户去 Settings 配置。

Gateway 继续复用现有 Console API 面，但新增强制认证和正式的启动配置加载链路。启动期安全配置进入 TOML 和环境变量体系，运行时可写 settings 继续留在 `settings.json`。

这样可以同时解决“未配置仍自动连”、“Settings 展示和真实运行脱节”以及“Gateway API 缺少显式认证边界”这三个问题。
