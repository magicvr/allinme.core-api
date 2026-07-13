---
status: archived
plan_id: PLN-0004
owner: 后端团队
created: 2026-07-13
last_updated: 2026-07-13
applies_to: implementation roadmap phase 4 refunds and dashboard
---

# 阶段四：退款与经营看板开发计划

配套路线：[实施路线图](../../06-implementation-roadmap.md)。本阶段在阶段三订单生命周期之上增加退款申请、审批和经营看板；附件、页面 YAML、Schema-UI 页面下发、真实支付网关、财务对账和账号管理不进入本轮。

## 1. 阶段目标

建立可重复演示且可验证的资金变化闭环：

- `operator`/`admin` 对已有已支付订单幂等发起部分或全额退款；
- `approver`/`admin` 审批或拒绝他人提交的待处理退款，申请人与审批人严格分离；
- 审批成功在单一 SQLite 事务中完成退款状态、订单支付状态、订单版本和审计信息更新；
- 四类角色读取由同一订单和退款数据计算的汇总、订单状态分布及 7/30 日趋势；
- 退款列表、订单详情与看板对同一笔资金变化给出一致结果；
- migrate、seed、reset、重启、并发和路由关闭均有自动化证据。

阶段完成不代表页面已实现。阶段六才创建退款队列和经营看板页面，本阶段只冻结并交付页面 datasource/Action 所需的 HTTP 契约。

## 2. 事实源与开工决策

[领域模型](../../05-domain-model.md)继续持有角色、退款状态、金额和看板业务口径；[目标 HTTP API](../../03-http-api-target.md)只持有待实现 endpoint 范围、角色和跨阶段边界；本计划在阶段四 active 期间持有详细 body、query、DTO、错误、短路顺序、实施步骤、迁移策略和发布门禁。P0 是对已冻结内容的交叉复核和 Evidence 记录，不重新发明契约，也不把 body/query/短路/snapshot 等 transport 细节复制进领域模型；若复核发现事实源冲突，必须先更新对应事实源，再开始 migration 和业务实现。实现及集成门禁通过后，详细 HTTP 契约一次性迁入[当前 HTTP API](../../03-http-api.md)，目标 API 删除已实现 endpoint。

以下值是阶段四的冻结决策。P0 必须完成跨文档同步、基线验证和 Evidence；若 P0 复核需要改变任一值，必须先修改本计划和事实源，不能只勾 checklist：

1. `PENDING` 退款占用可退额度：`available = order.totalAmount - completed - pending`，从申请时阻止队列超额；拒绝后释放，完成后从 pending 转为 completed。同一订单允许多笔 `PENDING`，只要合计占用不超额。
2. 订单支付状态映射：completed 为 0 时保持 `PAID`；`0 < completed < total` 为 `PARTIALLY_REFUNDED`；completed 等于 total 为 `REFUNDED`。`UNPAID` 不允许申请退款。
3. 退款审批更新订单 `version + 1` 和 `updatedAt`；申请或拒绝不修改订单版本。退款自身创建为 version 1，approve/reject 成功后原子变为 version 2。
4. 退款申请使用独立 `refund_idempotency_keys` 表和 refund snapshot v1，不改写阶段三 `idempotency_keys.order_id` 约束，也不改变订单创建重放语义。阶段三 order snapshot v1 不新增 capability 或 `availableRefundAmount`，不改变历史 JSON 或 digest；订单 create 重放的事实仍完全来自首次 snapshot，重放映射把 `availableRefundAmount` 固定为 `0`、`canRequestRefund` 固定为 `false`，其余阶段三 capability 仍按当前主体与 snapshot 状态重算，不读取当前订单或退款补字段。
5. 看板金额字段固定拆为 `grossAmount`、`completedRefundAmount` 和 `netAmount = grossAmount - completedRefundAmount`，不再使用含义不明确的单独“成交额”字段。`orderCount` 统计全部订单；`grossAmount` 纳入 paymentStatus 为 `PAID`、`PARTIALLY_REFUNDED`、`REFUNDED` 的订单原始总额，履约状态不影响金额统计；`completedRefundAmount` 只统计 `COMPLETED` 退款。
6. 看板统一使用注入 clock 和 UTC 日历日；`days=7|30` 包含 clock 所在今天和之前 `days-1` 天，并对无数据日期补零。趋势 `orderCount` 统计窗口内创建的全部订单并按 `createdAt` 归属日期；`grossAmount` 只统计纳入金额口径的订单并按 `createdAt` 归属日期；`completedRefundAmount` 按退款 `decidedAt` 归属日期，每日 `netAmount` 为后两项之差。该口径仅用于本地 demo。
7. 所有返回订单 DTO 的 endpoint 在阶段四新增整数分字段 `availableRefundAmount`，范围为 `0..totalAmount`；`UNPAID`/`REFUNDED` 固定为 0，`PAID`/`PARTIALLY_REFUNDED` 按 `totalAmount - pendingAmount - completedAmount` 计算。`canRequestRefund` 改为动态值：主体为 `operator`/`admin`、paymentStatus 为 `PAID`/`PARTIALLY_REFUNDED` 且 `availableRefundAmount > 0` 时为 `true`。订单列表必须批量加载当前页退款占用，禁止逐订单查询；订单 DTO 的 `canApproveRefund` 继续固定为 `false`，具体审批入口只由退款 DTO 的 `canApprove`/`canReject` 表达。
8. 阶段四路由关闭能力固定为测试装配字段 `DisableRefundRoutes` 和 `DisableDashboardRoutes`；前者关闭全部四条退款 endpoint，后者只关闭三个看板 endpoint。两者不增加环境变量，不是生产运行 feature flag。
9. P0 必须复核已同步的事实源：领域模型使用 `grossAmount/completedRefundAmount/netAmount` 口径；目标 API 的退款和看板权限列使用“允许角色”，显式列出 `operator/admin`、`approver/admin` 和 `authenticated`，不得通过“最低权限”推导隐式角色层级。Evidence 记录实际文档位置、复核日期和结果；发现残留才修改。
10. 阶段三 edit 在阶段四扩展事务不变量：DRAFT 订单的新总额必须不小于当前 `PENDING + COMPLETED` 占用；edit 在同一事务内校验全部退款聚合并按新总额重新推导 paymentStatus。合法 edit 可以改变 dashboard gross，本看板明确为当前状态统计而非不可变账本。
11. 退款资格与履约状态正交：只看 paymentStatus 和额度；`CANCELLED + PAID` 与其他履约状态一样允许退款，`UNPAID`/`REFUNDED` 不允许申请。
12. dashboard query 冻结为：summary/order-status 不接受任何 query；trend 必须且只能出现一次非空 `days`，值仅为 `7` 或 `30`。缺失、空值、重复、未知参数及非法值统一返回 `400 INVALID_REQUEST`，detail 指向 `days` 或首个非法参数。
13. 阶段四 schema 目标固定为 `PRAGMA user_version = 6`；固定产物和原始 Evidence 使用 `artifacts/phase4/v6/<revision>/` 与其 `evidence/` 子目录。
14. development seed 使用独立 `refund_demo=1`，保持 `order_demo=1` 的六条阶段三记录及校验不变；固定订单、退款、actor、金额和时间以 §5 表格为准。§5 同时冻结由这些 fixture 推导出的 dashboard 期望值，但 dashboard repository、endpoint 与响应变化只在 M3 验证。

