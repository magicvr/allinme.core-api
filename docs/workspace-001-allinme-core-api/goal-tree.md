---
title: 目标树
status: active
created: 2026-07-23
updated: 2026-07-24
parent: null
version: 0.4.0
---

# 目标树 · allinme.core-api

> 工作区：`docs/workspace-001-allinme-core-api/`（`workspace.md`）。真相源为本文件 + 各 `GOAL-*/` 五件套。

## 树

```text
GOAL-001-allinme-core-api          [active   10%]  可复用的 Schema-UI 核心 API 基座
└── GOAL-002-mvp-demo-admin        [active    5%]  MVP · Demo 完整 Admin（协议驱动）
```

## 状态表

| ID | 标题 | parent | status | progress | 备注 |
|----|------|--------|--------|----------|------|
| GOAL-001-allinme-core-api | 可复用的 Schema-UI 核心 API 基座 | `null` | **active** | 10% | 钉死协议 2.4.1（D-006）；I-006 verified |
| GOAL-002-mvp-demo-admin | MVP · Demo 完整 Admin（协议驱动） | GOAL-001-allinme-core-api | **active** | 5% | I-008 verified；推进 I-002～I-005 |

## 路线图摘要（Root）

| 阶段 | 状态 | 对应 |
|------|------|------|
| R0 治理与边界冻结 | **完成（阶段）** | 立项 + 协议钉死 2.4.1 |
| R0.5 协议演进（外仓） | **完成** | `schema-ui-docs` v2.4.1 |
| R1 MVP Demo Admin | **进行中** | GOAL-002 |
| R2 协议对齐与复用沉淀 | 未开始 | 待建 |
| R3 多项目消费就绪 | 未开始 | 待建 |

## 协议钉死（当前）

| 项 | 值 |
|----|-----|
| 制品 | `schema-ui-protocol` **2.4.1** |
| protocolVersion | `"2.4"` |
| tag | `v2.4.1` |
| artifact SHA-256 | `c027fa6c5b4bcb379a2fc90f6447f0e8df0729df5657fcb5d6a382d9ee3fbb18` |
| 决策 | Root D-006；GOAL-002 D-006 |
| Release | https://github.com/magicvr/schema-ui-docs/releases/tag/v2.4.1 |

## 开放门禁提示

- **I-001 / I-006（Root）/ I-008（GOAL-002）**：已关闭（策略 A + 2.4.1 verified）。
- **GOAL-002 I-002～I-005**：open，阻断方案冻结与受影响实施；下一步优先收集。
- **Root I-002 / I-003**：open，随 R1 推进。
