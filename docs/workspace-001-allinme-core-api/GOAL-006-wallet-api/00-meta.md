---
id: GOAL-006-wallet-api
title: 钱包 API 与种子数据
status: active
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.1.0
progress: 0%
---

# GOAL-006 · 钱包 API 与种子数据

## 概述

实现 GOAL-002 M3 的钱包业务切片：钱包领域与状态机、SQLite 持久化和种子、服务用例、Bearer/RBAC HTTP API，以及可核对的跨层测试。Page Schema 不在本目标范围内。

## 成功标准

- [ ] 钱包领域模型包含账户、余额、币种、active/frozen 状态与 version
- [ ] Repository port、service 用例和 SQLite 实现遵守既有 IoC 边界
- [ ] list/detail/create/update/freeze/unfreeze/batch-freeze API 完成
- [ ] `PUT` 仅更新允许的元数据，不可修改 `balanceCents`
- [ ] viewer 只读，admin/operator 可写；全部钱包路由需要 Bearer
- [ ] 空库种子幂等，至少覆盖 active/frozen 演示数据
- [ ] service、SQLite 与 HTTP 集成测试覆盖状态、CAS、RBAC、批量原子性和错误 envelope
- [ ] `go test -count=1 ./...`、`go vet ./...`、`git diff --check` 通过

## 范围边界

- 创建时可设置初始 `balanceCents`。
- 不提供调账、充值、提现或支付网关。
- Page Schema、仪表盘聚合与协议制品校验属于 GOAL-002 后续阶段。

## 高层路线图

| 阶段 | 状态 | 说明 |
|------|------|------|
| W0 实施契约审视 | **当前** | 固定路由、字段、状态/CAS、批量与错误语义 |
| W1 领域/port/service | 未开始 | 先用接口测试固定用例 |
| W2 SQLite/seed | 未开始 | schema、CAS、事务、幂等种子 |
| W3 HTTP/RBAC | 未开始 | handler、路由与集成测试 |
| W4 验证与自审 | 未开始 | 命令验证并对照成功标准 |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 钱包首切片完整 HTTP 契约：`/v1` 路由、筛选/分页、可编辑字段、version CAS、状态动作、batch-freeze 原子性、错误码 | W1～W3 实施 | **W1 前** | 在本目标 decision 固定并做 design-plan 自审 | **open** | 实施前关闭 | 父目标 D-008/D-009/D-015 仅给出轮廓，仍需细化 |
| I-002 | non-blocking | 钱包 Page Schema 的具体 action/表单映射 | GOAL-002 M4 | M4 前 | 父目标后续设计 | open | 不阻断本 API 目标 | 由 GOAL-002 I-010/M4 统一处理 |

## 父目标

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md)

## 依赖

- [GOAL-003-modular-ioc-foundation](../GOAL-003-modular-ioc-foundation/00-meta.md) — done
- [GOAL-004-auth-rbac-menu](../GOAL-004-auth-rbac-menu/00-meta.md) — done

## 备注

- 当前只完成立项与边界登记，尚未实施钱包代码。
- I-001 为到期实施门禁；下一步先形成并自审实施契约，不应直接编码。