P0 完成后，角色、状态、金额、capability 和统计口径同步到领域模型；目标 API 只同步 endpoint 范围、角色和跨阶段边界；详细 transport 契约保留在本计划和 checklist Evidence。未冻结的字段不得通过 migration 或 handler 偶然成为外部契约。

## 3. 退款领域契约

首版退款模型冻结如下；P0 负责把这些值同步到领域模型并记录 Evidence：

| 项 | 阶段四计划值 |
|---|---|
| ID | `rfd_` + 32 位小写十六进制，使用 `crypto/rand` 生成 |
| 状态 | 持久化状态仅 `PENDING`、`REJECTED`、`COMPLETED`；不持久化 `APPROVED` |
| 金额 | `int64` 分，范围 `1..9_999_999_999`，币种继承订单且首版仅 `CNY` |
| 原因 | 有效 UTF-8，拒绝 NUL；使用 Go `strings.TrimSpace` 后为 `1..500` UTF-8 bytes；允许内部换行并原样保存，digest 不做 CRLF/LF 等价化 |
| 版本 | 创建为 1；approve/reject 成功后为 2；终态不可再次转换 |
| 审计字段 | `requestedBy`、`decidedBy`、`createdAt`、`updatedAt`、`decidedAt`；创建时 `createdAt == updatedAt` 且 `decidedAt/decidedBy` 未设置，approve/reject 成功时 `updatedAt == decidedAt` 且使用同一注入 clock；失败、冲突和幂等重放不更新时间 |
| 申请权限 | `operator`、`admin`；订单必须为已支付或部分退款且仍有可退额度 |
| 审批权限 | `approver`、`admin`；主体 user ID 不得等于 `requestedBy` |
| 金额占用 | `PENDING + COMPLETED` 共同占用，任一并发申请都不得使占用总额超过订单总额 |

三个退款写操作分别冻结以下 service/repository 分类顺序，不使用一套通用顺序替代：

- create miss：订单存在性 → orderVersion → order writer fence 后复核 → 订单及既有退款聚合完整性 → paymentStatus 可退款 → amount 不超过当前 `availableRefundAmount`。不存在返回 `NOT_FOUND`，version 不同返回 `VERSION_CONFLICT`，既有订单/明细/退款行、actor、状态决定字段组合、checked arithmetic、`pending + completed <= totalAmount` 或 paymentStatus 映射损坏返回 `INTERNAL_ERROR`，不可退款支付状态返回 `STATE_CONFLICT`，amount 超出可退额度返回带 `amount` detail 的 `VALIDATION_FAILED`；任何失败都不得创建退款或幂等记录。
- approve：退款存在性 → refund version → `PENDING` 状态 → 申请人与审批人隔离 → order writer fence 后复核 refund/order → 关联订单存在且聚合完整 → completed 金额与 paymentStatus 一致性复核 → 双条件更新。refund 必须按 `id + version + PENDING` CAS，order 必须按 fence 后观察到的 `id + version` CAS；任一更新 0 row 都整体回滚。refund version 与状态同时不匹配时优先 `VERSION_CONFLICT`，self-approval 只在当前 version 的 `PENDING` 退款上返回 `FORBIDDEN`；关联订单缺失、损坏或已破坏额度不变量属于 `INTERNAL_ERROR`。order CAS 因相邻合法写竞争未命中属于可重试 `SERVICE_UNAVAILABLE`，不得伪装成客户端未提交的 order version 冲突；用原 approve 输入重试后必须重新观察订单和全部退款聚合。
- reject：退款存在性 → refund version → `PENDING` 状态 → 申请人与审批人隔离 → 条件更新。reject 不读取或校验订单 paymentStatus/退款额度；version 与状态同时不匹配时优先 `VERSION_CONFLICT`，self-approval 在 version/state 之后返回 `FORBIDDEN`。

