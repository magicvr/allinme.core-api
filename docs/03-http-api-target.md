---
status: target
principles_stage: baseline
endpoints_stage: draft
owner: 后端团队
last_updated: 2026-07-12
applies_to: order operations demo HTTP API target
---

# 目标 HTTP API

> 本文不描述当前可调用接口。通用原则是目标实现基线；所有具体 path、字段、错误码和页面 ID 均为 `draft target`，应在对应路线阶段实现时由测试和 Schema-UI 页面用例收敛，未收敛前不构成兼容性承诺。当前接口见 [当前 HTTP API](./03-http-api.md)。

## 1. 设计基线

以下原则已确定；若实现需要改变，应更新领域模型或新增 ADR：

- 业务 API 使用版本化基础路径，JSON 为默认媒体类型，上传除外。
- 除登录、liveness 和 readiness 外，业务请求必须认证；授权由后端按角色、资源和状态执行。
- JSON 输入限制大小、拒绝未知字段和多余值；multipart 使用独立限制。
- 时间使用 UTC RFC 3339；金额使用整数最小货币单位并携带 ISO 4217 币种。
- 列表使用服务端分页与 allowlist 排序，不把客户端字段直接拼接到 SQL。
- 创建型关键操作支持幂等；资源更新和状态 Action 使用 `version` 乐观锁。
- 错误响应具有稳定机器码、安全消息、request ID 和可选字段级详情。

目标列表 envelope 与错误字段的具体名称仍是草案，应在首个业务 endpoint 实现时冻结。

## 2. 运行状态（draft target）

| Method | Path | 目标语义 |
|---|---|---|
| `GET` | `/healthz` | liveness；进程能响应即成功，不检查依赖 |
| `GET` | `/readyz` | readiness；检查 migrations、SQLite 和页面配置是否已就绪 |

`/readyz` 在阶段一实现。页面模块尚未启用时不应伪造页面检查成功，具体分阶段策略在实现时由测试固定。

## 3. 认证（draft target）

| Method | Path | 权限 | 行为 |
|---|---|---|---|
| `POST` | `/api/v1/auth/login` | public | 校验本地账号，创建 session 并签发短期 JWT |
| `GET` | `/api/v1/auth/me` | authenticated | 返回当前身份与 token 到期时间 |
| `POST` | `/api/v1/auth/logout` | authenticated | 撤销当前 JWT 对应 session |

错误密码、未知用户名和禁用账号使用相同认证失败响应。角色集合与能力以 [领域模型](./05-domain-model.md) 为唯一事实源。

账号管理属于 admin 目标能力，但不在首批 HTTP surface；需要实施时先在路线图增加独立阶段，再定义 endpoint 和测试。

## 4. 页面配置（draft target）

| Method | Path | 权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/pages` | authenticated | 返回当前角色可访问页面摘要 |
| `GET` | `/api/v1/pages/{pageId}` | page-specific | 返回启动时已校验的 Schema-UI JSON 页面 |

目标页面暂定为经营看板、订单列表、订单编辑、退款队列和订单附件。最终 page ID 随阶段六页面文件一起冻结。服务端使用 allowlist，不把请求 path 映射为文件路径；响应目标支持 ETag。

## 5. 订单（draft target）

| Method | Path | 最低权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/orders` | viewer | 搜索、筛选、排序和分页 |
| `POST` | `/api/v1/orders` | operator | 创建草稿订单，支持幂等 |
| `GET` | `/api/v1/orders/{orderId}` | viewer | 返回订单详情、附件摘要和可执行能力 |
| `PATCH` | `/api/v1/orders/{orderId}` | operator | 编辑草稿并校验 `version` |
| `POST` | `/api/v1/orders/{orderId}/confirm` | operator | 确认订单 |
| `POST` | `/api/v1/orders/{orderId}/fulfill` | operator | 开始履约 |
| `POST` | `/api/v1/orders/{orderId}/ship` | operator | 标记发货 |
| `POST` | `/api/v1/orders/{orderId}/complete` | operator | 完成订单 |
| `POST` | `/api/v1/orders/{orderId}/cancel` | operator | 取消订单 |

状态转换、角色权限、金额计算和业务不变量只在 [领域模型](./05-domain-model.md) 维护。列表目标支持关键词、订单/支付状态、创建日期、分页和 allowlist 排序；具体 query 名称与 envelope 在阶段三实现时冻结。

列表可返回按当前主体与资源计算的 `canXxx` 展示字段，但 Action 必须重新鉴权和校验。创建/编辑由服务端根据订单项计算总额，不接受客户端总额作为事实。

## 6. 退款（draft target）

| Method | Path | 最低权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/refunds` | approver | 查询待审批及历史退款 |
| `POST` | `/api/v1/orders/{orderId}/refunds` | operator | 幂等发起退款 |
| `POST` | `/api/v1/refunds/{refundId}/approve` | approver | 审批并执行本地退款 |
| `POST` | `/api/v1/refunds/{refundId}/reject` | approver | 拒绝退款 |

退款状态和审批规则以 [领域模型](./05-domain-model.md) 为唯一事实源。请求字段、响应 envelope 与错误码在阶段四实现时冻结。

## 7. 附件（draft target）

| Method | Path | 最低权限 | 行为 |
|---|---|---|---|
| `POST` | `/api/v1/attachments` | operator | 单文件 multipart 上传，返回未绑定附件 ID |
| `GET` | `/api/v1/attachments/{attachmentId}` | viewer | 鉴权下载已绑定附件 |
| `DELETE` | `/api/v1/attachments/{attachmentId}` | operator | 删除本人创建且未绑定的附件 |

首版目标允许 PDF、PNG 和 JPEG，单文件目标上限 10 MiB；允许类型和上限在阶段五威胁测试完成后冻结。服务端检测内容、生成文件名并计算摘要，不返回本地路径或公开静态 URL。

## 8. 看板（draft target）

| Method | Path | 权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/dashboard/summary` | viewer | 订单数、成交额、退款额和币种 |
| `GET` | `/api/v1/dashboard/order-status` | viewer | 订单状态分布 |
| `GET` | `/api/v1/dashboard/trend` | viewer | 7/30 日订单与净成交趋势 |

统计业务口径只在 [领域模型](./05-domain-model.md) 维护。响应字段随阶段四页面 datasource 用例冻结。

## 9. Schema-UI 映射

| 页面能力 | 目标 API 族 |
|---|---|
| 经营看板 | `/api/v1/dashboard/*` |
| 搜索表格 | `GET /api/v1/orders` |
| 联动表单提交 | 订单创建/编辑 |
| 行级 Action | 订单履约、取消与退款审批 |
| UploadAction | 附件上传与订单绑定 |

页面只使用固定协议已有的 datasource、Action、Reaction 和 mapping 能力。页面 YAML 实施时必须通过 [Schema-UI 固定版本](./02-schema-ui-integration.md) 校验。

## 10. 草案收敛条件

每个 endpoint 从 `draft target` 收敛为目标契约前，至少具备：

- handler、业务用例和 repository 的请求/响应类型；
- 正常、输入错误、认证、权限、状态、并发和内部错误测试；
- 对应 Schema-UI datasource 或 Action 用例；
- [验证规则](./04-validation.md)中对应门禁已启用；
- 当前 API 文档、场景和 CHANGELOG 已同步。

实现完成后，把该 endpoint 移入 [当前 HTTP API](./03-http-api.md)，目标文档只保留尚未实现的草案。