---
title: 目标树
status: active
created: 2026-07-23
updated: 2026-07-25
parent: null
version: 0.18.0
---

# 目标树 · allinme.core-api

> 工作区：`docs/workspace-001-allinme-core-api/`（`workspace.md`）。真相源为本文件 + 各 `GOAL-*` 五件套。

## 树

```text
GOAL-001-allinme-core-api                   [active  25%]  可复用的 Schema-UI 核心 API 基座
├── GOAL-003-modular-ioc-foundation         [done   100%]  模块化 IoC 骨架 · R0.8
└── GOAL-002-mvp-demo-admin                 [active  60%]  MVP Admin · R1（渐进子目标）
    ├── GOAL-004-auth-rbac-menu             [done   100%]  鉴权、RBAC 与菜单闭环（补录）
    ├── GOAL-005-order-api-first-slice      [done   100%]  订单 API 首切片（补录）
    ├── GOAL-006-wallet-api                 [done   100%]  钱包 API 与种子数据
    └── GOAL-007-notification-api           [active  20%]  通知 API 与种子数据
```

## 状态表

| ID | 标题 | parent | status | progress | 备注 |
|----|------|--------|--------|----------|------|
| GOAL-001-allinme-core-api | 可复用的 Schema-UI 核心 API 基座 | `null` | **active** | 25% | R0.8 完成；R1 渐进推进 |
| GOAL-003-modular-ioc-foundation | 模块化 IoC 骨架（可换实现） | GOAL-001-allinme-core-api | **done** | 100% | 已关门 |
| GOAL-002-mvp-demo-admin | MVP · Demo 完整 Admin（协议驱动） | GOAL-001-allinme-core-api | **active** | 60% | 父目标；M3c 通知进行中（N1 完成） |
| GOAL-004-auth-rbac-menu | 鉴权、RBAC 与菜单闭环 | GOAL-002-mvp-demo-admin | **done** | 100% | 依据 2026-07-24 既有实施事实补录 |
| GOAL-005-order-api-first-slice | 订单 API 首切片 | GOAL-002-mvp-demo-admin | **done** | 100% | 不含单项 DELETE/refund |
| GOAL-006-wallet-api | 钱包 API 与种子数据 | GOAL-002-mvp-demo-admin | **done** | 100% | A-003 independent close-out pass；A-004 维持 done |
| GOAL-007-notification-api | 通知 API 与种子数据 | GOAL-002-mvp-demo-admin | **active** | 20% | A-001 pass；I-001 verified；N1 完成，N2 待 |

## 路线图摘要（Root）

| 阶段 | 状态 | 对应 |
|------|------|------|
| R0 治理与边界冻结 | 完成（阶段） | 立项 + 协议钉死 |
| R0.5 协议演进（外仓） | 完成 | schema-ui-docs v2.4.1 |
| **R0.8 模块化 IoC 骨架** | **完成** | GOAL-003 done |
| R1 MVP Demo Admin | **进行中** | GOAL-002；GOAL-004/005/006 done；GOAL-007 active |
| R2 协议对齐与复用沉淀 | 未开始 | 待建 |
| R3 多项目消费就绪 | 未开始 | 待建 |

## 协议钉死（当前）

| 项 | 值 |
|----|-----|
| 制品 | `schema-ui-protocol` **2.4.1** |
| protocolVersion | `"2.4"` |
| tag | `v2.4.1` |
| artifact SHA-256 | `c027fa6c5b4bcb379a2fc90f6447f0e8df0729df5657fcb5d6a382d9ee3fbb18` |
| 决策 | Root D-006 |

## 架构原则（摘要）

Root **D-008**：P-M1～P-M8；落地见 GOAL-003 done + [modular-ioc.md](../../architecture/modular-ioc.md)。

## 门禁与下一步提示

- **GOAL-006**：A-003 independent close-out pass；A-004 响应维持 `done` / 100%；I-002 保留给 GOAL-002 M4。
- **GOAL-007**：A-001 design-plan pass；I-001 verified；N1 domain/port/service 完成（20%）；下一步 N2 SQLite/seed。
- **GOAL-002 后续**：M3d 订单 DELETE/refund、M4 Page Schema/I-010、M5 验收尚未立项。
- GOAL-003、GOAL-004、GOAL-005：**已关门**。