create、DRAFT edit 和 approve 都会影响同一订单的退款不变量，必须在读取退款聚合前取得 SQLite writer serialization，并在取得后复核输入或已观察到的 order version。实现可使用事务内 `UPDATE orders SET version = version WHERE id = ? AND version = ?` 作为不改变业务字段的 order writer fence，或使用能提供等价保证的 `BEGIN IMMEDIATE`/repository 原语；不得先基于 deferred read snapshot 完成聚合，再在没有 fence 的情况下提交。create fence 使用客户端 `orderVersion`，edit fence 使用 edit version，approve 在读取退款以定位 order 后使用事务内观察到的 order version，并在 fence 后重新读取 refund/version/PENDING 与订单退款聚合。create fence 不增加订单版本或更新时间；最终业务更新仍遵守申请不改订单 version、edit/approve 各只增加一次的规则。`orderVersion` 防止退款申请与 edit/approve 的订单事实竞争，不是退款序号；在没有 edit/approve 成功提交期间，连续退款申请可以使用同一订单版本。

审批事务中任一步失败必须整体回滚，不允许出现退款 `COMPLETED` 但订单仍为 `PAID` 的中间状态。

approve 的“订单及退款聚合完整性”固定为：先校验关联订单字段与明细总额；再读取该订单全部退款并逐行校验 ID、order ID、金额、状态、version、actor、UTC 时间和状态/决定字段组合，`PENDING` 必须无决定信息，`REJECTED/COMPLETED` 必须具有一致的 `decidedBy/decidedAt/updatedAt`；使用 checked arithmetic 汇总 pending/completed；验证 `pending + completed <= totalAmount`；最后验证当前 paymentStatus 与 completed 汇总映射一致。actor 行缺失、用户名不可读取/非法、退款行损坏、汇总溢出或映射不一致均返回 `INTERNAL_ERROR`，不得用占位 actor 或跳过损坏行。

阶段四同时扩展 DRAFT edit：repository 按输入 version 取得 order writer fence 后，在原订单 version/state 校验和写入事务内读取并验证退款聚合。写入前必须先按旧 totalAmount 验证当前 paymentStatus 与退款历史一致；不得用 edit 顺便修复损坏数据。若现有聚合已损坏，返回 `INTERNAL_ERROR`；若 `newTotalAmount < pending + completed`，返回 `422 VALIDATION_FAILED` 与 `{"field":"items","message":"calculated total must not be less than occupied refund amount"}` 并完整回滚；否则按新总额重新推导 paymentStatus：无退款记录且原状态为 `UNPAID` 时保持 `UNPAID`，有退款记录或原状态为已支付集合时，completed 为 0/部分/等于新总额分别映射为 `PAID/PARTIALLY_REFUNDED/REFUNDED`。edit 成功仍只将订单 version 增加一次。并发 edit/refund create、edit/approve 和 refund create/approve 必须使用两个独立 DB 句柄覆盖；loser 若先得到 unavailable，winner 提交后必须用原输入重试并得到冻结的最终分类，且不得破坏金额、paymentStatus、version、退款状态或明细一致性。

`admin` 同时拥有申请和审批角色，但仍不得审批自己创建的退款。前端 capability 只用于展示；所有 endpoint 必须重新鉴权、校验 version 和状态。

订单退款 capability 属于阶段四现有订单查询契约的增量。`availableRefundAmount = order.totalAmount - pendingAmount - completedAmount`；`canRequestRefund` 按 P0 冻结的角色、paymentStatus 和 availableRefundAmount 计算。订单列表 repository 必须在当前页范围内批量取得退款占用并与订单 COUNT/page query 保持同一只读 SQLite snapshot；订单详情使用同一计算规则。列表和详情读取退款占用时必须验证与 create/approve 相同的退款行基本完整性、checked arithmetic、额度不变量和 paymentStatus 映射；损坏时返回 `INTERNAL_ERROR`，不得截断为 0、跳过损坏行或生成 capability。列表、详情、首次 create、edit 和履约 Action 返回的 Order DTO 都包含 `availableRefundAmount`，退款创建成功后重新读取订单必须立即反映新的金额和 capability。订单 create 幂等重放是明确例外：不得修改 order snapshot v1 或查询当前退款，映射时返回 `availableRefundAmount:0`、`canRequestRefund:false`。订单没有可定位到具体退款的 ID，因此 `canApproveRefund` 保持 `false`，审批能力只出现在退款列表 DTO。

