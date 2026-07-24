---
id: GOAL-006-wallet-api
title: 钱包 API 与种子数据
status: done
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.6.0
progress: 100%
---

# GOAL-006 · 钱包 API 与种子数据

## 概述

实现 GOAL-002 M3 的钱包业务切片：钱包领域与状态机、SQLite 持久化和种子、服务用例、Bearer/RBAC HTTP API，以及可核对的跨层测试。Page Schema 不在本目标范围内。

## 成功标准

- [x] 钱包领域模型包含账户、余额、币种、active/frozen 状态与 version
- [x] Repository port、service 用例和 SQLite 实现遵守既有 IoC 边界
- [x] list/detail/create/update/freeze/unfreeze/batch-freeze API 完成
- [x] `PUT` 仅更新允许的元数据，不可修改 `balanceCents`
- [x] viewer 只读，admin/operator 可写；全部钱包路由需要 Bearer
- [x] 空库种子幂等，至少覆盖 active/frozen 演示数据
- [x] service、SQLite 与 HTTP 集成测试覆盖状态、CAS、RBAC、批量原子性和错误 envelope
- [x] `go test -count=1 ./...`、`go vet ./...`、`git diff --check` 通过

## 范围边界

- 创建时可设置初始 `balanceCents`。
- 不提供调账、充值、提现或支付网关。
- Page Schema、仪表盘聚合与协议制品校验属于 GOAL-002 后续阶段。

## 高层路线图

| 阶段 | 状态 | 说明 |
|------|------|------|
| W0 实施契约审视 | **完成** | D-003 固定跨层契约；A-001 design-plan 自审 pass |
| W1 领域/port/service | **完成** | domain、WalletRepository port、全部 service 用例与 fake 接口测试 |
| W2 SQLite/seed | **完成** | wallets schema、CAS/错误分类、事务 batch-freeze、事务 seed 与 repository 测试 |
| W3 HTTP/RBAC | **完成** | composition root + startup seed、七端点 handler/RBAC、跨层 HTTP 与 internal 不泄露测试 |
| W4 验证与自审 | **完成** | 强制验证 pass；A-002 execution-facts/close-out 自审 pass；无 findings |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 钱包首切片完整 HTTP 契约：`/v1` 路由、筛选/分页、可编辑字段、version CAS、状态动作、batch-freeze 原子性、错误码 | W1～W3 实施 | **W1 前** | 在本目标 decision 固定并做 design-plan 自审 | **verified** | — | D-003；A-001 pass；W1～W3 门禁解除 |
| I-002 | non-blocking | 钱包 Page Schema 的具体 action/表单映射 | GOAL-002 M4 | M4 前 | 父目标后续设计 | open | 不阻断本 API 目标 | 由 GOAL-002 I-010/M4 统一处理 |

## 父目标

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md)

## 依赖

- [GOAL-003-modular-ioc-foundation](../GOAL-003-modular-ioc-foundation/00-meta.md) — done
- [GOAL-004-auth-rbac-menu](../GOAL-004-auth-rbac-menu/00-meta.md) — done

## 备注

- W0～W4 全部完成；A-002 close-out 自审 pass，无 required/recommended finding，目标 `done` / 100%。
- composition root、SQLite/seed、七个 API、Bearer/RBAC、稳定错误与跨层测试均有可重复证据。
- 强制 test/vet/diff checks 通过；额外 race 因本机 Windows cgo 工具链失败未完成，已在 A-002 如实记录为非阻断验证限制。
- I-001 verified；I-002 仍为 non-blocking/open，由父目标 M4 处理，不属于本 API 子目标关门范围。
