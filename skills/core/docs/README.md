---
title: 文档体系说明（消费方）
status: active
created: 2026-07-24
updated: 2026-07-24
parent: null
version: 0.1.0
---

# docs/ · 文档体系（消费方）

本目录承载 **Goal Governance** 在本仓库的核心规范与目标过程记录入口。  
**核心方法论**与 **Skills 适配器**同级必备：只有 `skills/` 而没有本目录下的 architecture / templates，视为**不完整安装**。

## 最小结构

```text
docs/
├── README.md                 # 本文件
├── architecture/
│   ├── overview.md
│   ├── principles.md         # P-001～P-005
│   ├── workspace-protocol.md
│   └── directory-layout.md
├── templates/
│   ├── README.md
│   ├── goal-folder/          # 目标五件套模板
│   └── workspace-context.md
└── workspace-<NNN>-<slug>/   # 工作区根（运行时真相；install 不自动创建）
    ├── workspace.md
    ├── goal-tree.md
    └── GOAL-*/               # 平铺目标
```

可选：`docs/shared-materials/`（共享资料候选库存，非目标状态）。

## 核心规则（摘要）

1. 目标平铺在当前 `docs/workspace-<NNN>-<slug>/`，**禁止**用嵌套文件夹表达父子关系。  
2. 每个工作区 `GOAL-001-*` 为 Root，`parent: null`。  
3. 层级只写在各目标 `00-meta.md` 的 `parent`。  
4. 变更 status / progress / parent / 新建后，必须更新该工作区 `goal-tree.md`。  
5. 每个目标五件套：`00-meta` / `01-decision` / `02-execution` / `03-audit` / `attachments/`。  
6. 治理原则全文见 [architecture/principles.md](architecture/principles.md)（P-001～P-005）。  
7. 工作区与共享资料边界见 [architecture/workspace-protocol.md](architecture/workspace-protocol.md)。  
8. 新建目标优先从 [templates/goal-folder/](templates/goal-folder/) 复制；工作区上下文从 [templates/workspace-context.md](templates/workspace-context.md) 复制。

## 与 Skills

| 组件 | 路径 | 作用 |
|------|------|------|
| AI 规则 | 根目录 `AGENTS.md` | 操作细则 |
| 编排 / 交叉审计 | `skills/prompts/`、宿主 `/govern` `/audit` | 消费适配器 |
| 机读契约 | `skills/contracts/` | 版本与兼容声明镜像 |

日常推进：安装完成后建立工作区，再调用 **`/govern`**。

## 不在本精简入口

维护者 monorepo 专有内容（dogfood 目标树、Web 实现、发版流水线、standalone 空仓测试脚本等）不在此复制。需要完整上游时，参阅 [goal-governance](https://github.com/magicvr/goal-governance) 仓库的 `docs/`。
