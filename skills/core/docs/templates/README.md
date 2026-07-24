---
title: 核心目标文档模板（消费方）
status: active
created: 2026-07-19
updated: 2026-07-24
parent: null
version: 0.1.0
---

# 核心目标文档模板

本目录是 Goal Governance 在**本仓库**的模板层：创建新目标与工作区上下文时从此复制。它属于核心方法论，不依赖特定 AI 宿主。

## 目录

`goal-folder/` 包含目标完整五件套：

- `00-meta.md` — 元信息、成功标准、父子关系与信息就绪概览  
- `01-decision.md` — 决定、理由、未选方案与信息需求  
- `02-execution.md` — 可核对事实时间线  
- `03-audit.md` — 复盘与 `self` / `independent` 审计意见  
- `attachments/` — 附件目录（可空）

`workspace-context.md` 复制为 `docs/workspace-<NNN>-<slug>/workspace.md`，绑定 Root Goal、canonical 范围与共享资料引用。

## 使用边界

- 新目标实例写在当前工作区根 `docs/workspace-<NNN>-<slug>/`，遵守根目录 `AGENTS.md` 与该工作区 `goal-tree.md`。  
- 本目录不是运行中的目标记录。  
- 共享资料规则见 [architecture/workspace-protocol.md](../architecture/workspace-protocol.md)。  
- P-005：目标可带未知立项；信息需求表用于登记级别、门禁与证据。  
- 若同时存在 `skills/templates/`，内容应与本目录一致；**以本目录为创建时的首选路径**。
