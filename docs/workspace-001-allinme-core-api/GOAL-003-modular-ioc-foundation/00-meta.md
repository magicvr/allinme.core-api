---
id: GOAL-003-modular-ioc-foundation
title: 模块化 IoC 骨架（可换实现）
status: done
parent: GOAL-001-allinme-core-api
created: 2026-07-24
updated: 2026-07-24
version: 0.4.0
progress: 100%
---

# GOAL-003 · 模块化 IoC 骨架（可换实现）

## 概述

在业务三域大规模编码之前，落地 **高内聚、低耦合** 的 Go 模块骨架：以 **构造注入（IoC）** 与 **接口端口** 定义模块边界，使持久化等实现可替换且不修改调用方。原则权威见 Root **D-008（P-M1～P-M8）**。

本目标**不**交付完整订单/钱包/通知业务与 Admin page schema 全集（属 GOAL-002）。

**下游交接**：H1～H7 已勾选；GOAL-002 **I-009 → verified**（2026-07-24 关门）。

## 成功标准

- [x] 存在文档化模块图与依赖方向说明（attachments 定稿 active），与 P-M1～P-M8 一致（→ H5）
- [x] `cmd/server`（或约定 composition root）为**唯一**组装根；业务包不互相 `New` 具体实现（→ H1）
- [x] 出站端口 **`MetaStore`** + **SQLite 实现** + fake/memory 测试实现（→ H2/H3；D-004）
- [x] 配置支持 SQLite 路径；进程可启动；既有 `/healthz` `/readyz` `/v1/ping` 仍可用（→ H4）
- [x] 至少 1 个测试：service 仅依赖接口，无 SQLite 时用 fake 跑通（→ H3）
- [x] README 或 docs 简述「如何新增业务模块 / 如何换 Repository」；可含空 BC 目录占位（→ H6；D-005）
- [x] 不引入重型 DI 容器（→ H7）
- [x] 交接清单 H1～H7 正式勾选并关闭 GOAL-002 I-009（A-003 pass + A-004 self 关门；2026-07-24）

## 高层路线图

| 阶段 | 名称 | 状态 |
|------|------|------|
| S1 | 模块图与包布局决策 | **完成** |
| S2 | 目录与端口/接口落地 | **完成** |
| S3 | SQLite 适配器 + composition root 接线 | **完成**（I-003：modernc.org/sqlite） |
| S4 | 测试 + 文档 + 交接勾选 + 验收 | **完成**（关门 2026-07-24） |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 包布局与端口放置 | S2 | S2 前 | D-001 | **decided** | — | D-001 |
| I-002 | non-blocking | 是否引入 wire | S3 | S3 | 决策 | **decided** | — | 不引入 |
| I-003 | non-blocking | SQLite 驱动库选型 | S3 接线 | S3 当日 | execution | **verified** | — | modernc.org/sqlite v1.54.0 |
| I-004 | required | 垂直切片端口形态 | S2 | S2 前 | D-004 | **decided** | — | MetaStore |

## 父目标

- [GOAL-001-allinme-core-api](../GOAL-001-allinme-core-api/00-meta.md)

## 下游

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md) — I-009 **verified**

## 备注

- **`status: done`**（2026-07-24）：A-003 independent pass；A-004 self 关门；用户确认响应并关闭 I-009。
