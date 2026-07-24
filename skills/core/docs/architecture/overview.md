---
title: 架构概览（消费方）
status: active
created: 2026-07-18
updated: 2026-07-24
parent: null
version: 0.1.0
---

# 架构概览

## 目标

用「**核心协议为规范、目标文档为真相、Skills 为 AI 消费适配器**」支撑目标治理闭环：

```text
目标 (Goal)
  ├── 决策 (Decision)
  ├── 执行 (Execution)
  └── 审计 (Audit)
```

## 逻辑架构

```text
┌──────────────────────────────────────────────────────┐
│ 核心方法论与文档协议                                  │
│ docs/README.md + docs/architecture/ + docs/templates/ │
└───────────────────────┬──────────────────────────────┘
                        │ 规范结构与生命周期
                        ▼
              ┌──────────────────┐
              │ Skills / 提示词   │
              │ AI/Agent 适配器   │
              └─────────┬────────┘
                        │ 读写（经用户确认）
                        ▼
             ┌──────────────────────┐
             │ workspace-<NNN>-slug/ │  ← 工作区运行时真相源
             │ workspace.md + goal-tree.md + GOAL-* │
             └──────────────────────┘
```

`workspace.md` 绑定当前工作区的 Root Goal、canonical 范围和共享资料固定引用；**不**保存目标生命周期状态。目标状态只在该工作区根内的 `goal-tree.md` 与 `GOAL-*` 五件套中。

可选的人类 Web 工作台或其他产品面，若存在，同样只消费上述真相源，不得另立状态库。

## 仓库布局（消费方）

| 路径 | 职责 |
|------|------|
| `AGENTS.md` | AI 强制操作细则 |
| `docs/architecture/` | 治理原则与工作区协议（本目录） |
| `docs/templates/` | 五件套与工作区上下文模板 |
| `docs/workspace-<NNN>-<slug>/` | 当前工作区目标与过程记录（扁平） |
| `docs/shared-materials/` | 可选；工作区外资料候选库存 |
| `skills/` | AI 适配器、prompts、contracts、install |

## 原则与协议入口

- [principles.md](principles.md) — P-001～P-005  
- [workspace-protocol.md](workspace-protocol.md) — 工作区与共享资料  
- [directory-layout.md](directory-layout.md) — 最小目录树  

## 产品边界

- **核心方法论与 Skills 同级必备**；仅装适配器、无 `docs/architecture` 为不完整安装。  
- 目标过程树由各项目自行建立，不随 Skills 包复制。  
- 实现技术栈（语言、框架）由各项目自定，不在本包强制。
