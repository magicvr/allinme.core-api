# 已归档后端审计

归档按编号倒序记录审计主题、文件链接、关键修复和验证证据。

- `0001` 阶段一 1A 运行基础：[plan](./0001-2026-07-12-plan.md) / [checklist](./0001-2026-07-12-checklist.md)。实现配置与应用装配、SQLite migration/seed/reset、HTTP readiness 与运行中间件；`go test ./... -count=1`、`go vet ./...`、`go test -race ./...` 及 Windows smoke 均通过。