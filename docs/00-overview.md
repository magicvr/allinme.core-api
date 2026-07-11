---
status: active
owner: 后端团队
last_updated: 2026-07-12
applies_to: allinme.core-api
---

# allinme.core-api 文档总纲

> 本文同时描述当前实现与目标架构。除非明确标记为 `target` 或 `planned`，行为声明均以当前源码和测试为准。

首次阅读顺序：本文 → [实施路线](./06-implementation-roadmap.md) → [领域模型](./05-domain-model.md) → [当前 API](./03-http-api.md) / [目标 API](./03-http-api-target.md)。文档状态与单一事实源规则见 [文档规则](./README.md)。

## 1. 项目职责

`allinme.core-api` 是 Go HTTP 服务，也是 Schema-UI 协议的生产消费者和订单运营 demo API 宿主。当前已实现：

- API 进程启动、路由和健康检查；
- Schema-UI 协议算法的 Go 实现与 conformance 验证；
- 请求构造、响应映射、Action、Reaction、表格状态和上传执行语义；
- 后端测试、vet 与 CI 门禁。

目标态（`target`）提供可重复演示且具有真实状态变化的订单运营能力：

- SQLite 持久化的账号、订单、订单项、退款、附件和会话数据；
- 本地账号登录与 JWT Bearer 认证，内置 `viewer`、`operator`、`approver`、`admin` 角色；
- 搜索分页、联动表单、履约与退款 Action、经营看板和附件上传 API；
- 仓内 YAML 页面源文件，经启动时校验后由 JSON API 下发；
- 本地文件存储附件内容、SQLite 保存元数据，以及可重复的 reset/seed 命令。

本仓库不负责定义 Schema-UI 字段和跨端语义。前后端联调、页面配置生成和协议升级都必须以 [`schema-ui-docs`](../../schema-ui-docs/README.md) 为核心契约。

## 2. 文档地图

| 文档 | 读者 | 用途 |
|---|---|---|
| [README.md](./README.md) | 维护者 / AI | 文档分类、权威边界和写作规则 |
| [01-architecture.md](./01-architecture.md) | 后端开发者 | 进程、HTTP 与协议模块职责 |
| [02-schema-ui-integration.md](./02-schema-ui-integration.md) | 前后端 / AI | 协议对接与版本升级入口 |
| [03-http-api.md](./03-http-api.md) | 前后端开发者 | 当前已实现 HTTP 端点与行为 |
| [03-http-api-target.md](./03-http-api-target.md) | 后端开发者 / AI | 目标 API 原则与 draft endpoint |
| [04-validation.md](./04-validation.md) | 维护者 / CI | 本地和远端验证门禁 |
| [05-domain-model.md](./05-domain-model.md) | 后端开发者 / AI | 订单领域、状态机和权限目标 |
| [06-implementation-roadmap.md](./06-implementation-roadmap.md) | 维护者 / AI | 从当前态到完整 demo 的实施阶段 |
| [scenarios/](./scenarios/README.md) | 产品 / 开发 / 测试 | 本仓业务工作流覆盖 |
| [decisions/](./decisions/README.md) | 维护者 | 后端内部架构决策 |
| [audit/](./audit/README.md) | 维护者 / AI | 审计生命周期与活跃清单 |
| [CHANGELOG.md](./CHANGELOG.md) | 所有人 | 本仓变更记录 |

## 3. 代码地图

| 路径 | 职责 |
|---|---|
| `cmd/api/main.go` | API 进程入口与监听配置 |
| `internal/httpapi/handler.go` | HTTP 路由；当前仅提供 `GET /healthz` |
| `internal/protocol/version_negotiation.go` | 页面协议版本与能力协商 |
| `internal/protocol/request_construction.go` | 结构化请求构造 |
| `internal/protocol/response_mapping.go` | 响应映射 |
| `internal/protocol/query_serialization.go` | query 字节序列化 |
| `internal/protocol/table_query_state.go` | 搜索与分页状态合并 |
| `internal/protocol/reaction_scheduler.go` | Reaction 调度 |
| `internal/protocol/action_outcome.go` | Action 成功/失败结果处理 |
| `internal/protocol/upload_execution.go` | 上传执行契约 |
| `internal/protocol/scenario_execution.go` | 官方场景步骤编排 |

## 4. 当前态与目标态

| 能力 | 当前状态 | 目标状态 |
|---|---|---|
| HTTP | 仅 `GET /healthz` | 页面、认证、订单、退款、附件和看板 API |
| 持久化 | 无 | SQLite + 可重复 migrations/seed |
| 认证授权 | 无 | 本地账号、JWT Bearer、角色与资源级授权 |
| 页面配置 | 无 | YAML 源文件校验后以 JSON 下发 |
| 文件 | 无 | 本地受控目录 + SQLite 元数据 + 鉴权下载 |

目标能力在对应文档中使用 `target` 或 `planned` 标记；只有实现、测试和文档证据齐全后才能改为 `active`。

## 5. 核心边界

- Schema-UI 定义页面、请求和映射语义；本仓提供经过鉴权和业务校验的 API。
- 前端显隐、禁用、确认和 capability 检查不能替代后端鉴权与数据校验。
- `internal/protocol` 用于证明 Go 对共享契约的解释一致，不代表所有算法都应暴露为 HTTP 端点。
- 业务 API 必须明确请求、响应、错误、幂等和权限，不得依赖前端猜测。
- 当前 HTTP 面仅有健康检查；未来端点在代码落地时同步更新 API 文档和场景。

## 6. 开始工作

- 修改当前 HTTP 服务：先读 [01-architecture.md](./01-architecture.md) 与 [03-http-api.md](./03-http-api.md)。
- 实现业务端点：再读 [03-http-api-target.md](./03-http-api-target.md)，实现时收敛对应 draft 并补充证据。
- 修改订单业务：先读 [05-domain-model.md](./05-domain-model.md)，再核对对应场景和权限矩阵。
- 修改协议消费：先读 [02-schema-ui-integration.md](./02-schema-ui-integration.md) 和 Schema-UI 总纲。
- 联调前端：以 Schema-UI 的数据源、Action 与场景文档为契约，再记录本仓业务端点细节。
- 开始实施目标能力：按 [06-implementation-roadmap.md](./06-implementation-roadmap.md) 的依赖顺序推进。
- 提交前：执行 [04-validation.md](./04-validation.md) 中对应门禁。