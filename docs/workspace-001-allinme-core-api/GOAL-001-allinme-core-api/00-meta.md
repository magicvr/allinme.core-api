---
id: GOAL-001-allinme-core-api
title: 可复用的 Schema-UI 核心 API 基座
status: active
parent: null
created: 2026-07-23
updated: 2026-07-24
version: 0.3.0
progress: 10%
---

# GOAL-001 · 可复用的 Schema-UI 核心 API 基座

## 概述

将 `allinme.core-api` 建设为**可被后续业务项目复用**的后端核心 API 基座：以项目模板/骨架复用为主，可复用能力逐步沉淀到 `pkg/`；并作为 **Schema-UI 协议的后端实现方**（页面生产 + 业务 API + 鉴权等），固定消费 `magicvr/schema-ui-docs` 协议制品，不另立协议语义。

**交付边界**：本仓**仅后端**。Admin 前端 / Renderer 在其他仓库；本仓不提供生产级 UI 框架 Renderer。

## 成功标准

- [ ] 新项目可按本仓分层与约定，fork/复制骨架后较快落地业务后端（模板复用为主）
- [ ] 可复用横切能力（配置、响应约定、鉴权中间件、Schema-UI page 生产辅助等）有计划地抽到 `pkg/`，且边界清晰
- [x] 固定消费某一不可变 Schema-UI 协议制品（tag/SHA-256）；**不得**私自漂移。**已钉死** `schema-ui-protocol` **2.4.1**（`meta.protocolVersion: "2.4"`；见 D-006）
- [ ] 存在可演示的端到端能力路径：本服务产出覆盖 Admin 入口的 page schema + 配套业务 API；与外部 Renderer 的联调可作为后续增强，非 Root 关门硬条件（MVP 验收口径见子目标）
- [ ] 关键目标文档与 `goal-tree.md` 可追踪阶段进展，重大取舍有决策留痕

## 高层路线图（P-001）

| 阶段 | 名称 | 状态 | 说明 |
|------|------|------|------|
| **R0** | 治理与边界冻结 | **完成（阶段）** | Root/MVP 立项；策略 A；协议钉死 2.4.1（D-006） |
| **R0.5** | 协议演进（外仓） | **完成** | `schema-ui-docs` 已发布 2.4.1；I-006 verified |
| **R1** | MVP Demo Admin | **进行中** | [GOAL-002](../GOAL-002-mvp-demo-admin/00-meta.md) `active`；推进 I-002～I-005 与方案 |
| **R2** | 协议对齐与复用沉淀 | 未开始 | 关闭协议缺口（若有）、强化 conformance/校验、抽取 `pkg/` 可复用包、文档化接入约定 |
| **R3** | 多项目消费就绪 | 未开始 | 第二个消费项目可按文档接入；稳定版本与变更门禁 |

> 同一项目的后续阶段优先更新本路线图并建串行子目标；仅当长期目的/成功边界/战略方向变化时才改写 Root 定义。

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 行内/批量支持边界与交付策略 | R1 策略 | 策略裁决前 | 对照协议 + 用户裁决 | **decided** | — | 用户选 **A 协议演进**；2.4.1 已含批量核心能力 |
| I-006 | required | 新协议制品可固定引用（版本、tag/SHA-256）且覆盖 MVP 批量等能力 | 解除 Root/R1 blocked；重新钉死协议 | 恢复 R1 前 | 协议仓发布 + 本仓核对 | **verified** | — | **2026-07-24**：制品 `2.4.1` / tag `v2.4.1` / artifact SHA-256 `c027fa6c5b4bcb379a2fc90f6447f0e8df0729df5657fcb5d6a382d9ee3fbb18` / contentDigest `sha256:d6852ee6ff12a19b00b4acf0b51e221457fe14691a53ff5ed828876877766efe`。能力覆盖见 D-006。 |
| I-002 | required | 鉴权与会话模型（机制、token、刷新、密码策略等） | R1 实施登录/权限 | R1 实施前 | 决策记录 + 最小实现方案 | open | 已恢复主动收集 | 真实登录/权限；选型未定 |
| I-003 | required | 订单 / 钱包 / 通知 的领域边界与最小数据模型 | R1 业务 API 与 page schema | R1 方案冻结前 | 决策草案或附件 ER/字段表 | open | 已恢复主动收集 | 三域已选定，细节未定 |
| I-004 | non-blocking | 外部 Admin Renderer 仓库与联调方式 | 浏览器可点 E2E | R2 或按需 | 指定仓库/约定联调 | open | MVP 验收不依赖浏览器 Renderer | MVP 以 page schema 覆盖入口为准 |
| I-005 | non-blocking | `pkg/` 首批抽取清单（哪些横切先进库） | R2 复用沉淀 | R2 方案前 | 实施后回顾可抽取点 | open | — | 策略：模板为主，包逐步抽 |

## 父目标

- `null`（Root）

## 子目标

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md) — 阶段 R1

## 备注

- 工作区：`docs/workspace-001-allinme-core-api/`（见同目录 `workspace.md`）。
- 协议权威在 `magicvr/schema-ui-docs`；本仓只消费，不定义协议语义。
- **`status: active`**（2026-07-24）：I-006 关闭；钉死 2.4.1（D-006）；用户确认解除 blocked（D-007）。
