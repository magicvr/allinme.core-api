---
id: GOAL-002-mvp-demo-admin
title: MVP · Demo 完整 Admin（协议驱动）
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-25
version: 0.16.0
progress: 60%
---

# GOAL-002 · MVP · Demo 完整 Admin（协议驱动）

## 概述

交付 **demo 性质但闭环完整** 的 Admin 后端能力：真实登录/会话与用户·角色权限；菜单与路由由 Schema-UI page schema 驱动；**订单、钱包、通知** 三套通用业务域的 CRUD + 列表筛选；仪表盘 grid；行内动作；批量动作。**不包含上传**。

本仓只提供后端（page schema 生产 + 业务 API + 鉴权）。MVP **验收口径**：本服务返回的 page schema **覆盖 Admin 全部入口**。

**实施协议**：`schema-ui-protocol` **2.4.1**（Root D-006）。  
**架构**：遵守 Root D-008；依赖 [GOAL-003](../GOAL-003-modular-ioc-foundation/00-meta.md)（已 done）。

## 成功标准

- [x] 真实登录与会话可用（JWT Bearer）；未认证访问受保护 API 被拒绝
- [x] 用户 / 角色 / 权限模型真实落地；菜单按角色过滤；后端独立鉴权（page 写路径 permissions 待 M4）
- [ ] 菜单与路由由后端下发定义驱动，覆盖 Admin 全部入口页面（菜单 API 已有；page schema 待 M4）
- [ ] **订单、钱包、通知** 三域均具备列表+筛选+完整写路径 API 与对应 page schema
- [ ] 至少一页仪表盘（grid）由 schema 定义并可取数
- [ ] 行内动作与批量动作按 2.4.1 核心 capability 实现
- [x] 不实现上传
- [ ] `meta.protocolVersion: "2.4"`；结构校验对照钉死 2.4.1 制品（消费路径见 **I-010**）
- [x] 默认 **SQLite** 持久化；repository 可替换驱动
- [ ] 本地可启动演示闭环；文档可指出各入口 schema 获取方式
- [x] 业务代码依赖接口组装，不绕过 GOAL-003 / D-008 边界

## 模块范围（已确认）

| 能力 | MVP |
|------|-----|
| 登录 / 会话 | JWT Bearer，真实校验 |
| 菜单 / 路由 schema 驱动 | 菜单 API 已落地；page 待 M4 |
| 业务域 | 订单、钱包、通知 |
| 行内 / 批量 | 需要（2.4.1） |
| 上传 | **不做** |
| 仪表盘 grid | 需要 |
| 用户 / 角色权限 | admin / operator / viewer |
| 持久化 | **SQLite 默认**，可换库 |

## 高层路线图（渐进子目标）

| 阶段 | 名称 | 状态 | 对应目标 / 说明 |
|------|------|------|-----------------|
| **M0** | 方案冻结 | **完成** | I-002～I-007 decided；父目标保留权威决策 |
| **M1** | 门禁：I-009 | **完成** | GOAL-003 done |
| **M2** | 鉴权 + RBAC + 菜单 | **完成** | [GOAL-004](../GOAL-004-auth-rbac-menu/00-meta.md)（依据既有事实补录，done） |
| **M3a** | 订单 API 首切片 | **完成** | [GOAL-005](../GOAL-005-order-api-first-slice/00-meta.md)（依据既有事实补录，done） |
| **M3b** | 钱包 API + 种子数据 | **完成** | [GOAL-006](../GOAL-006-wallet-api/00-meta.md)（done / 100%；A-002/A-003 close-out pass，A-004 响应维持 done） |
| **M3c** | 通知 API + 种子数据 | **进行中** | [GOAL-007](../GOAL-007-notification-api/00-meta.md)（active / 20%；I-001 verified；N1 完成，N2 待） |
| **M3d** | 订单 DELETE / refund 补齐 | 未开始 | 进入阶段时创建子目标；不把首切片误记为全量完成 |
| **M4** | page schema、仪表盘与协议校验 | 未开始 | 进入阶段时创建子目标；依赖 **I-010** verified |
| **M5** | MVP 集成验收 | 未开始 | 最后创建验收子目标，对照本目标全部成功标准 |

