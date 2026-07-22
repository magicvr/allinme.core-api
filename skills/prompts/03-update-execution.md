---
title: 提示词 · 更新执行进度
status: active
created: 2026-07-18
updated: 2026-07-20
parent: null
version: 0.4.0
role: primitive
---

# 03 · 更新执行进度（原语 / primitive）

## 说明

**角色**：文档原语，供 [00-govern-orchestrator.md](00-govern-orchestrator.md) 调用；也可高级直调。默认用户路径请用编排器。

解决「做了工作却不写、或写成虚假 100%」的问题。  
引导 AI 在 `02-execution.md` 按时间线追加**事实**，并在确有进展时同步 meta / goal-tree 的进度。

---

## 提示词正文

```markdown
# 角色
你是本项目的目标治理协作者。遵守 `AGENTS.md` 和/或 `.github/copilot-instructions.md`，包括 P-005 的信息就绪与未知项门禁。

# 任务
在 `02-execution.md` 按时间线追加**已发生事实**；确有进展时同步 progress / status 与 goal-tree。

# 用户输入（缺项先确认）
- 目标 ID / 路径：
- 工作区上下文（若存在）：【当前 `docs/workspace-<NNN>-<slug>/workspace.md` 的 id / root_goal；只有 legacy `docs/goals/` 时才写“隐式单工作区”】
- 今日日期：【YYYY-MM-DD】
- 本次实际完成（具体条目，含路径/产物更佳）：
  1. …
- 阻塞 / 风险：【或「无」】
- 关联信息项（可选）：【I-00N；收集 / 验证 / 新发现 / 状态变化】
- 共享资料引用（可选）：【`reference_id`、`material_id`、`source`、`version`、`sha256`；确认状态】
- 下一步计划（可选，标明为计划）：
- progress：【保持 / 调整为 N%，并给依据】
- status：【保持 / draft|active|blocked|done|cancelled】

# 步骤
1. 读当前 `docs/workspace-<NNN>-<slug>/workspace.md`、`00-meta.md`（含信息需求）、`01-decision.md`、`02-execution.md`、`goal-tree.md`。若 workspace Root Goal/canonical 范围与目标不匹配，停止受影响写入；没有显式工作区根时只处理 legacy 隐式单工作区。
2. 在时间线追加：

   ### YYYY-MM-DD · <短标题>
   - 事实（做了什么、改了哪些路径）
   - 阻塞（如有）
   - 下一步（计划单独标注）

3. 条目具体可核对；计划与已完成分开写。
4. 涉及共享资料时，先核对引用的 `workspace_id`、`material_id`、`source`、`version` 与有效 `sha256`；缺失或不匹配时记录拒绝/阻断事实，不能把资料内容写成 confirmed 事实、证据或跨工作区上下文。固定引用只说明来源，事实准入仍须用户显式确认。
5. 涉及 I-00N 时：记录实际收集/验证动作与证据路径；新发现的未知追加到信息表，并写明 `required`/`non-blocking`、影响门禁和最晚需要阶段。`deferred` 要保留理由、责任人与复核触发；没有证据时不得把状态改为 `verified`。
6. 刷新 `updated`。
7. 调整 progress/status 时：同步 meta 与 goal-tree（树 + 表）；关门前确认没有未处理的关门 required 信息项。
8. 完成某条成功标准时：勾选 meta 并在 execution 点明。
9. 决策论证写入 `01-decision`；execution 保持时间线事实。

# 进度说明（建议）
文末一两句说明百分比依据（对照成功标准）。

# 完成标准
- [ ] 新条目为可核对事实  
- [ ] updated 已刷新  
- [ ] progress/status 与 meta、goal-tree 一致（若有变更）  
- [ ] 工作区绑定已校验；共享资料引用（若有）固定且任何拒绝路径未产生写入
- [ ] 成功标准勾选与事实一致  
- [ ] I-00N 状态与可核对证据一致；新未知没有被遗漏或伪装为已知
```

---

## 使用注意事项

- 输入区尽量写「改了什么文件 / 达成什么可验证结果」，AI 才能避免空话。
- 小步提交时 progress 可保持不变，只追加时间线即可。
- 若工作实际属于另一目标，应改记到正确目标，勿堆在错误 ID 下。
