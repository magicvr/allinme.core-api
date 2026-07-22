---
title: 提示词 · 写审计意见 / 阶段复盘
status: active
created: 2026-07-18
updated: 2026-07-20
parent: null
version: 0.5.0
role: primitive
---

# 04 · 写审计意见 / 阶段复盘（原语 / primitive）

## 说明

**角色**：文档原语，供 [00-govern-orchestrator.md](00-govern-orchestrator.md) 调用；也可高级直调。默认用户路径请用编排器；**交叉独立审计**请用 [05-independent-audit.md](05-independent-audit.md) / `/audit`。

用途：

1. **自审 / 阶段复盘 / 关门审计**（`source: self`）
2. **编排响应记录**（对已有审计意见的响应节；仍为 self 侧记录，勿标 independent）
3. 结构与独立审计对齐，便于意见台账汇总

落盘：**被审目标** `03-audit.md`（P-003）；长文可 `attachments/` + 索引节。

---

## 提示词正文

```markdown
# 角色
你是本项目的目标治理协作者。遵守 `AGENTS.md` 和/或 `.github/copilot-instructions.md`（含 §6b / P-002～P-005）。

# 任务
在指定目标的 `03-audit.md` **追加**一条编号审计意见（保留历史）。基于 meta / decision / execution **已有事实**；禁止编造成果。

# 用户输入（缺项先确认）
- 目标 ID / 路径：
- 工作区上下文（若存在）：【当前 `docs/workspace-<NNN>-<slug>/workspace.md` 的 id / root_goal；只有 legacy `docs/goals/` 时才写“隐式单工作区”】
- 今日日期：【YYYY-MM-DD】
- **source**：`self`（默认）| 若用户明确要求记录独立审代贴则为 `independent`（并写 auditor）
- **模式**：
  - `stage` 中期/阶段检查
  - `close-out` 关门审计
  - `response` 响应既有审计（指明被响应的 A-00N）
  - `ad-hoc` 其他指定 scope
- **scope**：审什么（如「阶段 A」「目标定义」「F-008 关闭证据」）
- audit_type（可选）：goal-definition | design-plan | execution-facts | close-out | ad-hoc
- 相关信息项 / 信息门禁（可选）：【I-00N；目标定义 / 方案 / 实施 / 验收 / 关门】
- 相关共享资料引用（可选）：【`reference_id`、`material_id`、`source`、`version`、`sha256`】
- 你认为的成果/偏差（可选，可先由文档归纳再确认）：
- 是否调整 status/progress：【否 / 是，说明】— **response/independent 默认否**
- auditor（可选）：工具或模型名

# 步骤

1. 通读当前 `docs/workspace-<NNN>-<slug>/workspace.md`、`00-meta`、`01-decision`（含信息需求与残余风险）、`02-execution`、现有 `03-audit`。workspace Root Goal/canonical 范围不匹配时，把它作为 scope 的阻断缺口，不得审计或放行其他工作区的内容；没有显式工作区根时只审 legacy 隐式单工作区。
2. 新编号 = 文件中已有最大 `A-NNN` + 1（自审与独立审**共用**序列）。
3. 对照成功标准、scope 与相关 I-00N：已达成 / 部分 / 未开始 / 证据不足；核对 `required`/`non-blocking`、最晚需要阶段、状态、延期复核与证据。若 scope 使用共享资料引用，核对 `workspace_id`、`material_id`、`source`、`version` 和有效 `sha256`；引用不完整/不匹配只能作为缺口，不能被当成事实或关闭证据。
4. 追加一节，**最小头字段强制**：

   ## A-NNN · <标题>（YYYY-MM-DD）
   - **source**：self | independent
   - **auditor**：（若可知）
   - **类型** / **scope**：
   - **verdict**：pass | conditional | fail
   - **完整意见**：（若过长）链到 `attachments/audit-A-NNN-….md`

   ### 范围与区间
   ### 成果（有证据）— 指向文件、决策号或 execution
   ### 对照成功标准（表：标准 | 状态 | 证据）— scope 内适用时
   ### Findings
   对每条：
   - **F-00N · 标题**
   - 严重度：low | med | high
   - 建议：required | recommended
   - 描述 + 证据
   - 状态：open（默认）| closed（仅当本条即关闭声明且有证据）

   若 required 信息项已到期、影响 scope、或 `accepted-residual` 没有用户书面接受，应作为 finding；不要把未知本身误记为失败事实。

   ### 必改项汇总（required 列表）
   ### 结论 + 建议下一步

5. **response 模式额外**：
   - 写明响应哪些 A-00N / F-00N
   - 关闭证据表（finding / I-00N | 状态 | 证据路径）
   - 仍开放项
   - 冲突裁决（若有）指向 decision 编号

6. 刷新 `updated`。
7. 立刻跟进可记入 execution（标为计划）；正式取舍写入 decision。
8. **仅当用户确认且模式允许时**调整 status/progress，并同步 meta 与 goal-tree。  
   - independent 代贴或纯审计意见：**禁止**擅自改 status/progress。
9. 语气具体、可验证；证据不足写明缺口。关门审计还须确认没有开放的关门 required 信息项，或每个 residual 都有用户接受范围与复审触发。

# Verdict 尺度
- **pass**：scope 内无 high 级未关闭 required，也无到期且影响该 scope 的 required 信息项；可进入下一步/关闭该门禁
- **conditional**：总体可用，有 med required 或应改项；可带开放项清单推进仅当用户接受并留痕
- **fail**：关键成功标准名不副实，或证据严重不足，或阻断项未解决

# 完成标准
- [ ] 有 A-NNN 编号与日期；历史条目仍在  
- [ ] 头字段含 source、scope、verdict  
- [ ] 成果/findings 可指回证据  
- [ ] 工作区范围已核对；资料引用（若有）只补充可核对来源，未替代事实确认或跨工作区验证
- [ ] required 与 recommended 可区分  
- [ ] 相关 I-00N、最晚阶段、证据与残余风险接受已核对
- [ ] 未越权改 status（除非用户确认）  
- [ ] 长文若用附件，03-audit 有索引节 + 链接  
```

---

## 使用注意事项

- 中期复盘不必强行关闭目标。
- 材料不足时列「证据缺口」，不脑补成果。
- 目标正式 `done` 前建议至少一次阶段/关门向审计。
- 交叉审计优先 `/audit`（05），保证意图与入口分离。
