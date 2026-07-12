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

事实源按关注点拆分：状态机、金额、角色和业务不变量只在[领域模型](./05-domain-model.md)维护；本文件维护目标 endpoint 范围与跨阶段边界；订单 API 处于阶段三 active 期间，query、DTO、错误、CORS、幂等和成功状态只在[阶段三活跃计划](./audit/0003-2026-07-12-plan.md#3-已冻结的-http-与幂等契约)维护；实现完成后迁入[当前 HTTP API](./03-http-api.md)。其他文档只链接这些来源，不复制完整规则。

## 2. 已实现运行状态

`GET /healthz`、`GET /readyz`、request ID、运行错误 envelope 和 recovery 契约已在阶段一冻结，见 [当前 HTTP API](./03-http-api.md)。页面模块尚未启用，阶段六在现有 readiness 基础上扩展页面检查。

## 3. 页面配置（draft target）

| Method | Path | 权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/pages` | authenticated | 返回当前角色可访问页面摘要 |
| `GET` | `/api/v1/pages/{pageId}` | page-specific | 返回启动时已校验的 Schema-UI JSON 页面 |

目标页面暂定为经营看板、订单列表、订单编辑、退款队列和订单附件。最终 page ID 随阶段六页面文件一起冻结。服务端使用 allowlist，不把请求 path 映射为文件路径；响应目标支持 ETag。

## 4. 订单（draft target）

| Method | Path | 允许角色 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/orders` | authenticated | 搜索、筛选、排序和分页 |
| `POST` | `/api/v1/orders` | operator、admin | 创建草稿订单，支持幂等 |
| `GET` | `/api/v1/orders/{orderId}` | authenticated | 返回订单详情和可执行能力；阶段三不含附件摘要 |
| `PATCH` | `/api/v1/orders/{orderId}` | operator、admin | 编辑草稿并校验 `version` |
| `POST` | `/api/v1/orders/{orderId}/confirm` | operator、admin | 确认订单 |
| `POST` | `/api/v1/orders/{orderId}/fulfill` | operator、admin | 开始履约 |
| `POST` | `/api/v1/orders/{orderId}/ship` | operator、admin | 标记发货 |
| `POST` | `/api/v1/orders/{orderId}/complete` | operator、admin | 完成订单 |
| `POST` | `/api/v1/orders/{orderId}/cancel` | operator、admin | 取消订单 |

状态转换、角色权限、金额计算和业务不变量只在[领域模型](./05-domain-model.md)维护。订单 query、分页、排序、DTO、错误和 CORS 在阶段三 active 期间只由[阶段三活跃计划](./audit/0003-2026-07-12-plan.md#3-已冻结的-http-与幂等契约)维护；实现完成后随 endpoint 一起迁入当前 API，本文件不维护第二份字段清单。

列表可返回按当前主体与资源计算的 `canXxx` 展示字段，但 Action 必须重新鉴权和校验。创建/编辑由服务端根据订单项计算总额，不接受客户端总额作为事实。

附件摘要随阶段五附件生命周期一起新增并冻结；阶段三订单 DTO 不预留 `attachments` 字段。订单 endpoint 实现时以阶段三活跃计划中的冻结 query、DTO、CORS、幂等和错误契约为准，再迁入当前 API。

## 5. 退款（draft target）

| Method | Path | 最低权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/refunds` | approver | 查询待审批及历史退款 |
| `POST` | `/api/v1/orders/{orderId}/refunds` | operator | 幂等发起退款；幂等作用域包含 method 与 route/operation，可复用订单创建使用过的 key |
| `POST` | `/api/v1/refunds/{refundId}/approve` | approver | 审批并执行本地退款 |
| `POST` | `/api/v1/refunds/{refundId}/reject` | approver | 拒绝退款 |

退款状态和审批规则以 [领域模型](./05-domain-model.md) 为唯一事实源。请求字段、响应 envelope 与错误码在阶段四实现时冻结。

## 6. 附件（draft target）

| Method | Path | 最低权限 | 行为 |
|---|---|---|---|
| `POST` | `/api/v1/attachments` | operator | 单文件 multipart 上传，返回未绑定附件 ID |
| `GET` | `/api/v1/attachments/{attachmentId}` | viewer | 鉴权下载已绑定附件 |
| `DELETE` | `/api/v1/attachments/{attachmentId}` | operator | 删除本人创建且未绑定的附件 |

首版目标允许 PDF、PNG 和 JPEG，单文件目标上限 10 MiB；允许类型和上限在阶段五威胁测试完成后冻结。服务端检测内容、生成文件名并计算摘要，不返回本地路径或公开静态 URL。

## 7. 看板（draft target）

| Method | Path | 权限 | 行为 |
|---|---|---|---|
| `GET` | `/api/v1/dashboard/summary` | viewer | 订单数、成交额、退款额和币种 |
| `GET` | `/api/v1/dashboard/order-status` | viewer | 订单状态分布 |
| `GET` | `/api/v1/dashboard/trend` | viewer | 7/30 日订单与净成交趋势 |

统计业务口径只在 [领域模型](./05-domain-model.md) 维护。响应字段随阶段四页面 datasource 用例冻结。

## 8. Schema-UI 映射

| 页面能力 | 目标 API 族 |
|---|---|
| 经营看板 | `/api/v1/dashboard/*` |
| 搜索表格 | `GET /api/v1/orders` |
| 联动表单提交 | 订单创建/编辑 |
| 行级 Action | 订单履约、取消与退款审批 |
| UploadAction | 附件上传与订单绑定 |

页面只使用固定协议已有的 datasource、Action、Reaction 和 mapping 能力。页面 YAML 实施时必须通过 [Schema-UI 固定版本](./02-schema-ui-integration.md) 校验。

## 9. 草案收敛条件

每个 endpoint 从 `draft target` 收敛为当前契约前，至少具备：

- handler/transport 类型与正常、method/path、输入、context、超时、关闭和内部错误测试；
- [验证规则](./04-validation.md)中对应门禁已启用；
- 当前 API 文档和 CHANGELOG 已同步。

仅业务 endpoint 额外要求业务用例/repository 类型、认证、权限、状态与并发测试，以及对应 Schema-UI datasource/Action 场景。`/healthz`、`/readyz` 等运行端点不要求虚构业务用例、repository 或 Schema-UI 映射。

实现完成后，把该 endpoint 移入 [当前 HTTP API](./03-http-api.md)，目标文档只保留尚未实现的草案。