> 采用渐进拆分：已有独立证据的完成切片补录为 `done` 子目标；当前阶段创建 `active` 子目标；未来阶段仅保留路线图，进入阶段时再立项，避免提前生成空目标。

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 批量策略 | 策略 | 策略前 | 用户裁决 | **decided** | — | A；2.4.1 支持批量 |
| I-008 | required | 新协议制品 | 解除 blocked | 恢复前 | 发布核对 | **verified** | — | 2.4.1 |
| I-002 | required | 鉴权选型与种子用户 | 实施登录 | 实施前 | D-007 | **decided** | — | JWT Bearer；已实施 |
| I-003 | required | 三域模型与 API | 方案冻结/实施 | 方案冻结前 | D-008 + 附件 | **decided** | — | 附件；M3 实施中 |
| I-004 | required | RBAC 粒度与角色 | 方案冻结（权限） | 方案冻结前 | D-009 | **decided** | — | admin/operator/viewer；菜单已过滤 |
| I-005 | required | 2.4.1 CRUD 生命周期映射 | 方案冻结（写路径 UI） | 方案冻结前 | D-010 + 附件 | **decided** | — | 映射附件 |
| I-006 | required | 持久化选型与可换库 | 实施数据层 | 实施前 | D-011 | **decided** | — | SQLite |
| I-007 | required | page schema 存储与下发 | 实施 page 生产 | 实施前 | D-012 | **decided** | — | embed |
| I-009 | required | GOAL-003 骨架可验收 | M2 业务编码 | M1→M2 | handover + done | **verified** | — | 2026-07-24 |
| I-010 | required | 2.4.1 制品本地落仓与校验 | M4 校验宣称 | **M4 前** | D-016 + 实施 | **open** | M4 前关闭 | 候选 A/B |

## 父目标

- [GOAL-001-allinme-core-api](../GOAL-001-allinme-core-api/00-meta.md)

## 依赖

- [GOAL-003-modular-ioc-foundation](../GOAL-003-modular-ioc-foundation/00-meta.md) — **done**

## 子目标

- [GOAL-004-auth-rbac-menu](../GOAL-004-auth-rbac-menu/00-meta.md) — M2，**done**（补录）
- [GOAL-005-order-api-first-slice](../GOAL-005-order-api-first-slice/00-meta.md) — M3a，**done**（补录）
- [GOAL-006-wallet-api](../GOAL-006-wallet-api/00-meta.md) — M3b，**done**
- [GOAL-007-notification-api](../GOAL-007-notification-api/00-meta.md) — M3c，**active**

## 备注

- 2026-07-24：M2 落地 JWT + RBAC 菜单。
- 2026-07-25：M3 订单首切片完成（API、SQLite seed、测试）。
- 2026-07-25：采用渐进子目标拆分；补录 GOAL-004/005，创建 GOAL-006；父目标 progress 暂保持 50%，不因治理重排机械重算。
- 2026-07-25：GOAL-006 D-003 + A-001 关闭钱包 I-001，W0 完成。
- 2026-07-25：GOAL-006 W1～W3 实施完成。
- 2026-07-25：GOAL-006 A-002 close-out pass，钱包 API 子目标 `done` / 100%；父目标因真实产品增量调整为 60%。
- 2026-07-25：GOAL-006 A-003 independent close-out pass；A-004 响应维持 done。进入 M3c：创建 GOAL-007，D-003 冻结通知首切片契约；父目标 progress 仍 60%（尚无通知产品代码）。
- 2026-07-25：GOAL-007 A-001 design-plan pass，I-001 verified；N1 domain/port/service 完成（20%）；父目标 progress 仍 60%（N2/N3 未交付 HTTP）。
- I-010 仍 open 且仅阻断 M4 校验宣称。
