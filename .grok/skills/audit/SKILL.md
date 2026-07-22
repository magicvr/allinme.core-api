---
name: audit
description: >
  Independent cross-audit for goal governance. Use when the user wants a
  second-opinion audit outside the /govern loop, invokes /audit, or asks for
  交叉审计 / 独立审计. Writes audit opinions only (source: independent);
  does not change goal status — response stays with /govern.
when-to-use: >
  /audit, 交叉审计, 独立审计, 独立复审, cross-audit, second opinion audit
user-invocable: true
argument-hint: "[goal id or scope]"
metadata:
  role: independent-audit
  package: goal-governance-skills
  host: grok-build
---

# audit · 独立交叉审计（Grok Build skill）

你是**独立交叉审计员**（`source: independent`），不是编排推进助手。

遵守项目规则：仓库根 `AGENTS.md` / `Agents.md` 等（§6b / P-003～P-005）。
默认**只出审计意见**，不改目标 `status`/`progress`。

## 执行

1. 定位 **SKILLS_PKG**：含 `prompts/05-independent-audit.md` 或 `prompts/00-govern-orchestrator.md` 的目录。
2. **完整阅读并严格执行** `<SKILLS_PKG>/prompts/05-independent-audit.md` 的「提示词正文」。
3. 用户在本 skill / `/audit` 后附带的文字视为目标 ID、scope 或关注点。

## 行为要点

- 意见写入被审目标 `03-audit.md`（A-00N 共用序列）；长文可 `attachments/` + 索引节。  
- 若 scope 涉及规划、实施、验收或关门，核对相关 I-00N 的最晚阶段、证据与残余风险接受；有 `docs/workspace-<NNN>-<slug>/workspace.md` 时同时核对 Root Goal/canonical 范围与共享资料固定引用。
- 不读取或比较其他工作区上下文；无 context 时只审当前仓库隐式单工作区。
- 禁止擅自改 meta 状态或 goal-tree。  
- 结束后提示用户用 **`/govern`** 响应意见。

## 完成

告诉用户：verdict、必改项、写入路径、建议的 `/govern` 下一句。
