---
title: MVP 领域模型与 API 摘要
status: active
created: 2026-07-24
updated: 2026-07-24
parent: GOAL-002-mvp-demo-admin
version: 0.1.0
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

**HTTP（均需鉴权，前缀建议 `/v1`）**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/orders` | 列表；query：status、q、page、pageSize |
| GET | `/orders/{id}` | 详情 |
| POST | `/orders` | 创建 |
| PUT | `/orders/{id}` | 更新（带 version） |
| DELETE | `/orders/{id}` | 删除 |
| POST | `/orders/batch-delete` | body：`{ "ids": [] }` |
| POST | `/orders/{id}/mark-paid` | 行内：仅 pending |
| POST | `/orders/{id}/cancel` | 行内：仅 pending |

列表响应形状对齐 Schema-UI mapping：`{ "data": { "list": [], "total": n } }`（具体 envelope 与现有 `internal/response` 统一时再定，但 mapping 路径需稳定）。

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
| PUT | `/wallets/{id}` | 更新元数据（非任意改余额可另定 adjust） |
| POST | `/wallets/{id}/freeze` | 行内 |
| POST | `/wallets/{id}/unfreeze` | 行内 |
| POST | `/wallets/batch-freeze` | body：`{ "ids": [] }` |

不做支付网关、充值渠道对接。

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
