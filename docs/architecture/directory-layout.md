---
title: 目录布局（消费方）
status: active
created: 2026-07-18
updated: 2026-07-24
parent: null
version: 0.1.0
---

# 目录布局

消费方在完整安装 Skills（含 core）并建立第一个工作区后的**最小**布局：

```text
your-project/
├── AGENTS.md                      # AI 强制规则（install 写入）
├── docs/
│   ├── README.md                  # 文档入口（core 精简版）
│   ├── architecture/
│   │   ├── overview.md
│   │   ├── principles.md          # P-001～P-005
│   │   ├── workspace-protocol.md
│   │   └── directory-layout.md    # 本文件
│   ├── templates/
│   │   ├── README.md
│   │   ├── goal-folder/           # 五件套模板
│   │   └── workspace-context.md
│   ├── shared-materials/          # 可选
│   └── workspace-001-<slug>/      # 显式工作区根（需创建）
│       ├── workspace.md
│       ├── goal-tree.md
│       ├── GOAL-001-<root-slug>/
│       └── GOAL-00N-.../
└── skills/                        # Skills 包（可改名）
    ├── prompts/
    ├── templates/                 # 包内分发镜像（可选副本）
    ├── contracts/
    ├── core/                      # 包内 core 源（install 已拷到 docs/）
    └── install.*
```

## 约束

- `docs/workspace-<NNN>-<slug>/GOAL-*` 之间**不得**再嵌套目标目录。  
- 新目标只新增当前工作区根内的同级文件夹，并改 `parent` + 该工作区 `goal-tree.md`。  
- `docs/templates/goal-folder/` 是本仓库创建目标时的模板源；`skills/templates/` 为包内镜像，不保存目标状态。  
- `workspace.md` 绑定一个 Root Goal 与该工作区根范围。没有显式工作区根但存在 `docs/goals/` 时，才按 **legacy** 隐式单工作区兼容。  
- 共享资料只以版本/哈希固定引用出现；不能成为第二套目标状态。  
- **不**要求存在 monorepo 维护者目录（如上游 `web/`、dogfood 过程树、`artifacts/`）。

## 与 Skills 包的关系

| 包内 | 安装后 |
|------|--------|
| `skills/core/docs/**` | → `docs/**`（默认 install） |
| `skills/prompts/**` | 留在 skills 目录（或 `--all` 同步） |
| `skills/contracts/**` | 留在 skills 目录 |
