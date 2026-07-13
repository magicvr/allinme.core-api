---
status: active
owner: 后端团队
last_updated: 2026-07-13
applies_to: allinme.core-api
---

# 后端架构

## 1. 当前形态

服务使用 Go 标准库 HTTP 栈和 `database/sql`：`cmd/api` 是薄入口，`internal/app` 组装生命周期，`internal/httpapi` 提供运行状态、认证、订单、退款、看板 API 与通用中间件，`internal/auth` 负责 bcrypt、JWT、session 用例和角色策略，`internal/order` 持有订单/退款状态机、幂等和看板 service，`internal/store` 使用 `modernc.org/sqlite v1.53.0` 管理 SQLite v6。`internal/protocol/` 提供 Schema-UI 共享算法的 Go 实现和 conformance 证明，尚未暴露页面业务 API。

| 层次 | 路径 | 当前职责 |
|---|---|---|
| 进程入口 | `cmd/api/`、`cmd/admin/` | 启动 API，执行 migrate/seed/reset/bootstrap-admin |
| 应用装配 | `internal/app/`、`internal/config/` | 配置、依赖组装、关闭顺序和运行模式 |
| HTTP 适配 | `internal/httpapi/` | health/ready、认证、订单、退款、看板路由、CORS、限流、request ID、访问日志、recovery 和错误 envelope |
| 认证用例 | `internal/auth/` | 密码、JWT、session 认证与角色 allowlist |
| 订单/退款用例 | `internal/order/` | 订单、履约、退款、capability、幂等 snapshot 和看板统计 service |
| 数据存储 | `internal/store/` | SQLite v6、migration、users/session、订单/退款 seed、事务、repository 和 readiness 状态 |
| 协议执行 | `internal/protocol/` | 与 Schema-UI conformance 对齐的纯算法 |

业务能力增长时应保持 transport、业务用例和协议算法边界，避免把鉴权或业务规则写进 fixture 适配代码。

## 2. 目标形态

目标态采用单进程、模块化分层架构，保留 Go 标准库 HTTP 边界，并为 SQLite、文件和页面配置建立显式适配器：

| 层次 | 目标职责 |
|---|---|
| `cmd/` | API 与开发期 migrate/seed/reset 命令入口 |
| `internal/httpapi/` | 路由、中间件、JSON/multipart 输入输出和稳定错误映射 |
| `internal/auth/` | 密码校验、JWT 签发/验证、会话撤销与角色授权 |
| `internal/order/` | 订单、履约、退款、附件绑定和看板用例；状态机、金额、capability 与 service 位于此层 |
| `internal/store/` | SQLite migrations、事务和 repository 实现 |
| `internal/pages/` | 嵌入 YAML、启动时协议校验和 JSON 页面响应 |
| `internal/files/` | 临时上传、本地文件持久化、清理和鉴权下载 |
| `internal/protocol/` | Schema-UI 共享算法与 conformance，不依赖业务模块 |

HTTP handler 只负责 transport；用例层拥有业务事务语义和状态转换，store 拥有 SQL 事务对象与提交/回滚实现，并可提供聚合级原子 repository 方法或事务执行器；SQLite 与文件系统通过窄接口注入。页面配置引用业务 API，但业务代码不得读取页面 YAML 决定权限或规则。

## 3. HTTP 层规则

- `GET /healthz` 是 liveness，只证明进程可响应；`GET /readyz` 当前检查 SQLite 可访问且 schema 为最新，阶段六再扩展页面模块检查。
- handler 负责 method、Content-Type、输入限制、状态码和稳定错误结构。
- 请求 context 必须贯穿业务调用；阻塞 I/O 应响应取消和超时。
- 读取 body 时设置合理大小限制，JSON 输入应明确未知字段和多值处理策略。
- 写响应前完成可能失败的编码准备，避免部分响应后改变状态码。
- 认证、授权、幂等和资源状态由后端执行，不信任前端显隐、确认或页面配置。
- 新端点设计先更新 [目标 HTTP API](./03-http-api-target.md)；实现和测试完成后移入 [当前 HTTP API](./03-http-api.md)，并同步相关场景。

## 4. 认证与数据边界

- 本地账号使用适合密码存储的自适应哈希；登录响应签发短期 JWT Bearer。
- JWT 至少包含 subject、role、expiry、issued-at 和唯一 token ID；服务端同时检查 SQLite session 是否未撤销，以支持登出和禁用账号立即失效。
- JWT 签名密钥来自环境或受控 secret，不进入源码、页面配置、seed 或日志；生产模式缺失密钥时拒绝启动。
- SQLite 启用 foreign keys、busy timeout 和 WAL；每个 SQLite 原子阶段内涉及订单、退款和附件元数据的修改必须在单一事务中完成。SQLite 与文件系统不能组成共同事务：需要文件隔离的附件 edit/remove 或内部订单清理使用“准备事务 → 文件隔离 → 最终 SQLite 事务 → purge”，准备态可被并发请求观察并按领域状态稳定分类，最终事务重新验证 version、退款历史和 operation token；进程退出后的 restore/finalize/purge 由持久 journal、独占启动恢复和受限 cleanup 负责。
- 写操作使用资源 `version` 做乐观并发控制；幂等操作在数据库中保存幂等键与结果。
- 上传文件位于配置的数据目录，临时文件与最终文件不得由 HTTP 静态目录直接暴露。

## 5. 页面配置（target）

- 页面源文件固定放在 `internal/pages/yaml/*.yaml`；YAML 是可审查源，运行时 API 只返回 JSON。
- `internal/pages` 在包内使用 `//go:embed yaml/*.yaml` 将页面纳入发布二进制，不跨包目录读取父级路径，也不在运行时读取工作目录中的任意文件。
- Go 测试使用 CI 固定的 Schema-UI checkout 校验全部页面；API 启动时再次解析并执行固定版本与 capability 防御性校验。
- 任一页面解析或校验失败时 API 拒绝启动；启动成功后 `/readyz` 才能报告页面模块就绪。
- 页面 ID 使用显式 allowlist 映射到嵌入资源，不把请求路径拼接为文件路径。
- 页面响应可按角色过滤整页可用性，但不得动态改写协议语义来替代 API 授权。

## 6. 协议模块约束

- Go 的 nil、空值、缺失字段和 JSON `null` 必须保持协议可观察语义。
- 需要区分缺失/空值时使用 `json.RawMessage`、指针或显式 presence 类型，不依赖零值猜测。
- query、path 和 body 使用结构化构造，并保持与参考实现相同的字节序列化。
- `scenario_execution.go` 只编排已定义步骤，不维护另一套业务语义。
- 每个协议模块的行为变化必须同步相邻测试，并由共享 fixture 提供正例和反例。

## 7. 依赖与安全

- 优先使用标准库；新增依赖必须有明确收益、固定版本和维护状态评估。
- `go.mod` / `go.sum` 必须由 Go 工具维护，不手工删改校验记录。
- 错误对外提供稳定、安全的信息，对内保留可诊断上下文；不得泄露 token、内部路径或敏感数据。
- goroutine、timer、response body 和其他资源必须有清晰所有权与关闭路径。

## 8. 依赖方向

`cmd` 组装所有依赖；`httpapi` 依赖业务用例接口；业务模块依赖 repository、session、clock 和文件接口；适配器实现这些接口。`protocol` 不依赖 HTTP、业务或存储模块。跨前后端契约变化遵循 [Schema-UI 接入契约](./02-schema-ui-integration.md)，本仓内部架构变化记录在 [`decisions/`](./decisions/README.md)。
