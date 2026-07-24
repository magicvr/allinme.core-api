---
id: GOAL-003-modular-ioc-foundation
title: 模块化 IoC 骨架（可换实现）
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-24
updated: 2026-07-24
version: 0.2.0
progress: 5%
---

# GOAL-003 · 模块化 IoC 骨架（可换实现）

## 概述

在业务三域大规模编码之前，落地 **高内聚、低耦合** 的 Go 模块骨架：以 **构造注入（IoC）** 与 **接口端口** 定义模块边界，使持久化等实现可替换且不修改调用方。原则权威见 Root **D-008（P-M1～P-M8）**。

本目标**不**交付完整订单/钱包/通知业务与 Admin page schema 全集（属 GOAL-002）。

**下游交接**：验收通过且 [handover-to-goal-002.md](attachments/handover-to-goal-002.md) **H1～H7** 可勾选时，构成关闭 GOAL-002 **I-009** 的充分条件（见 D-003）。

## 成功标准

- [ ] 存在文档化模块图与依赖方向说明（attachments 定稿 active），与 P-M1～P-M8 一致（→ H5）
- [ ] `cmd/server`（或约定 composition root）为**唯一**组装根；业务包不互相 `New` 具体实现（→ H1）
- [ ] 出站端口 **`MetaStore`**（键值/元数据，供 ready 探测与可换存储证明）+ **SQLite 实现** + fake/memory 测试实现（→ H2/H3；D-004）
- [ ] 配置支持 SQLite 路径；进程可启动；既有 `/healthz` `/readyz` `/v1/ping` 仍可用（→ H4）
- [ ] 至少 1 个测试：service 仅依赖接口，无 SQLite 时用 fake 跑通（→ H3）
- [ ] README 或 docs 简述「如何新增业务模块 / 如何换 Repository」；可含空 BC 目录占位（无业务逻辑）（→ H6；D-005）
- [ ] 不引入重型 DI 容器（→ H7）
- [ ] 交接清单 [handover-to-goal-002.md](attachments/handover-to-goal-002.md) 与成功标准对齐，验收时填写 H1～H7

## 高层路线图

| 阶段 | 名称 | 状态 |
|------|------|------|
| S1 | 模块图与包布局决策 | **完成**（D-001；map 草案已有；A-001 响应后收尾） |
| S2 | 目录与端口/接口落地 | **下一步**（`MetaStore` + 空 BC 占位可选） |
| S3 | SQLite 适配器 + composition root 接线 | 未开始（当日关闭 I-003 驱动选型） |
| S4 | 测试 + 文档 + 交接勾选 + 验收 | 未开始 |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 包布局与端口放置 | S2 | S2 前 | D-001 | **decided** | — | D-001 |
| I-002 | non-blocking | 是否引入 wire | S3 | S3 | 决策 | **decided** | — | 不引入 |
| I-003 | non-blocking | SQLite 驱动库选型 | S3 接线 | **S3 当日** | 写入 execution 关闭 | open | **不得拖到 S4 验收争论** | 倾向 modernc/sqlite 或 database/sql+驱动 |
| I-004 | required | 垂直切片端口形态 | S2 | S2 前 | D-004 | **decided** | — | **MetaStore**（非二选一悬空） |

## 父目标

- [GOAL-001-allinme-core-api](../GOAL-001-allinme-core-api/00-meta.md)

## 下游

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md) — I-009；交接 [handover-to-goal-002.md](attachments/handover-to-goal-002.md)

## 备注

- Root **R0.8**。2026-07-24：A-001 审计响应（F-001 交接契约等）。
