---
title: 提示词 · 记录决策
status: active
created: 2026-07-18
updated: 2026-07-20
parent: null
version: 0.4.0
role: primitive
---

# 02 · 记录决策（原语 / primitive）

## 说明

**角色**：文档原语，供 [00-govern-orchestrator.md](00-govern-orchestrator.md) 调用；也可高级直调。默认用户路径请用编排器。

解决「做了取舍却没写清楚、或写了一堆空话」的问题。  
引导 AI 在目标的 `01-decision.md` 中追加结构化决策条目：决定了什么、为什么、未选方案是什么。

---

## 提示词正文

```markdown
# 角色
你是本项目的目标治理协作者。遵守 `AGENTS.md` 和/或 `.github/copilot-instructions.md`。P-001 与 P-005 以 AGENTS 为准。

# 任务
为指定目标在 `01-decision.md` 追加真实决策：写清「决定了什么」与「为什么」。

# 用户输入（缺项先确认）
- 目标 ID / 路径：
- 工作区上下文（若存在）：【当前 `docs/workspace-<NNN>-<slug>/workspace.md` 的 id / root_goal；只有 legacy `docs/goals/` 时才写“隐式单工作区”】
- 决策标题：
- 决定了什么：
- 为什么（背景、约束、收益）：
- 未选方案（建议有）：【方案 + 简短理由】
- 影响范围（可选）：
- 后续动作（可选）：
- 关联信息项 / 门禁（可选）：【I-00N；`required`/`non-blocking`、新建、更新、验证、延期、接受残余风险或关闭】
- 共享资料引用（可选）：【`reference_id`、`material_id`、`source`、`version`、`sha256`；资料内容是否仅为候选】
- 今日日期：【YYYY-MM-DD】

# 步骤
1. 读当前 `docs/workspace-<NNN>-<slug>/workspace.md`、其 `goal-tree.md`、`00-meta.md` 与现有 `01-decision.md`。若 workspace Root Goal/canonical 范围与目标不匹配，停止受影响决策，先记录或请求修复上下文；没有显式工作区根时只处理 legacy 隐式单工作区。
2. 追加条目（D-001 起递增）：

   ### D-NNN · <决策标题>
   - **日期** / **状态**（accepted | proposed | superseded）
   - **决定** / **理由** / **未选方案** / **影响** / **后续**

3. 刷新 `updated`；小改可保持 version。
4. 若使用共享资料引用，先核对 `workspace_id` 匹配、资料目录不是 `none`、`material_id`/`source`/`version`/有效 `sha256` 齐全。任何缺失或不匹配的引用不得作为事实、证据、跨工作区上下文或 finding 关闭依据；资料内容仍须按来源与用户确认规则标为候选。
5. 若决策改变范围、成功标准、路线图或信息门禁：同步 `00-meta` / 信息需求表，并在 `02-execution` 记一句「记录决策 D-NNN：…」。`deferred` 必须写清理由、责任人和复核触发；残余风险接受必须写清范围、期限、缓解/监控与复审触发，且不得把状态改写为 `verified`。
6. 若 status/progress 变化：同步 `goal-tree.md`。
7. 过程流水账写在 execution；decision 保持可执行结论。

# 完成标准
- [ ] 条目含决定 + 理由；重要取舍含未选方案  
- [ ] 编号连续；updated 已刷新  
- [ ] meta / execution / goal-tree 在需要时已对齐  
- [ ] 工作区绑定已校验；共享资料引用（若有）已固定版本/哈希且未越界
- [ ] 若涉及 I-00N，信息表状态、受影响门禁和残余风险留痕已对齐
- [ ] 不确定处标「待确认」；内容为真实取舍  
```

---

## 使用注意事项

- 一条提示词可记多条决策，但请在输入区逐条列清，避免 AI 合并成含糊一段。
- 若决策已过时，用新条目 `superseded` 旧决策，并在旧条目状态改为 `superseded`，不要静默删除历史。
