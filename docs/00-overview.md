---
status: active
owner: 后端团队
last_updated: 2026-07-16
applies_to: allinme.core-api
---

# allinme.core-api 文档总纲

> 本文同时描述当前实现与目标架构。除非明确标记为 `target` 或 `planned`，行为声明均以当前源码和测试为准。

首次阅读顺序：本文 → [实施路线](./06-implementation-roadmap.md) → [领域模型](./05-domain-model.md) → [当前 API](./03-http-api.md) / [目标 API](./03-http-api-target.md)。文档状态与单一事实源规则见 [文档规则](./README.md)。

## 1. 项目职责

`allinme.core-api` 是 Go HTTP 服务，也是 Schema-UI 协议的生产消费者和订单运营 demo API 宿主。当前已实现：

- API 进程启动、健康/readiness 检查和有序 shutdown；
- SQLite migration/seed/reset、账号、会话、订单、订单项、退款和两类幂等快照持久化；
- 本地账号登录、JWT Bearer session 撤销和 `viewer`、`operator`、`approver`、`admin` 四角色授权；
- 订单列表/详情、幂等创建、草稿编辑、履约 Action、退款申请/审批/拒绝、经营看板和可选可信 origin CORS；
- Schema-UI 协议算法的 Go 实现与 conformance 验证；
- 请求构造、响应映射、Action、Reaction、表格状态和上传执行语义；
- 后端测试、vet 与 CI 门禁。

目标态（`target`）提供可重复演示且具有真实状态变化的订单运营能力：

- SQLite 持久化的附件元数据和本地文件内容；
- 本地账号登录与 JWT Bearer 认证，内置 `viewer`、`operator`、`approver`、`admin` 角色；
- 附件上传、绑定和鉴权下载 API；
- 仓内 YAML 页面源文件，经启动时校验后由 JSON API 下发；
- 本地文件存储附件内容、SQLite 保存元数据，以及可重复的 reset/seed 命令。

本仓库不负责定义 Schema-UI 字段和跨端语义。前后端联调、页面配置生成和协议升级都必须以 [`schema-ui-docs`](../../schema-ui-docs/README.md) 为核心契约。

## 2. 项目宪章与防漂移规则

目标优先级固定为：**Demo 闭环 > Admin 可联调场景 > 模板抽象 > 流程优化**。

| 层级 | 目标 | 可观察完成结果 |
|---|---|---|
| 主目标 | 可运行、可重置且具有真实状态变化的订单运营 Demo API | 从全新数据目录可完成登录、查询、写入、履约、审批、看板、附件和页面加载场景 |
| 次目标 | 为通用 Admin 前台提供真实后台交互 | 场景覆盖认证、列表、表单、权限、状态机、审批、看板、附件和服务端页面配置 |
| 派生目标 | 从已验证实现提炼后续 API 项目可复用的结构 | 复用边界由第二个真实消费者验证，不以预建框架或复制治理历史代替 |

本仓默认不以以下方向为目标：

- 通用 BaaS、无代码 CRUD 平台或多租户管理平台；
- 生产级分布式订单系统及其完整发布、调度和灾难恢复体系；
- AI 审计或开发治理框架。治理工具可以辅助交付，但默认不作为模板产品的一部分。

资产按迁移价值解释：

| 资产 | 默认策略 |
|---|---|
| `cmd` / `app` / `config` / HTTP 基础设施 / migration 等 runtime 骨架 | 可作为后续 API 项目的结构参考，经真实复用后再抽取 |
| `auth` | 可选能力；身份模型不同的项目应替换，不把当前四角色视为框架契约 |
| `order` / refund / dashboard | 示例领域；复用状态机、幂等和乐观锁模式，不默认复制业务模型 |
| `protocol` | Schema-UI 生态消费者；仅同协议项目启用 |
| audits / remediations / skills / prompts 等 governance 资产 | 可选维护工具；默认不复制到新 API 项目，也不作为产品完成度 |

完成度使用三类指标，不使用 AUD、REM、validator 或文档行数代替：

