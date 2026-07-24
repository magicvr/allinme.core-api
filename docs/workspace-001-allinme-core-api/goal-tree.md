---
title: 目标树
status: active
created: 2026-07-23
updated: 2026-07-24
parent: null
version: 0.9.0
---

# 目标树 · allinme.core-api

> 工作区：`docs/workspace-001-allinme-core-api/`（`workspace.md`）。真相源为本文件 + 各 `GOAL-*` 五件套。

## 树

```text
GOAL-001-allinme-core-api               [active  25%]  可复用的 Schema-UI 核心 API 基座
├── GOAL-003-modular-ioc-foundation     [done   100%]  模块化 IoC 骨架 · R0.8
└── GOAL-002-mvp-demo-admin             [active  40%]  MVP Admin · R1（M2 完成；下一步 M3）
```

## 状态表

| ID | 标题 | parent | status | progress | 备注 |
|----|------|--------|--------|----------|------|
| GOAL-001-allinme-core-api | 可复用的 Schema-UI 核心 API 基座 | `null` | **active** | 25% | R0.8 完成；R1 M2 鉴权完成 |
| GOAL-003-modular-ioc-foundation | 模块化 IoC 骨架（可换实现） | GOAL-001-allinme-core-api | **done** | 100% | 已关门 |
| GOAL-002-mvp-demo-admin | MVP · Demo 完整 Admin（协议驱动） | GOAL-001-allinme-core-api | **active** | 40% | JWT+菜单；下一步三域 API |

## 路线图摘要（Root）

| 阶段 | 状态 | 对应 |
|------|------|------|
| R0 治理与边界冻结 | 完成（阶段） | 立项 + 协议钉死 |
| R0.5 协议演进（外仓） | 完成 | schema-ui-docs v2.4.1 |
| **R0.8 模块化 IoC 骨架** | **完成** | GOAL-003 done |
| R1 MVP Demo Admin | **进行中** | GOAL-002；M2 完成；M3 下一步 |
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

## 开放门禁提示

- **GOAL-002 M3**：三域 API（下一步）。
- **GOAL-002 I-010**：协议制品落仓/校验；阻断 M4 校验宣称。
- GOAL-003：**已关门**。
