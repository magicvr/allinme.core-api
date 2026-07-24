---
title: 模块图草案（GOAL-003）
status: draft
created: 2026-07-24
updated: 2026-07-24
parent: GOAL-003-modular-ioc-foundation
version: 0.1.0
---

# 模块图草案

> 实施完成后将 `status` 改为 active，并与仓库实际目录对齐。

```text
                    ┌─────────────────┐
                    │   cmd/server    │  composition root
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  internal/app   │  NewApp: wire-up
                    └────────┬────────┘
           ┌─────────────────┼─────────────────┐
           │                 │                 │
   ┌───────▼──────┐  ┌───────▼──────┐  ┌───────▼──────┐
   │   handler    │  │   service    │  │   config     │
   └───────┬──────┘  └───────┬──────┘  └──────────────┘
           │                 │
           │         ┌───────▼──────┐
           │         │    port      │  interfaces only
           │         └───────▲──────┘
           │                 │
           │         ┌───────┴──────────┐
           │         │ repository/sqlite│  (future: postgres)
           │         └──────────────────┘
           ▼
        net/http
```

**依赖规则**：箭头表示「知道/依赖」；`repository` 不得 import `handler`；`service` 不得 import `repository/sqlite`。

**GOAL-002 扩展点**：在 `service`/`port`/`handler` 下增加 `auth`、`order`、`wallet`、`notification`、`schemaui` 子树，仍只在 `app` 组装。
