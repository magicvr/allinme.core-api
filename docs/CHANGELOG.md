# Changelog

本文件记录 `allinme.core-api` 的 API、架构边界、协议 pin 和质量门禁变化。Schema-UI 协议本身的版本历史以 [`schema-ui-docs/docs/CHANGELOG.md`](../../schema-ui-docs/docs/CHANGELOG.md) 为准。

## Unreleased

### Added

- 实现阶段二认证授权：bcrypt 本地账号、严格 HS256 JWT、SQLite 可撤销 session、四角色策略和 login/me/logout API。
- 增加 development 四角色 auth seed、production 空库 `bootstrap-admin`、固定窗口登录限流和 migration v2。

- 实现阶段一运行基础：配置与应用装配、纯 Go SQLite、嵌入式 migration、runtime seed，以及 development-only reset。
- 增加 `GET /readyz`、统一运行错误 envelope、request ID、结构化访问日志和 panic recovery。

- 建立后端文档总纲、架构、Schema-UI 接入、HTTP API、验证、场景、ADR 与审计规则。
- 明确 Schema-UI 文档仓是前后端对接的核心契约，本仓文档不重新定义协议。
- 初始化订单运营 demo 的目标领域、状态机、角色权限、HTTP API、完整业务场景和分阶段实施路线。
- 记录 SQLite + 本地文件、本地 JWT + 可撤销 session、后端 YAML 页面 + JSON API 三项架构决策。
- 增加认证、幂等、并发、退款、附件、看板和页面联调的目标验证矩阵。
- 建立文档状态词汇和单一事实源规则，拆分当前 HTTP API 与 draft 目标 API。
- 明确 liveness/readiness、目标门禁启用状态，以及 `internal/pages/yaml/*.yaml` 的包内嵌入与校验链路。
- 收敛目标 API 的 baseline/draft 层级、场景错误语义和 endpoint 实现迁移流程。
- 建立阶段一 1A 运行基础开发计划与可执行 checklist，明确 migration/seed/reset、readiness 和错误框架的实施证据。
- 建立阶段二认证授权开发计划与可执行 checklist，冻结 auth migration/seed、JWT/session、角色策略、登录限流、安全验证和文档收敛边界。

### Changed

- 协议 fixture pin 的当前值统一由 Schema-UI 接入文档维护，README 和 CHANGELOG 不再复制 SHA。