## 4. HTTP 契约冻结范围

计划实现以下 endpoint：

| Method | Path | 角色 | 计划行为 |
|---|---|---|---|
| `GET` | `/api/v1/refunds` | `approver`、`admin` | 按状态、订单和分页查询退款队列及历史 |
| `POST` | `/api/v1/orders/{orderId}/refunds` | `operator`、`admin` | 使用 `Idempotency-Key` 创建退款申请 |
| `POST` | `/api/v1/refunds/{refundId}/approve` | `approver`、`admin` | 审批并在同一事务完成本地退款 |
| `POST` | `/api/v1/refunds/{refundId}/reject` | `approver`、`admin` | 拒绝待处理退款 |
| `GET` | `/api/v1/dashboard/summary` | authenticated | 返回 `orderCount/grossAmount/completedRefundAmount/netAmount/currency` |
| `GET` | `/api/v1/dashboard/order-status` | authenticated | 按 `DRAFT, CONFIRMED, FULFILLING, SHIPPED, COMPLETED, CANCELLED` 返回数量，缺失状态补零 |
| `GET` | `/api/v1/dashboard/trend` | authenticated | `days=7|30` 的逐日 `orderCount/grossAmount/completedRefundAmount/netAmount` |

退款申请 body 固定为 `{"amount":100,"reason":"customer request","orderVersion":3}`；approve/reject body 固定为 `{"version":1}`。三个写 endpoint 的 body 上限固定为 8 KiB。所有整数沿用阶段三严格整数词法，拒绝字符串、小数、指数、`null` 和溢出；请求沿用 `application/json`、未知字段拒绝、单一 JSON 值和稳定错误 envelope。transport 必须在 JSON 解码前对实际 body bytes 执行 `utf8.Valid`，因为 Go `encoding/json` 会把非法字节替换为 `U+FFFD`；非法 UTF-8 固定返回 `400 INVALID_REQUEST`，不得创建退款或幂等记录。领域层仍独立验证 `utf8.ValidString`、NUL、trim 后 byte 长度和内部换行保存。amount 小于 1 或超过 `9_999_999_999` 返回 `422 VALIDATION_FAILED` 与 `{"field":"amount","message":"must be between 1 and 9999999999"}`；create miss 的 amount 超过当前可退额度返回 `{"field":"amount","message":"must not exceed availableRefundAmount"}`。

HTTP 与业务短路分别冻结如下：

- create：已注册 route/method → CORS → authentication → `operator/admin` role → Content-Type → 原始 body UTF-8 → `Idempotency-Key` → orderId path 形状 → JSON 结构/整数词法 → typed normalization → idempotency lookup → create miss 的订单业务分类顺序。
- approve/reject：已注册 route/method → CORS → authentication → `approver/admin` role → Content-Type → 原始 body UTF-8 → refundId path 形状 → JSON 结构/整数词法 → typed version validation → 对应 approve/reject 业务分类顺序。
- 格式非法和格式合法但不存在的 orderId/refundId 都返回 `404 NOT_FOUND`；authorization 先于 Content-Type，Content-Type 先于 path ID 和 JSON。handler 只执行 transport 短路，资源/version/state/self-approval 的相对顺序由 service/repository 保证。

退款列表只接受 `status`、`orderId`、`page`、`pageSize`；未知参数、重复参数以及 `sort/order` 均返回 `400 INVALID_REQUEST`。status 只允许 `PENDING/REJECTED/COMPLETED`，orderId 使用冻结格式；page 默认 1 且至少为 1，pageSize 默认 20、范围 1..100，整数和 offset 溢出规则沿用订单列表。排序固定为 `createdAt DESC, id DESC`。成功返回 `200` 与 `{"items":[],"total":0,"page":1,"pageSize":20}`。repository 必须让 COUNT、page rows 和 requestedBy/decidedBy actor hydration 共享同一显式只读事务；actor 按当前页去重 ID 批量加载，禁止逐退款查询，并以 query observer 或等价证据固定查询次数上界。并发 create/approve 提交期间，单个响应的 total、items、actor 和 capability 必须来自同一 SQLite snapshot。

退款 DTO 字段顺序固定为 `id/orderId/amount/currency/reason/status/version/requestedBy/decidedBy/createdAt/updatedAt/decidedAt/canApprove/canReject`；actor 为 `{"id":"...","username":"..."}`，未决定时 `decidedBy` 和 `decidedAt` 均为 JSON `null`。`canApprove/canReject` 仅在当前主体为 `approver/admin`、退款为 `PENDING` 且 requestedBy 不是当前 user ID 时为 `true`。actor 关联行缺失、ID/username 损坏或 username 不可读取返回 `500 INTERNAL_ERROR`，不得返回空字符串或占位用户。

`GET /api/v1/refunds` 继续只允许 `approver/admin`，用于阶段六审批队列。阶段四不提供 operator 的逐笔退款历史；operator 通过订单 `paymentStatus`、`availableRefundAmount` 和 `canRequestRefund` 观察聚合结果。若阶段六确认需要申请人逐笔跟踪，必须先扩展目标 API 和本计划，不得复用审批队列时临时放宽角色。

