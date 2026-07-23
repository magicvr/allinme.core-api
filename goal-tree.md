---
title: 目标树
status: active
created: 2026-07-23
updated: 2026-07-23
parent: null
version: 0.2.0
---

# 目标树 · allinme.core-api

> 工作区：隐式单工作区（仓库根）。真相源为本文件 + 各 `GOAL-*/` 五件套。

## 树

```text
GOAL-001-allinme-core-api          [blocked  5%]  可复用的 Schema-UI 核心 API 基座
└── GOAL-002-mvp-demo-admin        [blocked  0%]  MVP · Demo 完整 Admin（协议驱动）
```

## 状态表

| ID | 标题 | parent | status | progress | 备注 |
|----|------|--------|--------|----------|------|
| GOAL-001-allinme-core-api | 可复用的 Schema-UI 核心 API 基座 | `null` | **blocked** | 5% | 等 `schema-ui-docs` 新协议（I-006）；策略 A |
| GOAL-002-mvp-demo-admin | MVP · Demo 完整 Admin（协议驱动） | GOAL-001-allinme-core-api | **blocked** | 0% | 等新协议（I-008）；批量走协议演进 |

## 路线图摘要（Root）

| 阶段 | 状态 | 对应 |
|------|------|------|
| R0 治理与边界冻结 | 暂停 | 立项完成；钉死改等新协议 |
| **R0.5 协议演进（外仓）** | **进行中（schema-ui-docs）** | 本仓不实施 |
| R1 MVP Demo Admin | **阻塞** | GOAL-002 |
| R2 协议对齐与复用沉淀 | 未开始 | 待建 |
| R3 多项目消费就绪 | 未开始 | 待建 |

## 阻塞与恢复

| 项 | 内容 |
|----|------|
| **原因** | 用户选 A：先演进协议；本仓暂停 |
| **外仓** | `magicvr/schema-ui-docs`（批量等能力进核心协议并发布新制品） |
| **门禁** | Root I-006 / GOAL-002 I-008 |
| **恢复** | 新协议可固定引用 → 钉死版本决策 → 用户确认 → `/govern` 解除 blocked 并推进 R1 |

## 开放门禁提示

- **I-001（策略）**：已 decided = A。
- **I-006 / I-008**：新协议制品 collecting（阻断解除 blocked）。
- **I-002～I-005（GOAL-002）**：暂停主动收集，待新协议后再推进。