1. **端到端 Admin 场景数**：以[场景目录](./scenarios/README.md)和[路线图完整验收](./06-implementation-roadmap.md#8-完整验收)为证据；
2. **current API 覆盖**：以[当前 API](./03-http-api.md)和[验证矩阵](./04-validation.md#4-目标-demo-验证矩阵)中 `enabled: yes` 的真实入口为证据；
3. **可复用边界**：记录第二个真实项目实际复用或替换了哪些 runtime、auth、domain 和 protocol 资产。

正常迭代的投入参考为约 **70% 产品/场景、20% 测试与契约、10% 文档与治理**。连续迭代中治理投入超过 20% 时，必须说明它解除的具体产品阻塞；不能只以治理完整性作为理由。

每个里程碑结束时检查：

1. 最近提交是否仍以产品能力和用户场景为主；
2. 路线图当前阶段是否出现了对应产品代码进展；
3. 是否出现尚无第二个真实消费者的抽象或框架；
4. 流程和校验是否在服务自身，而不是解除产品交付阻塞；
5. 本周期是否新增了可运行、可演示的 Admin 场景。

抽象规则：先完成业务实现；只有第二个真实项目出现相同需求时才稳定抽取，禁止为想象中的后续项目预建框架。

## 3. 文档地图

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
| [plans/](./plans/README.md) | 维护者 / AI | 实施计划、checklist 与归档索引 |
| [audits/](./audits/README.md) | 维护者 / AI | 审计记录、发现历史与追溯关系 |
| [remediations/](./remediations/README.md) | 维护者 / AI | 审计整改记录及复审队列 |
| [CHANGELOG.md](./CHANGELOG.md) | 所有人 | 本仓变更记录 |

## 4. 代码地图

| 路径 | 职责 |
|---|---|
| `cmd/api/main.go` | API 进程入口与监听配置 |
| `internal/app/api.go` | API 装配、共享数据库/进程锁所有权与有序 shutdown |
| `internal/httpapi/` | health/readiness、认证、订单、退款、看板和 CORS 路由及稳定错误映射 |
| `internal/auth/` | 本地账号、JWT、session 与角色授权 |
| `internal/order/` | 订单/退款领域模型、查询/写入/看板 service、状态机与幂等快照 |
| `internal/store/` | SQLite、migration/seed、认证、订单、退款与看板 repository |
| `internal/protocol/version_negotiation.go` | 页面协议版本与能力协商 |
| `internal/protocol/request_construction.go` | 结构化请求构造 |
| `internal/protocol/response_mapping.go` | 响应映射 |
| `internal/protocol/query_serialization.go` | query 字节序列化 |
| `internal/protocol/table_query_state.go` | 搜索与分页状态合并 |
| `internal/protocol/reaction_scheduler.go` | Reaction 调度 |
| `internal/protocol/action_outcome.go` | Action 成功/失败结果处理 |
| `internal/protocol/upload_execution.go` | 上传执行契约 |
| `internal/protocol/scenario_execution.go` | 官方场景步骤编排 |

## 5. 当前态与目标态

| 能力 | 当前状态 | 目标状态 |
|---|---|---|
| HTTP | health/readiness、认证、订单查询/写入/履约 Action、退款、看板、CORS | 页面和附件 API |
| 持久化 | SQLite + 可重复 migrations/seed/reset；账号、session、订单、退款和幂等快照 | 附件元数据与页面配置持久化 |
| 认证授权 | 本地账号、JWT Bearer、可撤销 session 与四角色订单/退款/看板授权 | 附件和页面资源级授权 |
| 页面配置 | 无 | YAML 源文件校验后以 JSON 下发 |
| 文件 | 无 | 本地受控目录 + SQLite 元数据 + 鉴权下载 |

目标能力在对应文档中使用 `target` 或 `planned` 标记；只有实现、测试和文档证据齐全后才能改为 `active`。

## 6. 核心边界

- Schema-UI 定义页面、请求和映射语义；本仓提供经过鉴权和业务校验的 API。
- 前端显隐、禁用、确认和 capability 检查不能替代后端鉴权与数据校验。
- `internal/protocol` 用于证明 Go 对共享契约的解释一致，不代表所有算法都应暴露为 HTTP 端点。
- 业务 API 必须明确请求、响应、错误、幂等和权限，不得依赖前端猜测。
- 当前 HTTP 面以 [当前 API](./03-http-api.md) 为准；附件和页面端点仍是后续阶段目标。

## 7. 开始工作

- 修改当前 HTTP 服务：先读 [01-architecture.md](./01-architecture.md) 与 [03-http-api.md](./03-http-api.md)。
- 实现业务端点：再读 [03-http-api-target.md](./03-http-api-target.md)，实现时收敛对应 draft 并补充证据。
- 修改订单业务：先读 [05-domain-model.md](./05-domain-model.md)，再核对对应场景和权限矩阵。
- 修改协议消费：先读 [02-schema-ui-integration.md](./02-schema-ui-integration.md) 和 Schema-UI 总纲。
- 联调前端：以 Schema-UI 的数据源、Action 与场景文档为契约，再记录本仓业务端点细节。
- 开始实施目标能力：按 [06-implementation-roadmap.md](./06-implementation-roadmap.md) 的依赖顺序推进。
- 提交前：执行 [04-validation.md](./04-validation.md) 中对应门禁。
