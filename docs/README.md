# 后端项目文档规则

本目录记录 `allinme.core-api` 自身的架构、HTTP API、Schema-UI 接入、验证方式、场景说明、架构决策和审计过程。

## 权威边界

文档按以下优先级解释：

1. 相邻 [`schema-ui-docs`](../../schema-ui-docs/README.md) 的当前稳定协议文档、JSON Schema、组件注册 DSL 与 conformance fixtures，是前后端对接的唯一核心契约。
2. 本仓源码和测试是后端实现行为的权威来源。
3. 本目录只描述本仓如何提供业务 API 和消费契约；不得复制、删改或另行解释协议字段。
4. 本仓文档与 Schema-UI 冲突时，以 Schema-UI 为准，并通过协议仓的 ADR/版本流程变更契约。

## 文档分类

| 分类 | 目录/文件 | 职责 |
|---|---|---|
| 总纲 | [`00-overview.md`](./00-overview.md) | 项目边界、模块地图、阅读入口 |
| 架构 | [`01-architecture.md`](./01-architecture.md) | Go 进程、HTTP 层与协议模块职责 |
| 契约接入 | [`02-schema-ui-integration.md`](./02-schema-ui-integration.md) | 协议来源、固定版本、API 边界与升级纪律 |
| API | [`03-http-api.md`](./03-http-api.md) | 本仓实际 HTTP 端点与错误约定 |
| 验证 | [`04-validation.md`](./04-validation.md) | test/vet 与 conformance 门禁 |
| 场景 | [`scenarios/`](./scenarios/README.md) | 本后端如何支持业务工作流，不复制协议 YAML |
| 决策 | [`decisions/`](./decisions/README.md) | 仅记录本仓实现选择，不替代协议 ADR |
| 审计 | [`audit/`](./audit/README.md) | 活跃审计、编号和归档生命周期 |
| 变更 | [`CHANGELOG.md`](./CHANGELOG.md) | 本仓 API 或集成行为变更 |

## 写作规则

- 每份规范性文档使用 frontmatter，至少包含 `status`、`owner`、`last_updated` 和 `applies_to`。
- HTTP 文档必须链接到 handler 和测试，并只声明当前实现或明确标注 planned。
- 不复制 Schema-UI 的字段表、Schema 或场景 YAML；使用稳定文件链接和固定 commit/tag 表达依赖。
- 新增跨前后端字段或语义时，先在 `schema-ui-docs` 修改协议、Schema、fixtures 和 ADR，再更新消费者。
- 仅影响本仓内部实现的重大选择写入 `decisions/`，文件名为 `NNNN-short-title.md`，编号全局递增。
- 文档变化应与代码变化同一提交，运行 [`04-validation.md`](./04-validation.md) 中与影响范围匹配的检查。
- 历史审计不得改写为当前规范；当前行为以总纲、架构、API、接入文档和源码为准。

## 维护流程

1. 从 [`00-overview.md`](./00-overview.md) 确认受影响边界。
2. 跨仓契约变化先走 Schema-UI 版本与 conformance 流程。
3. 更新本仓实现、测试、相关文档和 CHANGELOG。
4. 执行本地门禁；协议 pin 变化还需等待当前提交的远端 CI 成功。
5. 全量复审使用 `.github/prompts/backend-full-audit-cycle.prompt.md`，归档必须经用户确认。