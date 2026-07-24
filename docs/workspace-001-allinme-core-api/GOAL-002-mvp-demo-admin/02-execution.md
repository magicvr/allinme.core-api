---
id: GOAL-002-mvp-demo-admin
doc: execution
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-24
version: 0.7.0
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

## 待办

1. **M3** 订单 / 钱包 / 通知 API + 种子业务数据
2. M4 page schema embed + **I-010** 校验路径
3. M5 验收

## 进度评估

**约 40%**：鉴权与菜单闭环可用；三域与 page schema 未做。
