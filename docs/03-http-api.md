---
status: active
owner: 后端团队
last_updated: 2026-07-13
applies_to: implemented allinme.core-api HTTP API
---

# 当前 HTTP API

本文只记录当前源码和测试已经实现的 HTTP 行为。待实现的业务原则与 endpoint 草案见 [目标 HTTP API](./03-http-api-target.md)，不得据此推断目标接口当前可调用。

## `GET /healthz`

用于进程存活检查（liveness）。只要 HTTP 进程可以处理请求即返回成功，不检查数据库、migrations、文件目录、页面配置或外部依赖。

成功响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

实现位于 `internal/httpapi/handler.go`，行为由 `internal/httpapi/handler_test.go` 验证。

## `GET /readyz`

数据库文件可访问且 `PRAGMA user_version` 等于二进制最新 migration 版本时返回：

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ready"}
```

数据库缺失、不可用、未迁移、低于或高于支持版本时返回 `503`：

```json
{"error":{"code":"NOT_READY","message":"service is not ready","requestId":"req_..."}}
```

非 `GET` 请求返回 `405`、`Allow: GET` 和 `METHOD_NOT_ALLOWED` envelope。阶段六启用页面模块时再扩展 readiness 检查。

## 认证 API

`POST /api/v1/auth/login` 只接受不超过 4 KiB 的严格 JSON `{"username","password"}`。成功返回 `200`、短期 HS256 JWT、`Bearer` token type、UTC expiry 和用户摘要；未知账号、错误密码和禁用账号统一返回 `401 AUTHENTICATION_FAILED`。每个规范化 username 与客户端 IP 组合在一分钟内最多五次失败，第六次返回 `429 RATE_LIMITED` 和 `Retry-After`。

`GET /api/v1/auth/me` 要求唯一 `Authorization: Bearer <token>`，scheme 大小写不敏感。成功返回当前用户、角色和 token expiry；JWT、session、用户状态或当前角色任一失效时返回 `401 UNAUTHENTICATED` 和 `WWW-Authenticate: Bearer`。

`POST /api/v1/auth/logout` 使用相同认证链路，只撤销当前 token 对应 session，成功返回 `204` 空 body。撤销后旧 token 返回 `401 UNAUTHENTICATED`，同一账号的其他 session 不受影响。

三个 endpoint 的错误 method 分别返回 `405`、准确 `Allow` 和统一错误 envelope；login 的非 JSON Content-Type 返回 `415 UNSUPPORTED_MEDIA_TYPE`，字段、结构、大小或密码 byte 边界错误返回 `400 INVALID_REQUEST`。实现与测试位于 `internal/httpapi`、`internal/auth` 和 `internal/app`。

## 订单查询 API

`GET /api/v1/orders` 要求有效 Bearer token，四个当前角色均可读取。query 支持 `q`、`status`、`paymentStatus`、`createdFrom`、`createdTo`、`page`、`pageSize`、`sort` 和 `order`；畸形百分号编码、非编码分号、重复或未知参数返回 `400 INVALID_REQUEST`。默认 `page=1`、`pageSize=20`、`sort=createdAt`、`order=desc`，排序字段只允许 `createdAt`、`updatedAt`、`totalAmount`、`customerName`、`status`，并使用订单 ID 作为同方向稳定次序。响应固定为 `items/total/page/pageSize`，列表项不含明细；repository 仍会批量校验当前页每个订单的明细数量、position、UTF-8、checked amount 和总额一致性，损坏聚合返回 `500 INTERNAL_ERROR`。

`GET /api/v1/orders/{orderId}` 返回订单及按 position 排序的 `items`。非法格式和不存在的订单 ID 均返回 `404 NOT_FOUND`。除订单 create 幂等重放的冻结 snapshot v1 外，所有订单 DTO 都返回整数分 `availableRefundAmount`；`canRequestRefund` 仅在主体为 operator/admin、paymentStatus 为 `PAID/PARTIALLY_REFUNDED` 且可退金额大于零时为 `true`，订单 DTO 的 `canApproveRefund` 恒为 `false`。列表在同一只读 snapshot 中批量加载当前页退款占用，详情复用同一聚合规则；损坏退款、超额占用或 paymentStatus 映射不一致返回 `500 INTERNAL_ERROR`。已知订单 path 的错误 method 返回 `405 METHOD_NOT_ALLOWED`；集合、详情和 Action 的 `Allow` 分别为 `GET, POST`、`GET, PATCH` 和 `POST`，`HEAD` 不受支持。

订单时间均为 UTC RFC3339，金额使用最小货币单位。当前错误沿用 request ID envelope，并增加可选 `error.details` 字段；可定位的 query 错误使用 `field/message` 元素，内部数据库和扫描错误只对外返回安全的 `500 INTERNAL_ERROR`。

## 订单草稿写入 API

`POST /api/v1/orders` 仅允许 operator、admin，并要求唯一的 `Idempotency-Key` header；key 为 `1..128` bytes，匹配 `[A-Za-z0-9][A-Za-z0-9._:-]{0,127}`。请求严格接受 `customerName/currency/items`，明细包含 `sku/name/quantity/unitPrice`，拒绝未知字段、多余 JSON、非整数词法和超过 64 KiB 的 body。服务端 trim 字符串、只接受 `CNY` 并计算总额。首次与相同 normalized body 的重放都返回 `201`；相同主体、method、route/key 的不同 body 返回 `409 IDEMPOTENCY_CONFLICT`。

创建结果保存为不可变 snapshot v1。订单后续被编辑后，重放仍返回首次创建时 version 1、原始时间和明细，不读取当前订单补字段；重放会先用独立保存的 `snapshot_digest` 校验完整 JSON，再从快照中的创建事实重新计算 normalized request digest，并拒绝重复明细 ID。v5 迁移会在同一事务内为 v4 历史 snapshot JSON 回填摘要，因此合法历史记录继续重放。未知版本、损坏或被篡改的 JSON、合法但被替换的明细 ID、重复明细 ID 及元数据错配安全返回 `500 INTERNAL_ERROR`，不会重新创建。幂等作用域包含 principal user ID、method、route 和 key。

`PATCH /api/v1/orders/{orderId}` 仅允许 operator、admin，严格接受与创建相同的业务字段及正整数 `version`；`Idempotency-Key` 即使存在也被忽略。只有当前 version 的 `DRAFT` 可编辑，成功在单一事务内重写明细和总额、保持 `createdAt`、更新 `updatedAt` 并将 version 加一，返回 `200`。不存在、version 冲突和状态冲突分别返回 `404 NOT_FOUND`、`409 VERSION_CONFLICT` 和 `409 STATE_CONFLICT`。

编辑在取得当前 version 的订单 writer fence 后，于同一事务校验旧订单、明细和退款聚合；新总额不得小于 `PENDING + COMPLETED` 占用，否则返回 `422 VALIDATION_FAILED` 与 `items` detail。合法编辑按新总额和已完成退款重新推导 paymentStatus，订单 version 只增加一次；损坏聚合返回 `500 INTERNAL_ERROR` 且不得由编辑顺带修复。edit/refund create、edit/approve 的跨连接竞争由 writer fence 和 CAS 序列化，锁竞争可暂时返回 `503 SERVICE_UNAVAILABLE`，原输入重试后稳定分类。

typed command 业务校验返回 `422 VALIDATION_FAILED` 与 camelCase 字段详情；缺失/非法幂等 header 或 JSON 结构/类型/整数词法返回 `400 INVALID_REQUEST`，非 JSON Content-Type 返回 `415 UNSUPPORTED_MEDIA_TYPE`。写请求按 role authorization → Content-Type → idempotency key/order ID → JSON 的顺序短路。SQLite BUSY/LOCKED 通过驱动错误码映射为 `503 SERVICE_UNAVAILABLE` 和 `Retry-After: 1`。

## 订单履约 Action API

`POST /api/v1/orders/{orderId}/confirm`、`fulfill`、`ship`、`complete` 和 `cancel` 仅允许 operator、admin。Action body 仅允许 `{"version":n}`，上限 1 KiB；Content-Type、严格 JSON、整数词法和 typed version 校验与草稿编辑保持相同错误语义。

状态依次为 `DRAFT -> CONFIRMED -> FULFILLING -> SHIPPED -> COMPLETED`；cancel 只允许从 `DRAFT`、`CONFIRMED`、`FULFILLING` 进入 `CANCELLED`。成功原子递增 version、更新 `updatedAt` 并返回含有序 items 和当前 capability 的订单；失败不改变状态或时间。不存在、version 不匹配和源状态非法分别返回 `404 NOT_FOUND`、`409 VERSION_CONFLICT` 和 `409 STATE_CONFLICT`，version 与状态同时不匹配时优先返回 version 冲突。

## 退款 API

`GET /api/v1/refunds` 仅允许 approver、admin。query 只接受可选 `status=PENDING|REJECTED|COMPLETED`、规范 `orderId`、`page` 和 `pageSize`；默认 `page=1/pageSize=20`、最大 100，固定按 `createdAt DESC, id DESC` 排序。`sort/order`、未知、重复、空或非法参数返回 `400 INVALID_REQUEST`。COUNT、page rows 和 actor 批量加载共享显式只读事务；成功返回 `items/total/page/pageSize`。Refund DTO 固定包含 `id/orderId/amount/currency/reason/status/version/requestedBy/decidedBy/createdAt/updatedAt/decidedAt/canApprove/canReject`，未决定字段为 JSON `null`，审批 capability 仅对非申请人的 approver/admin 待处理退款为 true。

`POST /api/v1/orders/{orderId}/refunds` 仅允许 operator、admin，要求唯一合法 `Idempotency-Key`，严格接受不超过 8 KiB 的 `{"amount":n,"reason":"...","orderVersion":n}`。transport 在 JSON 解码前验证原始 UTF-8 bytes；未知字段、多余 JSON、字符串/小数/指数/`null` 整数、非法 UTF-8 或超限 body 返回 `400 INVALID_REQUEST`。amount/reason/orderVersion 的 typed 边界失败返回 `422 VALIDATION_FAILED`。首次创建和相同 normalized request 重放均返回 `201` 与首次 PENDING Refund DTO；同一主体、operation、orderId 和 key 的不同事实返回 `409 IDEMPOTENCY_CONFLICT`。重放校验独立 refund snapshot v1 和摘要，不重新读取当前订单或退款，也不重复占用额度。

`POST /api/v1/refunds/{refundId}/approve` 和 `/reject` 仅允许 approver、admin，严格接受不超过 8 KiB 的 `{"version":n}` 并执行相同原始 UTF-8/JSON/整数门禁。approve 按 existence → version → PENDING → self-approval → order fence/聚合 → refund/order 双 CAS 分类，在单一事务完成退款并更新订单 paymentStatus/version/updatedAt；reject 按 existence → version → PENDING → self-approval → refund CAS 分类，不读取或修改订单。成功返回 `200` 当前终态 Refund DTO；不存在、版本、状态和 self-approval 分别映射为 `404 NOT_FOUND`、`409 VERSION_CONFLICT`、`409 STATE_CONFLICT` 和 `403 FORBIDDEN`。

四条退款 path 均拒绝 `HEAD`，错误 method 的 `Allow` 只包含实际业务 method。SQLite BUSY/LOCKED 返回 `503 SERVICE_UNAVAILABLE` 与 `Retry-After: 1`；损坏 actor、退款事实、snapshot、聚合或关联返回安全的 `500 INTERNAL_ERROR`。

## 经营看板 API

三个看板 endpoint 均要求有效 Bearer token，viewer、operator、approver、admin 都可读取。`GET /api/v1/dashboard/summary` 不接受 query，返回 `orderCount/grossAmount/completedRefundAmount/netAmount/currency`；orderCount 统计全部订单，gross 只纳入当前 paymentStatus 为 `PAID/PARTIALLY_REFUNDED/REFUNDED` 的订单原始总额，completedRefundAmount 只纳入 `COMPLETED` 退款，summary net 为 checked `gross-completed` 且不得为负。

`GET /api/v1/dashboard/order-status` 不接受 query，按 `DRAFT, CONFIRMED, FULFILLING, SHIPPED, COMPLETED, CANCELLED` 固定顺序返回全部状态，缺失状态补零。

`GET /api/v1/dashboard/trend` 必须且只能提供一次严格 `days=7|30`；缺失、空值、重复、未知、前导零、符号或其他值返回 `400 INVALID_REQUEST` 与对应 field detail。窗口使用注入 clock 的 UTC 当日作为 endDate，包含首尾共 7/30 日；订单 count 和 gross 按 createdAt UTC 日期归属，完成退款按 decidedAt UTC 日期归属，缺失日期补零。每日 net 使用 checked subtraction 并允许合法负值，例如窗口外订单在窗口内完成退款。

看板 repository 在显式只读事务中批量读取并校验订单、明细、退款和 paymentStatus 映射，同一响应来自单一 SQLite snapshot，不执行逐订单或逐退款查询。三个 endpoint 均拒绝 `HEAD`，支持与其他已注册 API 相同的 CORS、request ID、取消、`503 SERVICE_UNAVAILABLE` 和安全 `500 INTERNAL_ERROR` 语义。

## 浏览器 CORS

可选配置 `CORS_ALLOWED_ORIGIN` 启用单一可信跨源 origin；未设置时不添加 CORS 行为。配置值必须是仅含 scheme、host 和可选 port 的绝对 `http`/`https` origin，不接受通配符、多个 origin、userinfo、path、query 或 fragment。

匹配 origin 的实际 API 请求返回精确 `Access-Control-Allow-Origin`、`Vary: Origin` 和 `Access-Control-Expose-Headers: X-Request-ID`，不启用 credentials。`Origin` 和 `Access-Control-Request-Method` 必须各自恰好出现一次，重复值会被拒绝。预检位于认证之前，仅对已注册 route 和 `GET`、`POST`、`PATCH`、`OPTIONS` 目标 method 通过，返回 `204`、允许请求头 `Authorization, Content-Type, Idempotency-Key, X-Request-ID` 和 `Access-Control-Max-Age: 600`；成功和拒绝预检都保留 `Vary: Origin, Access-Control-Request-Method, Access-Control-Request-Headers`。非法或不匹配的实际 origin 返回 `403 CORS_ORIGIN_DENIED`，预检返回 `403 CORS_PREFLIGHT_DENIED`；未知 path 保持普通 `404` 且不返回 CORS allow header。CORS 拒绝发生在认证之前，不替代业务认证、授权或状态校验。

## 运行错误与 Request ID

所有响应回写 `X-Request-ID`。入站值只接受 `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`；空值或非法值替换为 `req_` 加 32 位小写十六进制值。错误响应、header 和访问日志使用同一 request ID。

错误结构为 `{"error":{"code","message","requestId","details?"}}`。当前已实现 `NOT_READY`、`METHOD_NOT_ALLOWED`、`INTERNAL_ERROR`、认证错误、CORS 的 `CORS_ORIGIN_DENIED`/`CORS_PREFLIGHT_DENIED`，以及订单、退款和看板使用的 `INVALID_REQUEST`、`FORBIDDEN`、`NOT_FOUND`、`VERSION_CONFLICT`、`STATE_CONFLICT`、`IDEMPOTENCY_CONFLICT`、`UNSUPPORTED_MEDIA_TYPE`、`VALIDATION_FAILED` 和 `SERVICE_UNAVAILABLE`。panic 发生在响应提交前时返回完整 `500 INTERNAL_ERROR`；提交后不重写已发送响应，只记录 panic 和最终可观察状态。

## 当前未实现

- `/api/v1/*` 页面和附件 API。

新增 endpoint 只有在实现、测试和对应门禁齐全后才能写入本文。实现前的设计调整只修改 [目标 HTTP API](./03-http-api-target.md)。
