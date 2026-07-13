---
status: target
owner: 后端团队
last_updated: 2026-07-13
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
| Refund | `id`、`order_id`、`amount`、`status`、`reason`、`version`、`requested_by`、`decided_by`、时间戳 | `PENDING + COMPLETED` 共同占用额度且不得超过订单总额；申请人与审批人必须隔离 |
| Attachment | `id`、`order_id`、`original_name`、`stored_name`、`media_type`、`size`、`sha256` | 文件名由服务端生成；下载必须重新鉴权 |

数据库时间统一保存 UTC，HTTP 使用 RFC 3339。ID 使用服务端生成、URL 安全且不暴露顺序规模的字符串。SQLite migrations、seed 和 reset 必须可重复执行；reset 仅在显式开发模式开放。

阶段三订单金额使用 `int64` 最小货币单位，币种只接受 `CNY`。订单必须包含 `1..100` 个明细，每项 `quantity=1..10_000`、`unit_price=1..9_999_999_999`，因此 `total_amount` 合法范围为 `1..9_999_999_999`；service 使用 checked multiplication/addition 重算总额，数据库使用同范围 CHECK。阶段三不支持赠品或零金额订单；未来如需支持，必须先修改本领域模型再调整数据库与 HTTP 契约。

## 3. 订单状态

目标订单状态：

```text
DRAFT -> CONFIRMED -> FULFILLING -> SHIPPED -> COMPLETED
  |          |             |
  +----------+-------------+-> CANCELLED
```

| 操作 | 允许源状态 | 目标状态 | 允许角色 |
|---|---|---|---|
| 创建/编辑 | 新建或 `DRAFT` | `DRAFT` | `operator`、`admin` |
| 确认 | `DRAFT` | `CONFIRMED` | `operator`、`admin` |
| 开始履约 | `CONFIRMED` | `FULFILLING` | `operator`、`admin` |
| 标记发货 | `FULFILLING` | `SHIPPED` | `operator`、`admin` |
| 完成 | `SHIPPED` | `COMPLETED` | `operator`、`admin` |
| 取消 | `DRAFT`、`CONFIRMED`、`FULFILLING` | `CANCELLED` | `operator`、`admin` |

订单创建不接受客户端 `version`，由服务端将初始版本设为 `1`；编辑和状态 Action 必须提交当前 `version`。版本不匹配返回冲突，状态非法返回业务冲突；前端的按钮显隐不构成授权或状态校验。

订单创建后的 `version` 为 `1`，每次成功编辑或状态 Action 原子增加 `1`；`created_at` 创建后不可变，成功写操作更新 `updated_at`，失败和幂等重放不修改时间。冲突按资源存在性、version、源状态顺序分类：不存在、版本冲突、状态冲突分别保持可区分。

`admin` 是首批业务能力中 `operator` 与 `approver` 的显式超集，但角色本身不存在可排序层级。实现必须复用 `internal/auth.Role`、`auth.Principal` 和 `auth.RoleAllowed` 的显式 allowlist，不在订单模块复制角色常量或通过字符串比较推导权限。

## 4. 支付与退款

订单支付状态独立于履约状态，首版为 `UNPAID`、`PAID`、`PARTIALLY_REFUNDED`、`REFUNDED`。退款资格只看支付状态和可退额度，不看履约状态；因此包括 `CANCELLED + PAID` 在内的任意履约状态，只要支付状态为 `PAID` 或 `PARTIALLY_REFUNDED` 且仍有额度，均可申请退款。seed 数据可以直接创建这些组合以覆盖退款流程。

退款持久化状态为 `PENDING`、`REJECTED`、`COMPLETED`。`APPROVED` 仅表示审批检查已通过的事务内过渡，不作为可查询或可长期停留的对外状态：

1. `operator` 或 `admin` 对已支付且仍有可退金额的订单发起退款；`availableRefundAmount = totalAmount - pendingAmount - completedAmount`，同一订单允许存在多笔 `PENDING`，但合计占用不得超过订单总额；
2. `approver` 或 `admin` 审批待处理退款，申请人不得审批自己的申请；
3. 通过审批后，本地 demo 在同一事务内执行退款、将状态从 `PENDING` 更新为 `COMPLETED`，并更新订单支付状态；任一步失败则整体回滚；
4. 拒绝只把退款改为 `REJECTED`，不修改订单版本或支付状态；该笔 `PENDING` 占用随之释放，可退金额立即回升；
5. 重复请求不得产生重复退款，创建退款必须使用幂等键。

退款 ID 固定为 `rfd_` 加 32 位小写十六进制；金额使用 `int64` 分，范围为 `1..9_999_999_999`，币种继承订单且首版仅为 `CNY`。reason 必须是有效 UTF-8、不得包含 NUL，经 Go `strings.TrimSpace` 后为 `1..500` UTF-8 bytes；外围空白不保存，内部换行允许并保持原样。

