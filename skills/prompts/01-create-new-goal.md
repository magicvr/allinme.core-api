---
title: 提示词 · 创建新目标
status: active
created: 2026-07-18
updated: 2026-07-20
parent: null
version: 0.4.0
role: primitive
---

# 01 · 创建新目标（原语 / primitive）

## 说明

供 [00-govern-orchestrator.md](00-govern-orchestrator.md) 在用户确认「需要新建目标」后调用；也可高级直调。  
日常默认路径请用编排器。

交付物：正确编号的目标文件夹 + 完整五件套 + 已同步的 `goal-tree.md`。

---

## 提示词正文

```markdown
# 角色

你是本项目的目标治理协作者。遵守 `AGENTS.md` 和/或 `.github/copilot-instructions.md`。  
P-001（大目标先路线图）与 P-005（信息就绪与未知项门禁）以 AGENTS 为准；若存在 architecture 原则文档，可作补充。

# 任务

按已确认信息创建一个新目标：五件套齐全，goal-tree 已更新。

# 用户输入（缺项先问清再写）

- 目标标题：【用户语言】
- 英文短 slug（小写、短横线）：【如 improve-auth】
- 父目标完整 ID：【如 GOAL-001-my-root-slug；Root 则 null】
- 工作区上下文：【当前 `docs/workspace-<NNN>-<slug>/workspace.md` 的 id / root_goal；若仅有 legacy `docs/goals/` 则注明“隐式单工作区”】
- 一句话概述：
- 初始成功标准（2～5 条可验证项；尚受信息项影响的可标“暂定”）：
- 是否需要高层路线图（范围大/步骤不明）？【是 / 否】
  - 若是：本回合在 00-meta 或 01-decision 写阶段与先后；子目标留待后续阶段
- 已识别的信息需求 / 假设：【I-00N、`required`/`non-blocking`、问题、影响门禁、最晚需要阶段、验证/收集动作；`deferred` 另给理由、责任人和复核触发；无则明确“当前未识别”】
- 是否存在到期 required 信息门禁？【是 / 否；若是，本目标只能先执行澄清/收集或有界实验，不得伪造完整方案】
- 初始状态：draft 或 active（默认 draft）
- 今日日期：【会话/系统 YYYY-MM-DD】

# 步骤

1. 先定位当前工作区 `workspace.md` 和其 `goal-tree.md`：校验 `root_goal` 指向 `parent: null` 的 Root Goal、`canonical_scope` 覆盖当前工作区根。没有显式工作区根时只可处理 legacy `docs/goals/` 隐式单工作区；不得猜测外部工作区。
2. 新编号 = 当前工作区目标树最大编号 + 1（三位）。Root 固定为 GOAL-001。
3. 创建 `<workspace-root>/GOAL-NNN-<slug>/`（与现有目标平铺，层级只写在 parent）。不得把目标创建到其他工作区、共享资料目录或目录嵌套中。
4. 一次写入五件套：`00-meta` / `01-decision` / `02-execution` / `03-audit` / `attachments/`。
5. 优先定位项目的核心模板层 `docs/templates/goal-folder/`；若目标仓库没有独立核心层，再定位 **SKILLS_PKG**（含 `prompts/01-create-new-goal.md` 或 `templates/goal-folder/` 的目录）并参考包内镜像 `templates/goal-folder/`。两者结构必须一致。
6. Frontmatter 至少：status, created, updated, parent, version；meta 另含 id、title（建议 progress）。
   - Root 的 slug 使用用户确认的名称。
7. 正文：
   - meta：概述、成功标准、parent 链接；需要时含路线图与信息就绪概览
   - decision：已有取舍则写「决定 + 为什么」；信息需求、阶段门禁、残余风险接受也在此记录；否则「待立项后补充」
   - execution：只记已发生事实（如「今日创建目标」）
   - audit：可写「尚未到达复盘节点」
8. 更新 `goal-tree.md` 的 ASCII 树与状态表。
9. 如需，在父目标文档轻量提及新子目标；progress 与事实一致。

# 完成标准

- [ ] 编号无冲突；id = 文件夹名  
- [ ] 五件套齐全；parent 为完整 id 或 null  
- [ ] goal-tree 树与表已更新  
- [ ] 若存在工作区上下文，Root Goal / canonical 范围已校验且新目标未越界
- [ ] 大目标已写路线图（若适用）  
- [ ] 已识别信息项已登记；到期 required 项没有被伪装成已验证或可直接实施
- [ ] 内容真实，无虚构完成项  
```

---

## 使用注意事项

- 缺信息时先确认并登记：目标可带未知立项，但要写明影响门禁与最晚需要阶段；只有信息工作有独立范围或证据时才拆出子目标。
- 创建后建议用 03 追加一条「目标已创建」执行记录。
- Root：`GOAL-001` + `parent: null`；slug 由用户定。
