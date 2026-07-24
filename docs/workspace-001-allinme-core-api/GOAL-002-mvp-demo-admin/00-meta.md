---
id: GOAL-002-mvp-demo-admin
title: MVP · Demo 完整 Admin（协议驱动）
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-24
version: 0.3.0
progress: 5%
---

# GOAL-002 · MVP · Demo 完整 Admin（协议驱动）

## 概述

交付 **demo 性质但闭环完整** 的 Admin 后端能力：真实登录/会话与用户·角色权限；菜单与路由由 Schema-UI page schema 驱动；**订单、钱包、通知** 三套通用业务域的 CRUD + 列表筛选；仪表盘 grid；行内动作；批量动作（协议核心能力）。**不包含上传**。

本仓只提供后端（page schema 生产 + 业务 API + 鉴权）。MVP **验收口径**：本服务返回的 page schema **覆盖 Admin 全部入口**（不强制本回合对接外部 Renderer 做浏览器可点验收）。

**实施协议（已钉死 · Root D-006）**：`schema-ui-protocol` **2.4.1**（`meta.protocolVersion: "2.4"`；tag `v2.4.1`；artifact SHA-256 见 I-008 / Root D-006）。  
**历史对照**：2.0.0 仅作缺口记录，**不得**作为实施基线。

## 成功标准

- [ ] 真实登录与会话可用（非纯前端 mock）；未认证访问受保护 API 被拒绝
- [ ] 用户 / 角色 / 权限模型真实落地；page 与操作可按权限显隐或拒绝
- [ ] 菜单与路由由后端下发的 Schema-UI 定义驱动，覆盖 Admin 全部入口页面
- [ ] **订单、钱包、通知** 三域均具备：列表 + 筛选 + 创建/读/更新/删除（或等价完整写路径）的业务 API，以及对应 page schema
- [ ] 至少一页仪表盘（grid）由 schema 定义并可取数
- [ ] 行内动作：在目标协议支持范围内声明并有后端处理
- [ ] 批量动作：按**已钉死协议 2.4.1**核心能力实现（策略 A，见 D-002），不走 Host Extension 冒充协议
- [ ] 不实现上传相关 page/API（明确非目标）
- [ ] 对外声明并校验 `meta.protocolVersion: "2.4"`；页面 schema 通过与 **2.4.1 制品**一致的结构校验
- [ ] 本地可启动演示上述闭环；文档或执行记录可指出各 Admin 入口对应的 schema 获取方式

## 模块范围（已确认）

| 能力 | MVP |
|------|-----|
| 登录 / 会话 | 需要，**真实** |
| 菜单 / 路由 schema 驱动 | 需要 |
| 业务域 CRUD + 列表筛选 | **订单、钱包、通知** |
| 行内动作 | 需要（`actions.row.request`） |
| 批量动作 | 需要（`table.selection` + `actions.batch.request`，2.4.1） |
| 上传 | **不做** |
| 仪表盘 grid | 需要 |
| 用户 / 角色权限 | 需要，**真实** |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 批量动作交付策略 | 策略裁决 | 策略裁决前 | 用户裁决 | **decided** | — | **A 协议演进**；2.4.1 已含批量；见 D-002 / Root D-006 |
| I-008 | required | 含批量（及 MVP 所需相关能力）的 **新协议制品**已发布：版本号、tag/SHA-256、能力清单 | **解除 blocked / 方案冻结与实施** | 恢复实施前 | 协议仓发布后核对 | **verified** | — | **2026-07-24**：`2.4.1` / tag `v2.4.1` / SHA-256 `c027fa6c5b4bcb379a2fc90f6447f0e8df0729df5657fcb5d6a382d9ee3fbb18`；能力覆盖见 Root D-006 与本目标 D-006 |
| I-002 | required | 鉴权技术选型（如 session cookie / JWT / 刷新机制）与密码/种子用户策略 | 实施登录与中间件 | 实施前 | 决策 D-00x | open | **已恢复主动收集** | 只要真实实现，选型未定 |
| I-003 | required | 订单 / 钱包 / 通知最小领域模型（实体、状态机、关键 API） | 方案冻结与实施 | 方案冻结前 | 附件字段表或 decision | open | **已恢复主动收集** | 三域已选定，细节未定 |
| I-004 | required | 角色权限模型（RBAC 粒度：菜单/页面/操作/数据？） | 方案冻结（权限） | 方案冻结前 | 决策 | open | **已恢复主动收集** | 需要真实权限，模型未定 |
| I-005 | required | 目标协议对「编辑回填 / 详情导航 / 完整 CRUD 生命周期」的能力边界与本 MVP 的映射 | 方案冻结（写路径 UI） | 方案冻结前 | 对照 **2.4.1** 规范与 ADR | open | **已恢复主动收集**；制品已 verified，映射工作进行中 | 2.4.1 已具备 page trigger / navigate / form.record.load / recordView / batch 等；待写成 MVP 映射表 |
| I-006 | non-blocking | 持久化选型（内存 / SQLite / Postgres 等） | 实施数据层 | 实施前 | 决策 | open | — | — |
| I-007 | non-blocking | 页面 schema 的存储与下发方式（静态 YAML/JSON / DB / 代码生成） | 实施 page 生产 | 实施前 | 决策 | open | — | — |

## 父目标

- [GOAL-001-allinme-core-api](../GOAL-001-allinme-core-api/00-meta.md)

## 备注

- 对应 Root 路线图 **R1**。
- 上传明确排除；若后续需要，另开决策/阶段，不静默扩 scope。
- **`status: active`**（2026-07-24）：I-008 关闭；跟随 Root D-006/D-007 解除 blocked。下一步：I-002～I-005 与方案冻结。
