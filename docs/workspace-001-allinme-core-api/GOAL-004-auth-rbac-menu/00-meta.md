---
id: GOAL-004-auth-rbac-menu
title: 鉴权、RBAC 与菜单闭环
status: done
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.1.0
progress: 100%
---

# GOAL-004 · 鉴权、RBAC 与菜单闭环

## 概述

依据 GOAL-002 已发生的 M2 实施事实补录本子目标：交付 JWT Bearer 登录与会话、三角色 RBAC、按角色过滤的菜单，以及受保护 API 的后端鉴权。

> 本目标于 2026-07-25 补录治理结构；实施事实发生于 2026-07-24。补录不改变原始事实日期，也不表示当日重新实施。

## 成功标准

- [x] `POST /v1/auth/login` 可使用真实 bcrypt 密码校验并返回 JWT
- [x] `GET /v1/auth/me` 与 `GET /v1/admin/menu` 受 Bearer 保护
- [x] admin / operator / viewer 三角色及菜单过滤规则落地
- [x] 除 health / ready / login 外的既有受保护 API 拒绝未认证请求
- [x] service 仅依赖 port，具体安全与 SQLite 实现在 composition root 组装
- [x] auth 集成路径与无 token 401 已由测试覆盖

## 信息就绪与未知项

本子目标所需信息已在父目标中关闭：GOAL-002 I-002、I-004、I-006、I-009；权威决策为 D-007、D-009、D-011、D-013～D-014。补录时无新增 required 信息项。

## 父目标

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md)

## 备注

- 父目标路线图阶段：M2。
- 历史实现与验证证据保留在父目标 `02-execution.md`；本目标建立可独立追踪的交付边界。
