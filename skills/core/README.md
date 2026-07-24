---
title: Skills 包内 Core 方法论镜像
status: active
created: 2026-07-24
updated: 2026-07-24
parent: null
version: 0.1.0
---

# skills/core · 消费方核心方法论镜像

本目录是 **GOAL-019 D-003 / D-004** 规定的、随 Skills 包分发的**核心方法论子集**。

| 包内路径 | `install` 默认落到消费仓 |
|----------|---------------------------|
| `docs/README.md` | `docs/README.md`（精简入口） |
| `docs/architecture/*.md` | `docs/architecture/` |
| `docs/templates/**` | `docs/templates/` |

## 包含

- `architecture/principles.md` — P-001～P-005
- `architecture/workspace-protocol.md` — 工作区与共享资料协议
- `architecture/overview.md` — 逻辑架构（消费方；无 monorepo dogfood）
- `architecture/directory-layout.md` — 消费方最小目录树
- `templates/` — 五件套 + `workspace-context.md`
- `docs/README.md` — 精简文档入口

## 不包含

- `tech-stack.md`（实现栈）
- monorepo dogfood 目标树、`web/`、`artifacts/`
- 维护者-only 的 standalone 测试与 releases 长文

## 与 monorepo canonical

上游规范在仓库 `docs/architecture/`、`docs/templates/`、`docs/README.md`。本镜像为消费分发稿；语义变更应先改 canonical，再同步本目录并跑 `skills/tests`。

**缺 core = 不完整安装**（与 Skills 同级必备）。
