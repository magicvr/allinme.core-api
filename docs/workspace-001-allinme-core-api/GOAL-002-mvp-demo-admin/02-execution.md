---
id: GOAL-002-mvp-demo-admin
doc: execution
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-25
version: 0.12.0
---

# 执行记录 · GOAL-002

## 时间线

### 2026-07-23 · 目标立项 / 策略 A 阻塞

- 立项；批量走协议演进；blocked 至新协议。

### 2026-07-24 · 协议钉死、方案冻结、审计响应、I-009 关闭

- 2.4.1 钉死；I-002～I-007 decided；A-001 响应；GOAL-003 done；I-009 verified。

### 2026-07-24 · M2 鉴权 JWT + RBAC + 菜单

**代码（composition root 接线，service 只依赖 port）**：

| 路径 | 说明 |
|------|------|
| `internal/domain/user.go` | User / PublicUser / 角色判断 |
| `internal/port/user.go`、`security.go` | UserRepository、PasswordHasher、TokenService |
| `internal/security/bcrypt.go`、`jwt.go` | bcrypt + HS256 JWT（TTL 可配，默认 1h） |
| `internal/repository/sqlite/user.go`、`seed.go` | users 表 + 空库 seed |
| `internal/service/auth` | Login / Me / ParseToken |
| `internal/service/menu` | 静态菜单目录按角色过滤 |
| `internal/middleware/auth.go` | Bearer 校验注入 context |
| `internal/handler/auth.go`、`menu.go` | `POST /v1/auth/login`、`GET /v1/auth/me`、`GET /v1/admin/menu` |
| `internal/app/app.go` | 组装 users/hasher/tokens/auth/menu + seed |
| `internal/config` | `JWT_SECRET`、`JWT_TTL` |

**API**：

- 公开：`POST /v1/auth/login`
- 需 Bearer：`GET /v1/auth/me`、`GET /v1/admin/menu`、`GET /v1/ping`（及后续业务）
- 仍公开：`/healthz`、`/readyz`

**种子用户**（密码均为 `Demo@1234`）：`admin` / `operator` / `viewer`。

**菜单 RBAC**：admin 含「用户」入口；operator/viewer 见仪表盘与三域列表入口，无 users。

**验证**：`go test ./...` pass（含 auth 集成：login→me→menu、无 token 401）。

progress → **40%**；M2 **完成**；**未**开始三域业务 API（M3）。

### 2026-07-25 · M3 订单 API 首切片

**实现事实**：

| 路径 | 说明 |
|------|------|
| `internal/domain/order.go` | Order 与 `pending`/`paid`/`cancelled`/`refunded` 领域状态定义；本 HTTP 切片仅开放 pending→paid/cancelled。 |
| `internal/port/order.go` | OrderRepository、列表筛选与稳定 not-found/orderNo/version/state/input 错误。 |
| `internal/service/order/service.go` | 可注入时钟/ID 的创建、列表、详情、pending-only 更新、CAS 状态动作与批量删除用例。 |
| `internal/repository/sqlite/order.go`、`db.go`、`seed.go` | `orders` schema、SQLite 查询/CAS/事务 batch-delete；幂等种子覆盖 pending/paid/cancelled/refunded。 |
| `internal/handler/order.go`、`handler.go` | `/v1/orders` HTTP 适配、1 MiB 限制与拒绝未知 JSON 字段、Bearer/RBAC 路由。 |
| `internal/app/app.go` | 唯一 composition root 注入 SQLite OrderRepository 与 Order service，并启动 seed。 |
| `internal/service/order/service_test.go`、`internal/repository/sqlite/order_test.go`、`internal/handler/order_test.go` | service、SQLite 与完整 HTTP 集成覆盖。 |

**首切片 API（均为 `/v1`）**：

- `GET /v1/orders`、`GET /v1/orders/{id}`：Bearer 下三角色可读；list 返回 `data.list` / `data.total`，支持 `status`、`q`、`page`、`pageSize`（默认 1/20，上限100）。
- `POST /v1/orders`、`PUT /v1/orders/{id}`、`POST /v1/orders/{id}/mark-paid`、`POST /v1/orders/{id}/cancel`、`POST /v1/orders/batch-delete`：仅 admin/operator；写操作遵守 D-018 的 version CAS 与状态机。
- batch-delete body 为 `{ "ids": [] }`，最多100，拒绝空/重复，SQLite transaction all-or-nothing，且仅 pending/cancelled 可删。

