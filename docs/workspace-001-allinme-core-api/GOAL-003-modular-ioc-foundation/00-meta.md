---
id: GOAL-003-modular-ioc-foundation
title: 模块化 IoC 骨架（可换实现）
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-24
updated: 2026-07-24
version: 0.1.0
progress: 0%
---

# GOAL-003 · 模块化 IoC 骨架（可换实现）

## 概述

在业务三域大规模编码之前，落地 **高内聚、低耦合** 的 Go 模块骨架：以 **构造注入（IoC）** 与 **接口端口** 定义模块边界，使持久化等实现可替换且不修改调用方。原则权威见 Root **D-008（P-M1～P-M8）**。

本目标**不**交付完整订单/钱包/通知业务与 Admin page schema 全集（属 GOAL-002）。

## 成功标准

- [ ] 存在文档化模块图与依赖方向说明（本目标 attachments 或 `docs/` 短文），与 P-M1～P-M8 一致
- [ ] `cmd/server`（或约定 composition root）为**唯一**组装根；业务包不互相 `New` 具体实现
- [ ] 至少一条出站端口接口（如健康/探活用的 `HealthStore` 或占位 `ExampleRepository`）+ **SQLite 实现** + 可替换的 fake/memory 测试实现
- [ ] 配置支持 SQLite 路径；进程可启动；既有 `/healthz` `/readyz` `/v1/ping` 仍可用
- [ ] 至少 1 个测试证明：service 仅依赖接口，可在无 SQLite 时用 fake 跑通
- [ ] README 或 docs 简述「如何新增一个业务模块 / 如何换 Repository 实现」
- [ ] 不在本目标引入重型 DI 容器（手动注入；wire 可选且非必须）

## 高层路线图

| 阶段 | 名称 | 状态 |
|------|------|------|
| S1 | 模块图与包布局决策 | 进行中（D-001） |
| S2 | 目录与端口/接口落地 | 未开始 |
| S3 | SQLite 适配器 + composition root 接线 | 未开始 |
| S4 | 测试 + 文档 + 验收 | 未开始 |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 包布局与端口放置约定（port 集中 vs 按 BC） | S2 实施 | S2 前 | D-001 | **decided** | — | 见 D-001 |
| I-002 | non-blocking | 是否引入 google/wire | S3 接线 | S3 | 决策 | **decided** | — | MVP **不**引入；手动注入（Root P-M7） |
| I-003 | non-blocking | SQLite 驱动库选型（database/sql + modernc vs mattn） | S3 | S3 | 实施时选定并记 execution | open | 实施时关闭 | 纯 Go 优先倾向 modernc/sqlite 或标准 database/sql |

## 父目标

- [GOAL-001-allinme-core-api](../GOAL-001-allinme-core-api/00-meta.md)

## 下游

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md) — 业务实施依赖本目标可验收（GOAL-002 I-009）

## 备注

- 对应 Root 路线图 **R0.8**。
- 用户 2026-07-24 方案包 A 确认创建。
