---
id: GOAL-007-notification-api
title: 通知 API 与种子数据
status: active
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.2.0
progress: 20%
---

# GOAL-007 · 通知 API 与种子数据

## 概述

实现 GOAL-002 M3c 的通知业务切片：通知领域与状态机、SQLite 持久化和种子、服务用例、Bearer/RBAC HTTP API，以及可核对的跨层测试。Page Schema 不在本目标范围内。

## 成功标准

- [x] 通知领域模型包含 title/body、channel（inbox|email）、draft/published/archived 状态、version 与 publishedAt
- [ ] Repository port、service 用例和 SQLite 实现遵守既有 IoC 边界（N1：port + service 已完成；SQLite 待 N2）
- [ ] list/detail/create/update/delete/publish/batch-archive API 完成
- [x] 不真发邮件；channel 仅为枚举字段（领域/service 无外发依赖）
- [ ] viewer 只读，admin/operator 可写；全部通知路由需要 Bearer
- [ ] 空库种子幂等，至少覆盖 draft/published/archived 演示数据
- [ ] service、SQLite 与 HTTP 集成测试覆盖状态、CAS、RBAC、批量原子性和错误 envelope（service 接口测试已覆盖 N1 范围）
- [ ] `go test -count=1 ./...`、`go vet ./...`、`git diff --check` 通过

## 范围边界

- 状态机为 `draft → published → archived`；不提供 unpublish / restore。
- channel=`email` 仅作演示枚举，**不**接入真实邮件/短信通道。
- Page Schema、仪表盘聚合与协议制品校验属于 GOAL-002 后续阶段。

## 高层路线图

| 阶段 | 状态 | 说明 |
|------|------|------|
| N0 实施契约审视 | **完成** | D-003 固定跨层契约；A-001 design-plan 自审 pass |
| N1 领域/port/service | **完成** | domain、NotificationRepository port、全部 service 用例与 fake 接口测试 |
| N2 SQLite/seed | 未开始 | |
| N3 HTTP/RBAC | 未开始 | |
| N4 验证与自审 | 未开始 | |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 通知首切片完整 HTTP 契约：`/v1` 路由、筛选/分页、可编辑字段、version CAS、publish/delete/batch-archive、错误码 | N1～N3 实施 | **N1 前** | D-003 冻结 + design-plan 自审 | **verified** | — | D-003；A-001 pass；N1～N3 门禁解除 |
| I-002 | non-blocking | 通知 Page Schema 的具体 action/表单映射 | GOAL-002 M4 | M4 前 | 父目标后续设计 | open | 不阻断本 API 目标 | 由 GOAL-002 I-010/M4 统一处理 |

## 父目标

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md)

## 依赖

- [GOAL-003-modular-ioc-foundation](../GOAL-003-modular-ioc-foundation/00-meta.md) — done
- [GOAL-004-auth-rbac-menu](../GOAL-004-auth-rbac-menu/00-meta.md) — done

## 备注

- 2026-07-25：N0 完成（A-001 pass，I-001 verified）；N1 领域/port/service 与接口测试完成；progress **20%**。
- N2 SQLite/seed、N3 HTTP/RBAC 尚未开始。
