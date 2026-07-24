---
title: 目标树
status: active
created: 2026-07-23
updated: 2026-07-24
parent: null
version: 0.6.0
---

# 目标树 · allinme.core-api

> 工作区：`docs/workspace-001-allinme-core-api/`（`workspace.md`）。真相源为本文件 + 各 `GOAL-*` 五件套。

## 树

```text
GOAL-001-allinme-core-api               [active  15%]  可复用的 Schema-UI 核心 API 基座
├── GOAL-003-modular-ioc-foundation     [active   5%]  模块化 IoC 骨架 · R0.8（S1 完；S2 下一步）
└── GOAL-002-mvp-demo-admin             [active  22%]  MVP Admin · R1（方案冻；A-001 已响应；I-009/I-010 open）
```

## 状态表

| ID | 标题 | parent | status | progress | 备注 |
|----|------|--------|--------|----------|------|
| GOAL-001-allinme-core-api | 可复用的 Schema-UI 核心 API 基座 | `null` | **active** | 15% | 协议 2.4.1；D-008 模块化 |
| GOAL-003-modular-ioc-foundation | 模块化 IoC 骨架（可换实现） | GOAL-001-allinme-core-api | **active** | 5% | **优先**；交接 H1～H7；MetaStore |
| GOAL-002-mvp-demo-admin | MVP · Demo 完整 Admin（协议驱动） | GOAL-001-allinme-core-api | **active** | 22% | A-001 响应完；I-009/I-010 open |

## 路线图摘要（Root）

| 阶段 | 状态 | 对应 |
|------|------|------|
| R0 治理与边界冻结 | 完成（阶段） | 立项 + 协议钉死 |
| R0.5 协议演进（外仓） | 完成 | schema-ui-docs v2.4.1 |
| **R0.8 模块化 IoC 骨架** | **进行中** | **GOAL-003** |
| R1 MVP Demo Admin | 进行中（方案冻 / 实施待 003） | GOAL-002 |
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

Root **D-008**：P-M1 Composition Root · P-M2 依赖倒置 · P-M3 接口隔离 · P-M4 高内聚分包 · P-M5 稳定依赖方向 · P-M6 可替换实现 · P-M7 手动构造注入 · P-M8 协议边界不变。

## 开放门禁提示

- **GOAL-003**：推进 **S2**（MetaStore / composition root）；S3 当日关 I-003 驱动选型。
- **GOAL-002 I-009**：H1～H7 全勾后 verified（见 handover 附件）；阻断 M2。
- **GOAL-002 I-010**：协议制品落仓/校验；阻断 M4 校验宣称。
- A-001 required findings：**已响应关闭**（A-002）；I-009/I-010 信息门禁仍开。