退款创建版本为 `1`，approve 或 reject 成功后版本为 `2`，终态不可再次转换。审计字段包括 `requestedBy`、`decidedBy`、`createdAt`、`updatedAt`、`decidedAt`：创建时 `createdAt == updatedAt` 且决定字段为空；approve/reject 成功时 `updatedAt == decidedAt` 并记录审批主体；失败、冲突和幂等重放不更新时间。申请人 user ID 与审批主体 user ID 相同时必须拒绝，即使主体角色为 `admin`。

已完成退款为 0 时，已支付订单保持 `PAID`；`0 < completedAmount < totalAmount` 时为 `PARTIALLY_REFUNDED`；`completedAmount == totalAmount` 时为 `REFUNDED`。退款审批成功原子增加订单版本并更新时间；申请和拒绝不修改订单版本。阶段四启用退款后，DRAFT 编辑还必须在同一事务校验新总额不小于 `PENDING + COMPLETED`，拒绝损坏退款聚合，并按新总额重新推导支付状态；合法 edit 可以改变看板的当前 gross 统计。

阶段四所有非订单 create 幂等重放的 Order DTO 都返回整数分 `availableRefundAmount`。`UNPAID`、`REFUNDED` 固定为 `0`；`PAID`、`PARTIALLY_REFUNDED` 按 `totalAmount - pendingAmount - completedAmount` 计算。`canRequestRefund` 仅在当前主体为 `operator` 或 `admin`、支付状态可退款且 `availableRefundAmount > 0` 时为 `true`；`canApproveRefund` 在 Order DTO 中始终为 `false`，具体审批能力只由 Refund DTO 表达。列表必须批量加载当前页退款占用，列表、详情、首次 create、edit 和履约 Action 使用同一规则。阶段三订单 create snapshot v1 保持不变；其幂等重放不得读取当前退款，新增字段固定映射为 `availableRefundAmount=0`、`canRequestRefund=false`。

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
- 下载使用附件 ID 并重新执行订单查看权限。阶段五不提供订单删除 HTTP，只提供受信任 admin maintenance 编排可调用的内部 `ORDER_DELETE` cleanup 原语：仅允许当前仍为 `DRAFT`、expected version 匹配且不存在任何退款或退款幂等历史的订单；按“资源存在性 → version → 状态 → 退款历史”分类失败，并在取得附件 ownership 前拒绝。
- `ORDER_DELETE` 成功后永久保留历史 order-create v1/v2 idempotency snapshot，不以外键关联当前订单。相同主体/method/route/key 的重试继续返回首次冻结响应，即使响应中的订单已被内部清理；不得删除该记录后以同 key 创建第二个订单。该历史响应不表示订单仍存在，后续订单读取仍返回 not found。
- 未绑定上传应有过期清理机制，目标默认保留 24 小时。

## 7. 看板口径

经营看板从同一 SQLite 数据生成，至少提供订单数、状态分布和近 7/30 日趋势。金额明确拆为 `grossAmount`、`completedRefundAmount` 和 `netAmount = grossAmount - completedRefundAmount`：`orderCount` 统计全部订单；`grossAmount` 汇总支付状态为 `PAID`、`PARTIALLY_REFUNDED`、`REFUNDED` 的订单原始总额，履约状态不影响纳入；`completedRefundAmount` 只汇总 `COMPLETED` 退款。所有金额响应同时明确币种，seed 默认使用 `CNY`。summary 的 `netAmount` 不得为负，负值表示数据损坏。

趋势统一使用注入 clock 和 UTC 日历日。`days=7|30` 的窗口包含 clock 所在当天及之前 `days-1` 天，返回 UTC `YYYY-MM-DD`，无数据日期必须补零。每日 `orderCount` 统计当日创建的全部订单并按订单 `createdAt` 归属；每日 `grossAmount` 只统计当前支付状态属于上述纳入集合的订单，并按订单 `createdAt` 归属；每日 `completedRefundAmount` 按已完成退款的 `decidedAt` 归属；每日 `netAmount` 为后两项之差。由于订单可能在窗口外创建而退款在窗口内完成，单日 trend 的 `netAmount` 允许为负。

看板只用于演示一致的数据读取，不承诺生产级财务口径或分析性能。

## 8. Seed 验收数据

development reset/seed 使用 `DEMO_ACCOUNT_PASSWORD` 创建 `viewer`、`operator`、`approver`、`admin` 四个同名角色账号；文档只提示环境变量配置方式，运行时不输出密码。production 不启用演示账号或 reset，通过仅限空 users 表的一次性 `bootstrap-admin` 创建首个 admin。订单、支付和退款状态 seed 仍属于阶段三及后续阶段。

目标 HTTP 映射见 [03-http-api-target.md](./03-http-api-target.md)，实施顺序见 [06-implementation-roadmap.md](./06-implementation-roadmap.md)。
