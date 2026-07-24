---
title: 模块化与 IoC 组装约定
status: active
created: 2026-07-24
updated: 2026-07-24
parent: null
version: 0.1.0
---

# 模块化与 IoC 组装约定

权威原则：工作区 Root **D-008（P-M1～P-M8）**；骨架目标 [GOAL-003](../workspace-001-allinme-core-api/GOAL-003-modular-ioc-foundation/00-meta.md)。

## 依赖方向

```text
cmd/server → internal/app (composition root)
                ├── handler  → service → port
                └── repository/sqlite  ──┘（实现 port）
```

- **禁止**：`service` import `repository/sqlite`；`handler` 打开 DB；业务包互相 `New` 具体实现。
- **允许**：`app` 包构造 `sqlite.*` 与 `service.*` 并注入接口。

## 如何新增一个业务模块（BC）

1. 在 `internal/port`（或 `internal/<bc>/port`）定义接口。
2. 在 `internal/service/<bc>` 实现应用服务，构造函数只收接口。
3. 在 `internal/repository/sqlite` 增加表与实现。
4. 在 `internal/handler` 增加 HTTP 适配（依赖 service）。
5. **仅在** `internal/app.New` 中接线 `New*`。
6. 空占位目录已预留：`auth` / `order` / `wallet` / `notification` / `schemaui`（可先只有 `.gitkeep`）。

## 如何换 Repository 实现

1. 新包实现同一 `port` 接口（例如 `internal/repository/postgres`）。
2. 在 `config` 增加驱动分支（`DB_DRIVER`）。
3. 仅改 `internal/app.New` 的构造选择；**不改** service / handler 代码。

## 垂直切片（当前）

- 端口：`port.MetaStore`
- SQLite：`repository/sqlite`（驱动 **modernc.org/sqlite**，纯 Go）
- 测试 double：`repository/memory`
- 就绪：`GET /readyz` 经 `meta.Service.Ready` → `MetaStore.Ping`
