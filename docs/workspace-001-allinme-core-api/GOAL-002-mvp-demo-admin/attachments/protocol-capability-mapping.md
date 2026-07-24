---
title: Schema-UI 2.4.1 capability → MVP 页面映射
status: active
created: 2026-07-24
updated: 2026-07-24
parent: GOAL-002-mvp-demo-admin
version: 0.1.0
---

# 协议能力映射（GOAL-002 D-010 / I-005）

钉死制品：`schema-ui-protocol` **2.4.1**，`meta.protocolVersion: "2.4"`。

## 全局

| 项 | 约定 |
|----|------|
| protocolVersion | 所有 Admin page schema 声明 `"2.4"` |
| requiredCapabilities | 按页声明实际用到的预定义键 |
| 结构校验 | 对照钉死 2.4.1 制品内 schemas |
| 上传 | **不做** |
| 跨页全选 / 批量部分成功 / 行内编辑 / 导入导出 | **不做**（协议后续轨） |

## 页面清单与 capability

| pageId（建议） | 形态 | requiredCapabilities | 后端依赖 |
|----------------|------|----------------------|----------|
| `dashboard` | grid + DataRef | （基础列表/展示即可） | `GET /v1/dashboard/summary` |
| `order_list` | table + search + toolbar 新建 + 行 navigate + 行 request + selection 批量 | `actions.page.trigger`, `actions.row.navigate`, `actions.row.request`, `table.selection`, `actions.batch.request` | orders list/batch/row |
| `order_create` | form submit | `actions.page.trigger`（返回） | POST order |
| `order_edit` | form + recordSource | `form.record.load` | GET + PUT |
| `order_detail` | recordView | `record.view.load` | GET |
| `wallet_list` / `_create` / `_edit` / `_detail` | 同构 | 同上模式 | wallets API |
| `notification_list` / `_create` / `_edit` / `_detail` | 同构 | 同上模式 | notifications API |

登录页：**不**走 page schema（或极简非协议 HTML/前端自有页）；API 为 `POST /v1/auth/login`。

菜单：后端 `GET /v1/admin/menu` 返回树；项与 pageId/route 对应；按角色过滤。

## permissions 与角色（D-009）

page 节点使用协议 `permissions.view|edit|delete` 表达式，仅 `$context.user.*`，例如：

- viewer：多数写入口 `edit`/`delete` 为 false
- operator / admin：业务写为 true
- 后端 handler **独立**鉴权，不信任前端

可选使用 `permissions.inheritance`（2.3+）做容器级联；MVP 可用本地 permissions，不强制。

## Renderer 上下文

后端 `/auth/me` 与登录响应提供：

```json
{ "id": "...", "name": "...", "roles": ["admin"] }
```

映射到 Renderer `$context.user` 最小字段集（协议规定）。
