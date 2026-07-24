---
title: MVP 领域模型与 API 摘要
status: active
created: 2026-07-24
updated: 2026-07-25
parent: GOAL-002-mvp-demo-admin
version: 0.2.0
---

# MVP 领域模型与 API（GOAL-002 D-008）

> 权威决策见 `01-decision.md` D-008。本附件为字段与端点清单，实施时可微调字段名但不得静默扩 scope。

## 1. 订单 Order

| 字段 | 类型/说明 |
|------|-----------|
| id | 字符串主键 |
| orderNo | 业务单号，唯一 |
| customerName | 客户名 |
| status | `pending` → `paid` \| `cancelled`；`paid` → `refunded` |
| amountCents | int64，分 |
| currency | 默认 `CNY` |
| remark | 可选 |
| version | 乐观锁 |
| createdAt, updatedAt | RFC3339 |

**HTTP（均需 Bearer，统一前缀 `/v1`；D-018 首切片）**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/orders` | 列表；query：`status`、`q`、`page`、`pageSize`；默认 1/20，pageSize 最大 100；q 匹配 orderNo/customerName |
| GET | `/v1/orders/{id}` | 详情 |
| POST | `/v1/orders` | 创建；默认 pending、CNY、version=1 |
| PUT | `/v1/orders/{id}` | 带 version；仅 pending 可修改 customerName/amountCents/currency/remark |
| POST | `/v1/orders/batch-delete` | body：`{ "ids": [] }`；最多100，拒绝空/重复，事务 all-or-nothing；仅 pending/cancelled 可删 |
| POST | `/v1/orders/{id}/mark-paid` | 带 version；仅 pending |
| POST | `/v1/orders/{id}/cancel` | 带 version；仅 pending |

`viewer` 只读；`admin`/`operator` 可写。所有成功响应为 HTTP 200 + envelope：list `data={list,total}`、单项 `data=order`、批量 `data={deleted:n}`。错误码为 400 `bad_request`、404 `order_not_found`、409 `order_no_conflict`/`version_conflict`/`invalid_state`、500 `internal`，且不得泄露内部错误。

**后续订单范围（不属于 D-018 首切片）**：单项 `DELETE` 与 `refund` action 延后补齐；保留 paid→refunded 领域状态定义。这不缩减 D-008 或 GOAL-002 的总成功标准。

列表响应 envelope（D-015）：`{ "code": 0, "message": "ok", "data": { "list": [], "total": n } }`。  
Schema-UI `responseMapping` 稳定路径：`list: data.list`，`total: data.total`。

## 2. 钱包 Wallet

| 字段 | 说明 |
|------|------|
| id, accountNo, ownerName | 账户 |
| balanceCents, currency | 余额 |
| status | `active` \| `frozen` |
| version, createdAt, updatedAt | |

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/wallets` | 列表筛选 |
| GET | `/wallets/{id}` | 详情 |
| POST | `/wallets` | 创建 |
| PUT | `/wallets/{id}` | 仅元数据（如 ownerName）；**不得**改 balanceCents（D-015） |
| POST | `/wallets/{id}/freeze` | 行内 |
| POST | `/wallets/{id}/unfreeze` | 行内 |
| POST | `/wallets/batch-freeze` | body：`{ "ids": [] }` |

创建时可设初始 `balanceCents`；**无**调账/充值 API。不做支付网关。

## 3. 通知 Notification

| 字段 | 说明 |
|------|------|
| id, title, body | 内容 |
| channel | `inbox` \| `email`（枚举；不真发邮件） |
| status | `draft` → `published` → `archived` |
| createdAt, updatedAt, publishedAt? | |

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/notifications` | 列表 |
| GET | `/notifications/{id}` | 详情 |
| POST | `/notifications` | 创建 |
| PUT | `/notifications/{id}` | 更新 |
| DELETE | `/notifications/{id}` | 删除 |
| POST | `/notifications/{id}/publish` | 行内 draft→published |
| POST | `/notifications/batch-archive` | 批量归档 |

## 4. 鉴权相关（D-007）

| 方法 | 路径 | 鉴权 |
|------|------|------|
| POST | `/auth/login` | 公开 |
| GET | `/auth/me` | 需 Bearer |
| GET | `/admin/menu` | 需 Bearer；按角色过滤菜单 |
| GET | `/admin/pages/{pageId}` | 需 Bearer；下发 page schema |
| GET | `/dashboard/summary` | 需 Bearer；仪表盘聚合 |

## 5. 持久化

- 默认 SQLite；表结构由 repository/sqlite 拥有。
- 领域与 service **不**依赖 `database/sql` 具体方言。