幂等 header、字符集、长度和错误语义复用阶段三冻结值。退款幂等唯一作用域为 `principal user ID + POST + operation + normalized orderId + key`；operation 固定为 `POST /api/v1/orders/{orderId}/refunds`。因此同一主体可在不同 orderId 上复用同一 key 并分别成功，也可复用已用于 `POST /api/v1/orders` 的 key；同一 orderId 作用域内相同 key 的不同 normalized request 才返回幂等冲突。normalized request 使用固定字段顺序编码 canonical JSON：`operation`、`orderId`、`amount`、`reason`、`orderVersion`；orderId 为规范化 path ID，reason 使用领域 trim 后值。原始 JSON 空白、对象字段顺序和 reason 外围空白不同视为同一请求；内部换行、大小写或其他事实差异产生不同 digest。摘要为 canonical JSON 的 SHA-256。

完成 transport、主体授权、key 和 normalized request 校验后，repository/service 必须先查幂等记录；命中同一请求时直接校验并返回首次 refund snapshot v1，不再检查当前 order version、paymentStatus、availableRefundAmount 或当前退款状态。不同 request digest 返回 `409 IDEMPOTENCY_CONFLICT`。审批完成后再次重放原创建请求仍返回首次 snapshot，且不读取当前退款或订单补字段、不重复占用额度、不更新时间。

refund snapshot v1 的顶层固定为 `{"refund":{...}}`，内部 refund 字段与首次 `201 Created` 的裸 Refund DTO 完全一致：创建事实、`PENDING`、version 1、`decidedBy:null`、`decidedAt:null`、`canApprove:false`、`canReject:false`。snapshot 保存这些 capability，不在重放时重新计算；snapshot metadata 独立保存并核对 principal/method/operation/orderId/key scope、request digest、refund ID、snapshot version、snapshot digest 和 createdAt。未知版本、JSON/摘要/字段/元数据损坏返回 `500 INTERNAL_ERROR`，不映射为 `SERVICE_UNAVAILABLE` 且绝不重新创建。本阶段幂等记录只保留、不清理，development reset 统一删除；生产保留/清理策略属于后续运维工作和剩余风险。

退款 create 首次与重放均返回 `201 Created` 和裸 Refund DTO；approve/reject 成功返回 `200 OK` 和本次写入后的当前裸 Refund DTO，终态 capability 均为 false；不使用 `204`。dashboard 三个 endpoint 均返回 `200 OK`，JSON 结构与字段顺序固定如下，日期为 UTC `YYYY-MM-DD`，金额均为整数分：

```json
{"orderCount":10,"grossAmount":460000,"completedRefundAmount":120000,"netAmount":340000,"currency":"CNY"}
```

```json
{"items":[{"status":"DRAFT","count":1},{"status":"CONFIRMED","count":1},{"status":"FULFILLING","count":2},{"status":"SHIPPED","count":2},{"status":"COMPLETED","count":2},{"status":"CANCELLED","count":2}]}
```

```json
{"days":7,"startDate":"2026-01-01","endDate":"2026-01-07","items":[{"date":"2026-01-01","orderCount":10,"grossAmount":460000,"completedRefundAmount":0,"netAmount":460000},{"date":"2026-01-02","orderCount":0,"grossAmount":0,"completedRefundAmount":120000,"netAmount":-120000},{"date":"2026-01-03","orderCount":0,"grossAmount":0,"completedRefundAmount":0,"netAmount":0},{"date":"2026-01-04","orderCount":0,"grossAmount":0,"completedRefundAmount":0,"netAmount":0},{"date":"2026-01-05","orderCount":0,"grossAmount":0,"completedRefundAmount":0,"netAmount":0},{"date":"2026-01-06","orderCount":0,"grossAmount":0,"completedRefundAmount":0,"netAmount":0},{"date":"2026-01-07","orderCount":0,"grossAmount":0,"completedRefundAmount":0,"netAmount":0}]}
```

summary 的 `netAmount` 在聚合不变量成立时不得为负，负值视为损坏数据并返回 `INTERNAL_ERROR`。trend 每日 bucket 的 `netAmount` 允许为负，例如窗口外创建的订单在窗口内完成退款；checked subtraction 必须接受合法负数，只拒绝 int64 溢出。

dashboard query 规则固定为：`/summary` 和 `/order-status` 的 query 必须为空，任一参数（包括空值或重复值）返回 `400 INVALID_REQUEST`；`/trend` 必须且只能提供一次非空 `days`，值为严格整数词法 `7` 或 `30`，缺失、空值、重复、未知参数、字符串变体、符号、前导零或其他值均返回 `400 INVALID_REQUEST`。错误 detail 的 `field` 对缺失/空值/非法值为 `days`，对未知参数为按原始 query 顺序遇到的首个参数名。

新增退款错误语义时优先复用现有 `INVALID_REQUEST`、`FORBIDDEN`、`NOT_FOUND`、`VERSION_CONFLICT`、`STATE_CONFLICT`、`IDEMPOTENCY_CONFLICT`、`VALIDATION_FAILED`、`INTERNAL_ERROR` 和 `SERVICE_UNAVAILABLE`。若确需新增机器码，先在本计划冻结；对应 endpoint 实现和集成门禁通过后随完整契约迁入当前 API，不在目标 API 维护第二份详细错误表。

