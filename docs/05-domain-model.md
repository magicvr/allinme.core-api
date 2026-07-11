---
status: target
owner: 后端团队
last_updated: 2026-07-12
applies_to: order operations demo
---

# 订单运营领域模型

## 1. 目标与边界

本文定义本仓目标态的业务语义，不定义 Schema-UI 协议字段。首版覆盖本地账号、订单核心、履约、退款、附件和经营看板；支付网关、物流服务、真实退款通道、库存扣减和多租户不在首版范围内，相关结果由本地业务状态模拟。

演示数据必须足以触发成功、校验失败、权限不足、版本冲突和不可执行状态，不能只提供静态成功响应。

## 2. 核心实体

| 实体 | 关键属性 | 约束 |
|---|---|---|
| User | `id`、`username`、`password_hash`、`role`、`disabled_at` | 用户名唯一；密码只保存强哈希 |
| Session | `id`、`user_id`、`token_id`、`expires_at`、`revoked_at` | JWT 的 `jti` 必须对应未撤销会话 |
| Order | `id`、`customer_name`、`status`、`payment_status`、`currency`、`total_amount`、`version`、时间戳 | 金额使用整数最小货币单位；更新使用乐观锁 |
| OrderItem | `id`、`order_id`、`sku`、`name`、`quantity`、`unit_price` | 数量为正；订单总额由明细计算 |
| Refund | `id`、`order_id`、`amount`、`status`、`reason`、`version`、审核信息 | 累计成功退款不得超过已支付金额 |
| Attachment | `id`、`order_id`、`original_name`、`stored_name`、`media_type`、`size`、`sha256` | 文件名由服务端生成；下载必须重新鉴权 |

数据库时间统一保存 UTC，HTTP 使用 RFC 3339。ID 使用服务端生成、URL 安全且不暴露顺序规模的字符串。SQLite migrations、seed 和 reset 必须可重复执行；reset 仅在显式开发模式开放。

## 3. 订单状态

目标订单状态：

```text
DRAFT -> CONFIRMED -> FULFILLING -> SHIPPED -> COMPLETED
  |          |             |
  +----------+-------------+-> CANCELLED
```

| 操作 | 允许源状态 | 目标状态 | 最低角色 |
|---|---|---|---|
| 创建/编辑 | 新建或 `DRAFT` | `DRAFT` | `operator` |
| 确认 | `DRAFT` | `CONFIRMED` | `operator` |
| 开始履约 | `CONFIRMED` | `FULFILLING` | `operator` |
| 标记发货 | `FULFILLING` | `SHIPPED` | `operator` |
| 完成 | `SHIPPED` | `COMPLETED` | `operator` |
| 取消 | `DRAFT`、`CONFIRMED`、`FULFILLING` | `CANCELLED` | `operator` |

所有写操作必须提交当前 `version`。版本不匹配返回冲突，状态非法返回业务冲突；前端的按钮显隐不构成授权或状态校验。

## 4. 支付与退款

订单支付状态独立于履约状态，首版为 `UNPAID`、`PAID`、`PARTIALLY_REFUNDED`、`REFUNDED`。seed 数据可以直接创建已支付订单，以覆盖退款流程。

退款持久化状态为 `PENDING`、`REJECTED`、`COMPLETED`。`APPROVED` 仅表示审批检查已通过的事务内过渡，不作为可查询或可长期停留的对外状态：

1. `operator` 对已支付且仍有可退金额的订单发起退款；
2. `approver` 或 `admin` 审批待处理退款，申请人不得审批自己的申请；
3. 通过审批后，本地 demo 在同一事务内执行退款、将状态从 `PENDING` 更新为 `COMPLETED`，并更新订单支付状态；任一步失败则整体回滚；
4. 重复请求不得产生重复退款，创建退款必须使用幂等键。

## 5. 角色与权限

| 能力 | viewer | operator | approver | admin |
|---|---:|---:|---:|---:|
| 查看订单、附件和看板 | 是 | 是 | 是 | 是 |
| 创建、编辑和推进履约 | 否 | 是 | 否 | 是 |
| 发起退款 | 否 | 是 | 否 | 是 |
| 审批退款 | 否 | 否 | 是 | 是 |
| 管理本地账号与角色 | 否 | 否 | 否 | 是 |

列表响应中的 `canEdit`、`canAdvance`、`canCancel`、`canRequestRefund` 和 `canApproveRefund` 只用于 Renderer 展示。每个目标端点仍需根据认证主体、资源状态和版本独立授权。

账号与角色管理是 admin 的长期目标能力，不属于首批 HTTP surface；首批 admin 账号由 seed 创建。实施管理 API 前必须先扩展路线图、目标 API 和审计测试。

## 6. 附件生命周期

- 上传接口先写入临时受控目录，校验文件大小、允许类型和摘要后创建未绑定附件记录；不得信任客户端文件名或 `Content-Type`。
- 创建或编辑订单时提交附件 ID，服务端在事务中校验附件所有者、有效期和绑定状态。
- 附件内容位于应用数据目录，SQLite 只保存元数据；文件路径不得由请求直接指定或返回。
- 下载使用附件 ID 并重新执行订单查看权限；删除订单时按明确策略清理或保留附件，首版采用同步清理。
- 未绑定上传应有过期清理机制，目标默认保留 24 小时。

## 7. 看板口径

经营看板从同一 SQLite 数据生成，至少提供订单数、成交额、状态分布和近 7/30 日趋势。成交额按已支付金额减已完成退款统计；所有金额响应同时明确币种，seed 默认使用 `CNY`。

看板只用于演示一致的数据读取，不承诺生产级财务口径或分析性能。

## 8. Seed 验收数据

reset/seed 至少创建四个内置角色账号，以及覆盖每种订单、支付和退款状态的数据。固定账号仅用于本地开发，首次文档或启动输出应提示演示密码，生产模式不得启用 seed 账号或 reset 命令。

目标 HTTP 映射见 [03-http-api-target.md](./03-http-api-target.md)，实施顺序见 [06-implementation-roadmap.md](./06-implementation-roadmap.md)。