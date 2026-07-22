---
title: /update-execution · 更新执行进度（Copilot wrapper · advanced primitive）
description: 原语入口。默认请用 /govern。按 03-update-execution 更新进度。
status: active
created: 2026-07-18
updated: 2026-07-19
parent: null
version: 0.3.0
slash: /update-execution
role: advanced
---

<!--
  ADVANCED primitive wrapper — default entry is /govern.
  Core: skills/prompts/03-update-execution.md
-->

# /update-execution · 更新执行进度（advanced / 原语）

> **默认请使用 `/govern`。** 本命令仅在你已明确只要「更新执行」原子操作时使用。

你是本项目的目标治理协作者。遵守项目 AI 规则（根 `AGENTS.md` 和/或 `.github/copilot-instructions.md`），包括 P-005 信息就绪与未知项门禁。

---

## 第一步：收集参数（先分析，再补问）

**不要**一上来甩出完整参数表。按 A → B → C 顺序推进。

### A. 先分析上下文（必做，再问用户）

1. **优先读取** `<workspace-root>/goal-tree.md`：哪些目标 active / blocked，当前 progress。
2. **必要时**读取候选目标的：
   - `00-meta.md`（成功标准、progress、status）
   - `01-decision.md`（I-00N 信息项、级别、阶段门禁与残余风险）
   - `02-execution.md`（最近时间线条目，避免重复空话、保持风格）
3. 解析用户在 `/update-execution` 后已写的「今天做了什么」：提取**事实条目**与可能路径/产物名。
4. **尝试推断**：

| 可推断项 | 推断方式 |
|----------|----------|
| 目标 ID | 用户点名 > 工作内容明显只属于某一 active 目标 > 仅一个 active 工作目标时建议该目标 |
| 今日日期 | 会话/系统当前日期 `YYYY-MM-DD` |
| 当前 progress / status | 来自 meta 与 goal-tree；默认「保持」 |
| 阻塞 | 用户未提则默认「无」 |
| 下一步计划 | 用户未提则可不写；若写须标明为计划 |
| 进度是否上调 | 仅当事实明显覆盖更多成功标准时**建议**新百分比，须用户确认后再改 |

若工作可能跨多个目标，先列出归属建议，请用户确认记到哪一个（可拆成多次更新）。

### B. 向用户汇报推断结果

例如：

> **当前项目情况**：GOAL-003 active 70%、GOAL-004 active 0%……  
> **我推断**：记入 `GOAL-00X-…`；日期 `YYYY-MM-DD`；progress/status 默认保持（当前 N% / active）。  
> **从你描述提取的事实**：1）… 2）…  
> **仍需你确认 / 补充**：……（例如是否调整进度、是否有阻塞）

### C. 只问真正缺失或需确认的项

- 用户已写清的事实列表不要再整表复问；可结构化复述后请确认「有无遗漏/纠错」。
- **progress / status 默认保持**；仅在你建议调整或用户要求时确认新值与依据。
- **避免**一次性甩全表；核心是「目标 + 今日事实」。

**参数优先级（仅补缺）：**

| 优先级 | 参数 | 说明 |
|--------|------|------|
| 必须确认 | 目标 ID / 路径 | 给出 active 候选 |
| 必须确认 | 本次实际完成的工作 | 条目列表；尽量带路径/产物；必须是事实 |
| 可默认 | 今日日期 | 已推断则默认 |
| 可默认 | 阻塞 / 风险 | 默认「无」 |
| 可选 | 关联 I-00N | 收集、验证、新发现或状态变化；没有证据不得标 `verified` |
| 可选 | 下一步计划 | 标明为计划，非已完成 |
| 可默认 | 进度百分比 | 默认保持；若调整须给依据（对照成功标准） |
| 可默认 | status | 默认保持；取值 `draft` \| `active` \| `blocked` \| `done` \| `cancelled` |

对**无法确定且会影响写入**的字段，**不要猜测后继续**。绝不把未做的工作写成完成。

---

## 第二步：执行核心提示词

参数齐备后，**完整阅读并严格执行**核心提示词：

- 路径：定位 skills 包根（含 `prompts/03-update-execution.md`，名可能不是 `skills`）后读该文件  
  （参考：[03-update-execution.md](../../../prompts/03-update-execution.md)）
- 使用其中「提示词正文」的时间线格式、强制步骤、禁止项与交付检查清单
- 将第一步已确认的参数填入核心提示词的「用户输入」槽位后执行

---

## 必须遵守的 AGENTS 要点

1. **只记事实**：`02-execution.md` 时间线只追加**已发生**工作；禁止编造未完成、未提交的内容。
2. **具体可验证**：尽量写路径、产物名、可勾选结果；避免「优化了体验」类空话。
3. **计划与事实分离**：下一步必须标明为计划，不得标成已完成。
4. **同步 meta + goal-tree**：调整 progress / status 时，改 `00-meta.md`，并**必须**同步 `<workspace-root>/goal-tree.md`（树 + 表）。
5. **成功标准一致**：勾选 meta 中成功标准时，须与 execution 事实一致。
6. **与 decision 分工**：取舍论证写入 `01-decision.md`，不要塞进 execution。
7. **扁平存储与编号**：不嵌套目标文件夹；不擅自改 Root / parent 却不更新 goal-tree。
8. **归属正确**：工作属于另一目标时改记正确 ID，勿堆错目标。
9. **P-005 信息事实**：记录收集/验证的实际动作和证据；新未知须登记级别与最晚阶段。到期 `required` 项未处理时不得以进度更新掩盖受影响门禁。

---

## 完成后

用核心提示词中的**交付检查清单**逐条自检，并简短汇报：

- 新增时间线条目摘要
- progress / status 是否变更，是否与 meta、goal-tree 一致
- 检查清单勾选结果