所有阶段四 endpoint 继续明确拒绝 `HEAD`：已知 path 返回 `405 METHOD_NOT_ALLOWED`，`Allow` 只包含实际业务 method，不包含 `HEAD`。配置可信 origin 且携带完整合法 preflight headers 的 `OPTIONS` 由 CORS middleware 返回 `204`；带 `Origin` 但缺失/非法 `Access-Control-Request-Method` 的请求返回 `403 CORS_PREFLIGHT_DENIED`；未配置 CORS或不带 `Origin` 的普通 `OPTIONS` 沿用 mux 的稳定 `405`；未知 path 保持 `404`。route metadata、mux 注册和 CORS 判断必须来自同一组实际启用路由。

## 5. 数据库与迁移策略

阶段四使用仅前进 migration，从当前 v5 增加单个 additive migration v6，并在成功后固定 `PRAGMA user_version = 6`：

- `refunds`：退款事实、version、申请/审批主体和时间；
- `refund_idempotency_keys`：独立保存 principal、method、operation、normalized order ID、key 的唯一作用域，以及 request digest、refund ID、snapshot version/JSON/digest 和创建时间；
- 支持退款队列、订单退款汇总和趋势统计的必要索引。至少覆盖以下查询意图：退款幂等作用域唯一键；`refunds(order_id, status)` 聚合；`status + created_at + id` 和 `order_id + created_at + id` 的列表筛选/稳定排序；`status + decided_at` 的 completed refund 趋势。具体索引名和列合并方式可按 `EXPLAIN QUERY PLAN` 调整，但 Evidence 必须逐项说明由哪个索引承接。

migration 必须保护 ID、状态、金额、version、时间文本和外键基本形状。跨行可退金额、申请审批隔离、支付状态计算和 snapshot 语义由 service/repository 事务保证，不尝试用难以审计的触发器复制业务规则。

schema-only 是行为实现前的阻断门禁，执行顺序固定为：先只实现 v6 migration、兼容性读取、migration/readiness 测试；立即记录 Git revision、构建产物并完成新 schema 上的阶段一至三 smoke 和阶段四 404 smoke；只有该 Evidence 完成后，才允许实现 `refund_demo`、退款 service/repository、Order DTO、edit/capability 或注册阶段四路由。固定 revision 只包含 v6 migration、兼容性读取及其测试。使用该 revision 构建正常 `cmd/api`/`cmd/admin`，产物保存在 `artifacts/phase4/v6/<revision>/`，原始命令、hash、数据库和 smoke 输出保存在其 `evidence/` 子目录；记录 revision、SHA-256、构建/启动命令和数据库恢复点。该产物理解新 schema，可作为部署级阶段四功能回退，阶段一至三响应契约与固定 snapshot 必须保持一致，全部阶段四 endpoint 为 404。`DisableRefundRoutes`/`DisableDashboardRoutes` 只提供后续代码级兼容性和测试证据，不是已部署 binary 的运行时开关。若固定 revision 已混入 Order DTO/edit/capability 变化，只能称为 `pre-route compatible binary`，不得作为 schema-only Evidence，且 M1-A 不得视为完成。若必须回到无法识别新 schema 的旧 v5 binary，只允许在停写窗口恢复 migration 前完整 SQLite 主库/WAL/SHM 集合，禁止旧 binary 直接打开新版本数据库；schema-only 固定产物证据与旧 binary 整库恢复证据必须分别记录。

development seed 新增独立 `refund_demo=1`，执行顺序固定为 `runtime → auth_demo → order_demo → refund_demo`；不得 bump 或改写 `order_demo=1`。`refund_demo` 新增四个订单及其单明细，并按 username 查询 `auth_demo` 的用户 ID 写入 actor；固定 ID 只用于 seed：

| order ID | status / paymentStatus / version | total / createdAt / updatedAt | 退款基线 |
|---|---|---|---|
| `ord_00000000000000000000000000000007` | `FULFILLING / PAID / 1` | `70000 / 2026-01-01T06:00:00Z / 同 createdAt` | 无退款，可全额申请 |
| `ord_00000000000000000000000000000008` | `CANCELLED / PAID / 1` | `80000 / 2026-01-01T07:00:00Z / 同 createdAt` | 两笔 pending 共 15000、一笔 rejected 5000，证明履约正交、多 pending 和拒绝释放 |
| `ord_00000000000000000000000000000009` | `SHIPPED / PARTIALLY_REFUNDED / 2` | `90000 / 2026-01-01T08:00:00Z / 2026-01-02T05:00:00Z` | completed 20000，可继续退 70000 |
| `ord_0000000000000000000000000000000a` | `COMPLETED / REFUNDED / 2` | `100000 / 2026-01-01T09:00:00Z / 2026-01-02T07:00:00Z` | completed 100000，不可再退 |

item ID 使用与 order 尾号相同的 `itm_...07` 至 `itm_...0a`，quantity 为 1、unitPrice 等于订单总额。退款固定如下；申请时间按表给出，终态的 `updatedAt == decidedAt`，pending 的 `updatedAt == createdAt`：

