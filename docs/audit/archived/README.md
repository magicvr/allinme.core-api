# 已归档后端审计

归档按编号倒序记录审计主题、文件链接、关键修复和验证证据。

- `0004` 阶段四退款与经营看板：[plan](./0004-2026-07-13-plan.md) / [checklist](./0004-2026-07-13-checklist.md)。实现 v6 退款 schema、幂等申请、审批/拒绝、订单退款不变量、退款 HTTP 闭环和经营看板；schema-only/旧 v5 binary 回退、fresh-data smoke、全仓 test/vet/race、文档门禁及 PR #4 远端 `test`/`race` CI 均通过。
- `0003` 阶段三订单查询与履约：[plan](./0003-2026-07-12-plan.md) / [checklist](./0003-2026-07-12-checklist.md)。实现订单查询、幂等创建、草稿编辑、五个履约 Action、聚合完整性、CORS 与 shutdown 加固；PR #2 及 `main@de1b2219` 的 `test`/`race` CI 均通过。
- `0002` 阶段二认证授权：[plan](./0002-2026-07-12-plan.md) / [checklist](./0002-2026-07-12-checklist.md)。实现本地账号、严格 JWT Bearer、SQLite 可撤销 session、四角色授权、development seed、production bootstrap-admin、登录限流和三条认证接口；`go test ./... -count=1`、`go vet ./...`、`go test -race ./...` 及文档门禁均通过。
- `0001` 阶段一 1A 运行基础：[plan](./0001-2026-07-12-plan.md) / [checklist](./0001-2026-07-12-checklist.md)。实现配置与应用装配、SQLite migration/seed/reset、HTTP readiness 与运行中间件；`go test ./... -count=1`、`go vet ./...`、`go test -race ./...` 及 Windows smoke 均通过。
