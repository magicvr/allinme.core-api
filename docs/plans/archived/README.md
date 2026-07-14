# 已归档实施计划

本目录保存稳定路径归档规则生效前已经移动的历史计划，按 `PLN` 编号倒序记录。新计划归档时保留在 `docs/plans/` 原路径，只修改 plan/checklist 的 `status: archived`，避免破坏不可变审计与实施记录中的链接。归档表示实施生命周期已结束，不表示计划正文是当前规范；当前行为仍以源码、测试和事实源文档为准。

- `PLN-0004` 阶段四退款与经营看板：[plan](./PLN-0004-phase-04-refunds-dashboard.md) / [checklist](./PLN-0004-phase-04-refunds-dashboard-checklist.md)。实现 v6 退款 schema、幂等申请、审批/拒绝、订单退款不变量、退款 HTTP 闭环和经营看板；PR #4 已合入 `main@34b58c0`。
- `PLN-0003` 阶段三订单查询与履约：[plan](./PLN-0003-phase-03-order-fulfillment.md) / [checklist](./PLN-0003-phase-03-order-fulfillment-checklist.md)。实现订单查询、幂等创建、草稿编辑、履约 Action、聚合完整性、CORS 与 shutdown 加固。
- `PLN-0002` 阶段二认证授权：[plan](./PLN-0002-phase-02-authentication-authorization.md) / [checklist](./PLN-0002-phase-02-authentication-authorization-checklist.md)。实现本地账号、JWT Bearer、可撤销 session、四角色授权、登录限流和认证 API。
- `PLN-0001` 阶段一 1A 运行基础：[plan](./PLN-0001-phase-01-runtime-foundation.md) / [checklist](./PLN-0001-phase-01-runtime-foundation-checklist.md)。实现配置装配、SQLite migration/seed/reset、readiness 与运行中间件。