| refund ID | order / amount / status / version / reason | requestedBy → decidedBy | createdAt / decidedAt |
|---|---|---|---|
| `rfd_00000000000000000000000000000001` | `...08 / 10000 / PENDING / 1 / pending primary` | `operator → null` | `2026-01-02T00:00:00Z / null` |
| `rfd_00000000000000000000000000000002` | `...08 / 5000 / PENDING / 1 / pending secondary` | `admin → null` | `2026-01-02T01:00:00Z / null` |
| `rfd_00000000000000000000000000000003` | `...08 / 5000 / REJECTED / 2 / rejected request` | `operator → admin` | `2026-01-02T02:00:00Z / 2026-01-02T03:00:00Z` |
| `rfd_00000000000000000000000000000004` | `...09 / 20000 / COMPLETED / 2 / partial refund` | `operator → approver` | `2026-01-02T04:00:00Z / 2026-01-02T05:00:00Z` |
| `rfd_00000000000000000000000000000005` | `...0a / 100000 / COMPLETED / 2 / full refund` | `admin → approver` | `2026-01-02T06:00:00Z / 2026-01-02T07:00:00Z` |

固定 summary 快照为 `orderCount=10, grossAmount=460000, completedRefundAmount=120000, netAmount=340000, currency=CNY`；order-status 为 `DRAFT=1, CONFIRMED=1, FULFILLING=2, SHIPPED=2, COMPLETED=2, CANCELLED=2`。注入 clock `2026-01-07T12:00:00Z` 的 7 日 trend 在 `2026-01-01` 为 `orderCount=10/grossAmount=460000/completedRefundAmount=0/netAmount=460000`，在 `2026-01-02` 为 `0/0/120000/-120000`，其余日期补零。M1 的重复 seed 只读验证所有固定订单、明细、退款、actor 关系、版本、时间及可独立计算的 fixture 算术，不覆盖用户新增数据；M3 再通过 dashboard repository 和三个 endpoint 验证上述响应快照以及退款后的响应变化。production 不运行或写入 `refund_demo`。

## 6. 代码边界

- `internal/order` 持有退款 command、状态机、金额、capability、dashboard query/result 和 service；暂不新增独立顶层业务模块，避免订单与退款事务边界被拆散。
- `internal/store` 实现退款 repository、幂等 snapshot 和 dashboard 聚合 SQL；handler 不直接执行 SQL。
- `internal/httpapi` 只负责 route、认证授权、query/JSON 解码、大小限制和错误映射。
- `internal/app` 显式装配退款和看板依赖，并在测试装配中提供独立的 `DisableRefundRoutes`、`DisableDashboardRoutes`；不得增加对应环境配置或生产 feature flag。
- `internal/protocol` 不加入退款业务规则；页面 Action/Reaction 仍由阶段六验证。

所有 clock、ID generator 和 repository 依赖保持可注入。dashboard 必须使用注入 clock 计算 7/30 日窗口，测试不得依赖开发数据库、真实当前时间或执行顺序。

## 7. 端到端里程碑

### M1：退款核心与 schema

M1 允许拆成多个 PR，但顺序和完成定义固定，不能用后续 PR 补造前置 revision：

- **M1-A schema barrier**：冻结 P0，完成 v6 migration、兼容读取、migration/readiness 测试，并在任何 Order DTO/edit/capability/refund seed/service 变化前构建和验证 schema-only 固定产物；
- **M1-B refund core**：完成领域类型、repository contract、`refund_demo`、create/idempotency、approve/reject、`availableRefundAmount`、snapshot v1、订单支付状态更新和动态 `canRequestRefund`；
- **M1-C cross-invariants**：扩展 DRAFT edit，完成损坏聚合、超额、同/不同 key、跨 order key 复用、canonical digest、snapshot 损坏、并发审批、edit/create、edit/approve、create/approve、旧 version、self-approval、双 CAS、writer fence、rollback 和旧 v5 binary 整库恢复证据。

M1-A/B/C 可分别 review 和合并，但 M1 只有在 checklist 对应项全部完成后才完成。

### M2：退款 HTTP 闭环

- 一次性注册 list/create/approve/reject 四条退款路由；
- 使用四角色真实 token 覆盖允许和拒绝路径；
- CORS route metadata、method/Allow、Content-Type、原始 body UTF-8、body/query、取消和 unavailable 行为与现有 HTTP 栈一致；`HEAD` 为 405，合法 preflight `OPTIONS` 为 204，普通 `OPTIONS` 沿用现有拒绝；
- 退款列表 COUNT、page rows 和 actor 批量 hydration 使用同一只读 snapshot，并用 query observer 或等价证据禁止 N+1；
- 自动化主路径在首次部分退款完成后再次申请并审批，证明二次退款后的额度、refund/order version、paymentStatus 与订单/退款聚合一致；该流程引起的看板响应变化由 M3 验证；
- `DisableRefundRoutes` 关闭全部退款 endpoint，`DisableDashboardRoutes` 不影响退款；两个字段仅提供测试装配和代码级兼容性证据；
- 启用验证矩阵“退款”，并把已实现退款 endpoint 迁入当前 API 文档。

### M3：经营看板

