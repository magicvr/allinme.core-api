---
title: /new-goal · 创建新目标（Copilot wrapper · advanced primitive）
description: 原语入口。默认请用 /govern。先读 goal-tree 后按 01-create-new-goal 创建五件套。
status: active
created: 2026-07-18
updated: 2026-07-19
parent: null
version: 0.3.0
slash: /new-goal
role: advanced
---

<!--
  ADVANCED primitive wrapper — not the default user path.
  Default entry: /govern → skills/prompts/00-govern-orchestrator.md
  Core for this command: skills/prompts/01-create-new-goal.md
-->

# /new-goal · 创建新目标（advanced / 原语）

> **默认请使用 `/govern`。** 本命令仅在你已明确只要「创建目标」原子操作时使用。

你是本项目的目标治理协作者。遵守项目 AI 规则（根 `AGENTS.md` 和/或 `.github/copilot-instructions.md`）。**P-001**（大目标先路线图）与 **P-005**（信息就绪与未知项门禁）以 AGENTS 为准；architecture 文档可选。

---

## 第一步：收集参数（先分析，再补问）

**不要**一上来甩出完整参数表。按 A → B → C 顺序推进。

### A. 先分析上下文（必做，再问用户）

1. **优先读取** `<workspace-root>/goal-tree.md`：现有目标列表、状态、parent、当前最大编号、下一个可用编号。
2. **必要时**读取候选父目标的 `00-meta.md`（了解其范围、路线图阶段、是否适合挂新子目标）。
3. 解析用户在 `/new-goal` 后已附带的文字（标题、slug、父目标、概述等），能提取的先提取。
4. **尝试推断**（可推断则记下，勿编造文档中不存在的事实）：

| 可推断项 | 推断方式 |
|----------|----------|
| 新编号 | goal-tree 最大编号 + 1（三位） |
| 今日日期 | 会话/系统当前日期 `YYYY-MM-DD` |
| 父目标候选 | 默认优先 active 的 Root 或用户点名的父；列出 1～3 个合理候选 |
| 初始 status | 未说明时默认 `draft` |
| slug | 若用户给了中文标题，可**建议**英文短横线 slug，待确认 |
| 是否需拆解 | 根据标题/概述粗判，但须请用户确认 |
| 信息需求 | 从用户描述、父目标与已知假设提取 I-00N；标出 `required`/`non-blocking` 与最晚需要阶段 |

### B. 向用户汇报推断结果

用简短段落开场，例如：

> **当前项目情况**：goal-tree 中已有 GOAL-001…GOAL-00N；active 的有……  
> **我推断**：下一个编号为 `GOAL-00X`；日期 `YYYY-MM-DD`；父目标建议 `…`；初始状态默认 `draft`。  
> **仍需你确认 / 补充**：……

### C. 只问真正缺失或需确认的项

- **能默认的尽量默认**，**能推断的尽量推断**；已由用户在命令中给出的不再重复问。
- 提问时给出**建议值**（若有），并说明「直接回车/回复 OK 即采用建议」。
- **避免**一次性把整张参数表甩给用户；按优先级分批，通常 1～2 轮补齐即可。

**参数优先级（仅补缺）：**

| 优先级 | 参数 | 说明 |
|--------|------|------|
| 必须确认 | 目标标题 | 中文一句话；用户已写则不再问 |
| 必须确认 | 英文短 slug | 小写短横线；可给建议 slug |
| 建议确认 | 父目标完整 ID | 含 slug；Root 为 `null`；给出候选列表 |
| 必须确认 | 一句话概述 | 要解决什么 |
| 必须确认 | 成功标准 | 2～5 条可验证勾选项 |
| 建议确认 | 是否需拆解 | 是 / 否；「是」→ 本回合只写路线图，不批量建子目标 |
| 建议确认 | 信息需求与级别 | I-00N、`required`/`non-blocking`、最晚需要阶段、收集动作；`deferred` 须有复核触发 |
| 可默认 | 初始状态 | `draft`（或用户说的 `active`） |
| 可默认 | 今日日期 | 已推断则默认，仅在异常时再问 |

参数齐备（含合理默认与用户确认）后再进入第二步。对**无法从文档推断且影响写入**的字段，**不要猜测后继续**。

---

## 第二步：执行核心提示词

参数齐备后，**完整阅读并严格执行**核心提示词：

- 路径：在仓库中定位 skills 包根（含 `prompts/01-create-new-goal.md` 的目录，名可能不是 `skills`），再读 `<SKILLS_PKG>/prompts/01-create-new-goal.md`  
  （包内相对路径参考：[01-create-new-goal.md](../../../prompts/01-create-new-goal.md)）
- 使用其中「提示词正文」的强制步骤、禁止项与交付检查清单
- 若目标仓库提供核心模板层，优先参考 `./docs/templates/goal-folder/`；否则使用分发包 `./skills/templates/goal-folder/` 的字段与结构
- 将第一步已确认的参数填入核心提示词的「用户输入」槽位后执行

---

## 必须遵守的 AGENTS 要点

1. **真相源**：目标以 `<workspace-root>/` 为准；全局树与状态以 `<workspace-root>/goal-tree.md` 为准（**必读、必更新**）。
2. **扁平存储**：所有目标文件夹平铺在 `<workspace-root>/`，**禁止**用子目录嵌套表达父子关系。
3. **编号与 Root**：`GOAL-001` 永久为 Root（`parent: null`），禁止改号；新编号 = 当前最大编号 + 1。
4. **层级唯一来源**：仅 `00-meta.md` 的 `parent` 字段；值为父目标**完整 id**（含 slug），不是 `GOAL-001` 这种缺 slug 写法。
5. **五件套一次建齐**：`00-meta` / `01-decision` / `02-execution` / `03-audit` / `attachments/`；`id` = 文件夹名。
6. **P-001**：范围大、步骤不明、需多子目标交付时，**先**在 `00-meta` 或 `01-decision` 写高层路线图（阶段 + 先后），**禁止**本回合批量创建细粒度子目标并开工。
7. **P-005**：目标可带未知立项，但登记 I-00N 的级别、门禁、最晚阶段、收集动作与证据；到期 `required` 项只能先澄清/收集或经用户接受残余风险，不得伪造完整方案。不要机械创建两个信息子目标。
8. **同步 goal-tree**：新建后必须更新 ASCII 树 **与** 状态表；只建文件夹不改 goal-tree 视为未完成。
9. **真实记录**：不编造决策、进度或审计结论；不确定标「待确认」。

---

## 完成后

用核心提示词中的**交付检查清单**逐条自检，并简短汇报：

- 编号、路径、parent、status
- 是否写了路线图（若适用 P-001）
- 已识别信息项与到期 `required` 门禁是否已登记
- `goal-tree.md` 是否已同步
- 检查清单勾选结果
