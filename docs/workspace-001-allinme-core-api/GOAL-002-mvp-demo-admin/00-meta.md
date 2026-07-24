---
id: GOAL-002-mvp-demo-admin
title: MVP · Demo 完整 Admin（协议驱动）
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-24
version: 0.4.0
progress: 20%
---

# GOAL-002 · MVP · Demo 完整 Admin（协议驱动）

## 概述

交付 **demo 性质但闭环完整** 的 Admin 后端能力：真实登录/会话与用户·角色权限；菜单与路由由 Schema-UI page schema 驱动；**订单、钱包、通知** 三套通用业务域的 CRUD + 列表筛选；仪表盘 grid；行内动作；批量动作。**不包含上传**。

本仓只提供后端（page schema 生产 + 业务 API + 鉴权）。MVP **验收口径**：本服务返回的 page schema **覆盖 Admin 全部入口**。

**实施协议**：`schema-ui-protocol` **2.4.1**（Root D-006）。  
**架构**：遵守 Root D-008（IoC/接口模块化）；**编码实施优先依赖** [GOAL-003](../GOAL-003-modular-ioc-foundation/00-meta.md) 骨架可验收。

## 成功标准

- [ ] 真实登录与会话可用（JWT Bearer）；未认证访问受保护 API 被拒绝
- [ ] 用户 / 角色 / 权限模型真实落地；page 与操作可按权限显隐；后端独立鉴权
- [ ] 菜单与路由由后端下发定义驱动，覆盖 Admin 全部入口页面
- [ ] **订单、钱包、通知** 三域均具备列表+筛选+完整写路径 API 与对应 page schema
- [ ] 至少一页仪表盘（grid）由 schema 定义并可取数
- [ ] 行内动作与批量动作按 2.4.1 核心 capability 实现
- [ ] 不实现上传
- [ ] `meta.protocolVersion: "2.4"`；结构校验对照钉死 2.4.1 制品
- [ ] 默认 **SQLite** 持久化；repository 可替换驱动
- [ ] 本地可启动演示闭环；文档可指出各入口 schema 获取方式
- [ ] 业务代码依赖接口组装，不绕过 GOAL-003 / D-008 边界

## 模块范围（已确认）

| 能力 | MVP |
|------|-----|
| 登录 / 会话 | JWT Bearer，真实校验 |
| 菜单 / 路由 schema 驱动 | 需要 |
| 业务域 | 订单、钱包、通知 |
| 行内 / 批量 | 需要（2.4.1） |
| 上传 | **不做** |
| 仪表盘 grid | 需要 |
| 用户 / 角色权限 | admin / operator / viewer |
| 持久化 | **SQLite 默认**，可换库 |

## 高层路线图（本目标内）

| 阶段 | 名称 | 状态 | 说明 |
|------|------|------|------|
| **M0** | 方案冻结 | **完成** | I-002～I-007 decided；附件已齐 |
| **M1** | 依赖骨架 | **等待 GOAL-003** | 模块/IoC/SQLite 端口就绪后开工业务 |
| **M2** | 鉴权 + RBAC + 菜单 | 未开始 | I-002 / I-004 |
| **M3** | 三域 API + 种子数据 | 未开始 | I-003 |
| **M4** | page schema 生产与校验 | 未开始 | I-005 / I-007 |
| **M5** | 验收对照成功标准 | 未开始 | — |

## 信息就绪与未知项（P-005）

| ID | 级别 | 所需信息 / 问题 | 影响门禁 | 最晚需要阶段 | 验证 / 收集动作 | 状态 | 延期 / 复核 | 证据 / 结论 |
|----|------|-----------------|----------|--------------|-----------------|------|-------------|-------------|
| I-001 | required | 批量策略 | 策略 | 策略前 | 用户裁决 | **decided** | — | A；2.4.1 支持批量 |
| I-008 | required | 新协议制品 | 解除 blocked | 恢复前 | 发布核对 | **verified** | — | 2.4.1 |
| I-002 | required | 鉴权选型与种子用户 | 实施登录 | 实施前 | D-007 | **decided** | — | JWT Bearer；见 D-007 |
| I-003 | required | 三域模型与 API | 方案冻结/实施 | 方案冻结前 | D-008 + 附件 | **decided** | — | [mvp-domain-and-api.md](attachments/mvp-domain-and-api.md) |
| I-004 | required | RBAC 粒度与角色 | 方案冻结（权限） | 方案冻结前 | D-009 | **decided** | — | admin/operator/viewer；菜单+页+操作 |
| I-005 | required | 2.4.1 CRUD 生命周期映射 | 方案冻结（写路径 UI） | 方案冻结前 | D-010 + 附件 | **decided** | — | [protocol-capability-mapping.md](attachments/protocol-capability-mapping.md) |
| I-006 | required | 持久化选型与可换库 | 实施数据层 | 实施前 | D-011 | **decided** | — | 默认 SQLite；接口隔离驱动 |
| I-007 | required | page schema 存储与下发 | 实施 page 生产 | 实施前 | D-012 | **decided** | — | embed YAML/JSON 按 pageId 下发 |
| I-009 | required | GOAL-003 骨架可验收（或书面并行放行） | **M2 起业务编码** | M1→M2 | GOAL-003 成功标准 / 用户放行 | **open** | 责任人：本仓；触发：GOAL-003 done 或用户放行 | 方案可冻结；实施门禁 |

## 父目标

- [GOAL-001-allinme-core-api](../GOAL-001-allinme-core-api/00-meta.md)

## 依赖

- [GOAL-003-modular-ioc-foundation](../GOAL-003-modular-ioc-foundation/00-meta.md) — M1 骨架（非 parent，串行前置）

## 备注

- 2026-07-24：用户确认方案包 **A**；方案冻结完成；实施等 GOAL-003。