- 实现 summary、order-status 和 trend 三个只读 endpoint；
- 单次响应在一致的只读 SQLite snapshot 中完成相关聚合，避免跨查询观察到不同提交；
- 固定 seed 统计快照与订单/退款查询交叉一致；
- 自动化流程完成退款后再次查询看板，证明 `completedRefundAmount`、`netAmount` 和支付状态同步变化；
- 注入 clock 覆盖 UTC 午夜、窗口首日、窗口外订单和窗口外退款；
- 启用验证矩阵“看板”，并把已实现 endpoint 迁入当前 API 文档。

### R：发布与归档准备

- 全仓普通测试、vet、独立 race、文档 validator 和 diff check 通过；
- fresh migrate + seed、重启持久化、reset 恢复和阶段一至三回归通过；
- 更新 overview、architecture、当前/目标 API、validation、domain model、roadmap、scenario 和 CHANGELOG；
- 提交完成报告、CI 证据和剩余风险；只有用户明确确认后才归档 plan/checklist。

## 8. 风险与控制

| 风险 | 控制 |
|---|---|
| 并发申请导致总退款超额 | 事务内聚合 `PENDING + COMPLETED` 并竞争写入；双 DB 句柄并发测试 |
| 重复审批造成二次退款 | refund version + PENDING 条件更新；同 version 并发最多一次成功 |
| 不同退款审批丢失订单版本/支付状态更新 | refund 与 order 双 CAS；edit/approve、create/approve 双连接竞争及原输入重试 |
| 申请人通过 admin 身份自批 | 按稳定 user ID 比较，不按角色或用户名推断 |
| 复用旧幂等表破坏订单 snapshot | 使用独立退款幂等表和 snapshot v1，不修改旧记录语义 |
| 看板多查询观察到不同数据 | 显式只读事务和交叉一致性测试 |
| 退款列表 total/items/actor 跨提交不一致 | COUNT、page rows、actor 批量 hydration 共享显式只读事务 |
| 退款金额被重复扣减 | 只暴露 `grossAmount/completedRefundAmount/netAmount`，并冻结 paymentStatus 与履约状态纳入口径 |
| 趋势口径含糊 | P0 冻结注入 clock、UTC、日期归属、区间和补零规则 |
| 订单列表 capability 引入 N+1 | 当前页批量退款占用查询，与订单页保持同一只读 snapshot |
| migration 后回退误用旧 binary | 保存理解新 schema 的 schema-only 固定产物；测试开关不冒充运行时回退；旧 binary 仅可配合停写整库恢复 |
| edit 与退款竞争破坏金额不变量 | edit 同事务校验全部退款占用、重新推导 paymentStatus，并做双连接竞争与失败回滚测试 |
| 非法 UTF-8 被 JSON 解码器替换后绕过校验 | 解码前校验原始 body bytes，领域层再校验 normalized string |
| 聚合查询随数据量恶化 | 为筛选/日期/外键建立索引，并用代表性 fixture 检查 query plan；本阶段不承诺生产 SLA |

## 9. 非目标

- 附件上传、绑定、下载和清理；
- 页面 YAML、页面列表或页面 JSON API；
- 阶段六退款队列固定以 `/api/v1/refunds` 为 datasource/Action，不把审批塞入订单列表行 Action；若产品方向改变，先修改目标 API、场景和本计划，再改实现；
- 真实支付渠道、异步回调、账务分录、对账和发票；
- 多币种、汇率、税费、促销分摊和按订单项退款；
- 退款撤销、重开、批量审批和导出；
- 生产级 OLAP、缓存、预聚合或性能 SLA；
- refresh token、分布式限流、账号管理和多租户。

## 10. 完成标准

阶段四只有在以下条件全部满足后才可报告完成：

1. operator 发起的退款可由不同 user ID 的 approver 审批或拒绝，self-approval 永远失败且数据不变；
2. 相同幂等请求只创建一条退款，不同 body 冲突，并发申请和审批不能超额或重复执行；edit/create、edit/approve、create/approve 相邻竞争均有双连接与重试证据；
3. 审批成功后退款、订单 paymentStatus、订单 version 和 dashboard 在同一提交后保持一致；refund/order 双 CAS 任一步失败整体回滚；edit/refund 并发不能使占用超过新总额或借 edit 修复损坏聚合；除订单 create 重放固定返回 `availableRefundAmount=0/canRequestRefund=false` 外，所有 Order DTO 的 `canRequestRefund` 与实际 `availableRefundAmount` 一致且无逐订单查询，`canApproveRefund` 保持 `false`；
4. summary、order-status 和 7/30 日 trend 使用冻结的 `grossAmount/completedRefundAmount/netAmount` 口径，对固定 seed、注入 clock 边界及退款变化给出可重复结果；
5. 阶段四 endpoint 具备真实认证 HTTP、权限、原始 UTF-8、transport、并发、损坏数据和回退测试；退款列表和看板单响应均来自一致只读 snapshot；
6. `go test ./... -count=1`、`go vet ./...`、`go test -race ./... -count=1` 与文档门禁通过；
7. 当前/目标 API、验证矩阵、领域模型、路线图、场景和 CHANGELOG 与实现一致；
8. 完成报告记录远端 CI、剩余风险和归档确认状态。
