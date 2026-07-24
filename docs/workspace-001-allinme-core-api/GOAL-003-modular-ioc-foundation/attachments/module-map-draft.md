---
title: 模块图（GOAL-003）
status: active
created: 2026-07-24
updated: 2026-07-24
parent: GOAL-003-modular-ioc-foundation
version: 0.2.0
---

# 模块图（与仓库对齐）

```text
                    ┌─────────────────┐
                    │   cmd/server    │  入口；调用 app.New
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  internal/app   │  composition root（唯一 New 具体实现）
                    └────────┬────────┘
           ┌─────────────────┼─────────────────┐
           │                 │                 │
   ┌───────▼──────┐  ┌───────▼──────┐  ┌───────▼──────┐
   │   handler    │  │ service/meta │  │   config     │
   └───────┬──────┘  └───────┬──────┘  └──────────────┘
           │                 │
           │         ┌───────▼──────┐
           │         │  port.MetaStore │
           │         └───────▲──────┘
           │                 │
           │    ┌────────────┴────────────┐
           │    │ repository/sqlite       │
           │    │ repository/memory(fake) │
           │    └─────────────────────────┘
           ▼
        net/http
```

**GOAL-002 扩展点**（空目录占位）：`internal/service/{auth,order,wallet,notification,schemaui}` — 仍只在 `app` 组装。

详见 [modular-ioc.md](../../../architecture/modular-ioc.md)。
