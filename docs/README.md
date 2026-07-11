# 后端项目文档规则

本目录记录 `allinme.core-api` 自身的架构、HTTP API、Schema-UI 接入、验证方式、场景说明、架构决策和审计过程。

## 权威边界

文档按以下优先级解释：

1. 相邻 [`schema-ui-docs`](../../schema-ui-docs/README.md) 的当前稳定协议文档、JSON Schema、组件注册 DSL 与 conformance fixtures，是前后端对接的唯一核心契约。
2. 本仓源码和测试是后端实现行为的权威来源。
3. 本目录只描述本仓如何提供业务 API 和消费契约；不得复制、删改或另行解释协议字段。
4. 本仓文档与 Schema-UI 冲突时，以 Schema-UI 为准，并通过协议仓的 ADR/版本流程变更契约。

## 状态词汇

| status | 适用对象 | 含义 |
|---|---|---|
| `active` | 规范文档 | 已由当前源码和测试实现，是现行规范 |
| `target` | 目标规范 | 已完成方向设计但尚未全部实现；标记为 `draft` 的细节仍可调整 |
| `planned` | 路线与场景 | 描述实施计划或验收意图，细节可随实现收敛 |
| `accepted` | ADR | 决策已采纳，不表示对应能力已经实现 |
| `deprecated` | 规范文档 | 仍可兼容但不应新增使用 |
| `superseded` | ADR / 历史规范 | 已被明确的新决策或规范替代，仅保留历史 |

文档不得使用文件级 `active` 同时承载尚未实现的目标契约。目标能力只有在实现、测试、文档和对应门禁全部启用后才能改为 `active`。

## 单一事实源

| 信息 | 唯一维护位置 | 其他文档规则 |
|---|---|---|
| Schema-UI 固定版本与升级流程 | [`02-schema-ui-integration.md`](./02-schema-ui-integration.md) | 只链接，不重复固定 SHA |
| 当前 HTTP 行为 | [`03-http-api.md`](./03-http-api.md) + 源码/测试 | 只引用当前端点与证据 |
| 目标 endpoint 草案 | [`03-http-api-target.md`](./03-http-api-target.md) | 场景和路线只引用，不复制完整清单 |
| 角色、状态机与业务不变量 | [`05-domain-model.md`](./05-domain-model.md) | API、场景和 ADR 只描述自身影响 |
| 页面源路径与嵌入策略 | [`01-architecture.md`](./01-architecture.md) | ADR 只解释选择原因，路线只引用实施步骤 |
| 错误码草案与冻结结果 | [`03-http-api-target.md`](./03-http-api-target.md) → [`03-http-api.md`](./03-http-api.md) | 场景只描述错误语义，不冻结名称 |
| 验证命令与门禁启用状态 | [`04-validation.md`](./04-validation.md) | 路线只引用完成证据，不复制命令全集 |

ADR 记录“为何选择”，不替代上述现行或目标规范。CHANGELOG 只记录变化，不作为当前值的查询入口。

## 文档分类

| 分类 | 目录/文件 | 职责 |
|---|---|---|
| 总纲 | [`00-overview.md`](./00-overview.md) | 项目边界、模块地图、阅读入口 |
| 架构 | [`01-architecture.md`](./01-architecture.md) | Go 进程、HTTP 层与协议模块职责 |
| 契约接入 | [`02-schema-ui-integration.md`](./02-schema-ui-integration.md) | 协议来源、固定版本、API 边界与升级纪律 |
| 当前 API | [`03-http-api.md`](./03-http-api.md) | 已实现 HTTP 端点与验证证据 |
| 目标 API | [`03-http-api-target.md`](./03-http-api-target.md) | 待实现的原则与 endpoint 草案 |
| 验证 | [`04-validation.md`](./04-validation.md) | test/vet 与 conformance 门禁 |
| 领域 | [`05-domain-model.md`](./05-domain-model.md) | 订单、履约、退款、附件与权限目标 |
| 路线 | [`06-implementation-roadmap.md`](./06-implementation-roadmap.md) | 目标能力实施顺序与完成标准 |
| 场景 | [`scenarios/`](./scenarios/README.md) | 本后端如何支持业务工作流，不复制协议 YAML |
| 决策 | [`decisions/`](./decisions/README.md) | 仅记录本仓实现选择，不替代协议 ADR |
| 审计 | [`audit/`](./audit/README.md) | 活跃审计、编号和归档生命周期 |
| 变更 | [`CHANGELOG.md`](./CHANGELOG.md) | 本仓 API 或集成行为变更 |

## 写作规则

- 每份规范性文档使用 frontmatter，至少包含 `status`、`owner`、`last_updated` 和 `applies_to`。
- 当前 HTTP 文档必须链接到 handler 和测试；目标 HTTP 文档必须使用 `status: target`，并区分已冻结原则与 draft endpoint。
- 不复制 Schema-UI 的字段表、Schema 或场景 YAML；使用稳定文件链接和固定 commit/tag 表达依赖。
- 新增跨前后端字段或语义时，先在 `schema-ui-docs` 修改协议、Schema、fixtures 和 ADR，再更新消费者。
- 仅影响本仓内部实现的重大选择写入 `decisions/`，文件名为 `NNNN-short-title.md`，编号全局递增。
- 文档变化应与代码变化同一提交，运行 [`04-validation.md`](./04-validation.md) 中与影响范围匹配的检查。
- 历史审计不得改写为当前规范；当前行为以总纲、架构、API、接入文档和源码为准。

## 维护流程

1. 从 [`00-overview.md`](./00-overview.md) 确认受影响边界。
2. 跨仓契约变化先走 Schema-UI 版本与 conformance 流程。
3. 更新本仓实现、测试、事实源文档及引用方，并在 CHANGELOG 记录变化。
4. 执行本地门禁；协议 pin 变化还需等待当前提交的远端 CI 成功。
5. 全量复审使用 `.github/prompts/backend-full-audit-cycle.prompt.md`，归档必须经用户确认。