**验证事实（2026-07-25）**：已运行 `gofmt`；`go test -count=1 ./...` **pass**；`go vet ./...` **pass**；`git diff --check` **pass**。测试覆盖服务状态/版本与批量原子性、SQLite list/filter/pagination/unique/CAS/rollback/seed 幂等，以及认证、viewer 只读、operator/admin 写、envelope、create/get/update/actions/409/batch all-or-nothing HTTP 集成。

**复核修正事实（同日）**：订单 seed 改为在单一 SQLite transaction 内完成空表检查和四条种子插入，插入中途失败会回滚且可重试；列表在计算 offset 前拒绝 int 溢出，极大 page 返回 `bad_request` / HTTP 400；订单时间戳改为固定宽度 UTC 纳秒文本，保证 SQLite `TEXT` 时间排序。新增中途 seed 失败后 count=0/重试完整、极大分页 400、同秒 0ns/100ms 排序与文本格式测试。

progress → **50%**；M3 仍为进行中（订单首切片完成，钱包/通知待）。

### 2026-07-25 · GOAL-002 路线图渐进拆分

**治理事实**：

- 依据用户确认与 P-001，将 GOAL-002 从仅含内部 M0～M5 阶段调整为父目标 + 渐进子目标结构。
- 补录 [GOAL-004-auth-rbac-menu](../GOAL-004-auth-rbac-menu/00-meta.md)：承接 2026-07-24 已完成的 M2 鉴权/RBAC/菜单事实，`done` / 100%。
- 补录 [GOAL-005-order-api-first-slice](../GOAL-005-order-api-first-slice/00-meta.md)：承接 2026-07-25 已完成的订单首切片事实，`done` / 100%；明确不含 DELETE/refund。
- 创建 [GOAL-006-wallet-api](../GOAL-006-wallet-api/00-meta.md)：当前下一执行工作包，`active` / 0%；I-001 required 契约门禁 open，尚未实施代码。
- 通知、订单补齐、M4 和 M5 暂留在父目标路线图，进入阶段时再创建。
- 记录 D-019/D-020；父目标 progress 保持 **50%**，治理重排未被计为产品进度。

### 2026-07-25 · 响应 A-007 F-008：补齐订单稳定错误码测试

**修正事实**：

- `internal/handler/order.go` 将订单 handler 的 service 参数收敛为本地 `orderService` 接口，使 HTTP 适配层可注入失败 service 做错误边界测试；生产接线仍由 `internal/app/app.go` 注入原 `*order.Service`，业务行为未改。
- `internal/handler/order_test.go` 对 D-018 稳定错误码增加 HTTP 跨层断言：`bad_request`、`order_not_found`、`order_no_conflict`、`version_conflict`、`invalid_state`。
- 新增 `internal/handler/order_internal_test.go`：注入返回含敏感路径文本的失败 service，验证 HTTP 500 统一为 `code=internal`，响应不包含底层错误或敏感路径。
- 已运行 `gofmt`；`go test -count=1 ./...` **pass**；`go vet ./...` **pass**；`git diff --check` **pass**。

**治理边界**：本次关闭的是订单首切片错误契约的测试证据缺口，不新增产品范围、不实现 DELETE/refund，也不改变 GOAL-002 progress。GOAL-005 的 `done` / 100% 在关闭证据写入后保持不变；详见本目标 A-008 与 GOAL-005 A-002。

### 2026-07-25 · M3b 钱包 API 子目标完成

- [GOAL-006-wallet-api](../GOAL-006-wallet-api/00-meta.md) 完成 W0～W4，`done` / 100%。
- 钱包 domain/port/service、SQLite repository/事务 seed、composition root、七个 `/v1/wallets` API、Bearer/RBAC 与跨层测试全部落地。
- GOAL-006 A-002 execution-facts / close-out 自审 `pass`，无 required/recommended finding；I-001 verified，I-002 保持父目标 M4 non-blocking 范围。
- 强制 targeted/full test、vet、diff checks 通过；额外 race 因本机 Windows cgo 工具链失败未完成，已在子目标审计如实留痕。
- 父目标因钱包真实产品增量由 50% 调整为 **60%**；通知、订单 DELETE/refund、M4 Page Schema/I-010 和 M5 仍未完成。

## 待办

1. 按路线图创建并推进通知 API 子目标
2. 后续创建订单 DELETE/refund 补齐、M4 page schema + **I-010**、M5 验收

## 进度评估

**约 60%**：鉴权/菜单、订单首切片和钱包 API 子目标已完成；通知、订单 DELETE/refund、Page Schema/仪表盘/协议校验及最终集成验收仍待推进。I-010 继续只阻断 M4 校验宣称。
