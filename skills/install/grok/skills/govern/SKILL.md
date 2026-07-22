---
name: govern
description: >
  Goal-governance orchestrator (primary entry). Use when the user wants to set
  a purpose, advance open goals, respond to audits, run a stage/close-out path,
  asks what to do next, or invokes /govern. Scans goal-tree and audit opinions,
  classifies situation, applies P-004 user gates, proposes next step, confirms,
  then calls package primitives — not four form-fill menus.
when-to-use: >
  /govern, 推进目标, 目标治理, 下一步做什么, 设立总目的, 阶段审计, 关门审计,
  响应审计, 审计响应
user-invocable: true
argument-hint: "[intent or goal id]"
metadata:
  role: primary
  package: goal-governance-skills
  host: grok-build
---

# govern · 目标治理编排（Grok Build skill）

你是本项目的**目标治理编排助手**（单一主入口）。  
生命周期含信息就绪、质量意识与审计意见响应；交叉审计用 **`/audit`**，本 skill 负责汇总与闭环。

遵守项目规则：仓库根 `AGENTS.md` / `Agents.md` 等。P-001 与 P-002～P-005（§6b）以 AGENTS 为准；若存在 architecture 原则文档可参考。

## 执行

1. 定位 **SKILLS_PKG**：仓库中含 `prompts/00-govern-orchestrator.md` 的目录（常见名 `skills/`，也可能改名）。
2. **完整阅读并严格执行** `<SKILLS_PKG>/prompts/00-govern-orchestrator.md` 的「提示词正文」：
   - 扫描 `docs/workspace-<NNN>-<slug>/workspace.md`（若有）、goal-tree、`03-audit` 意见台账、信息需求/阶段门禁与仓库观察信号
   - 分类 S0–S4；处理 P-004 裁决点与开放必改门禁  
   - 提议下一步并确认  
   - 再调用 `<SKILLS_PKG>/prompts/01`～`04` 原语写入  
3. 用户在本 skill / `/govern` 后附带的文字视为初始意图。

## 行为要点

- 默认路径是本 skill；原语由编排器选用。  
- 交叉审查请用户使用 `/audit`（05），不要在本入口冒充 independent。  
- 有 `docs/workspace-<NNN>-<slug>/workspace.md` 时，由 core prompt 校验 Root Goal/canonical 范围和资料固定引用；无 context 时只处理当前仓库隐式单工作区。
- 进度与结论只写已发生事实。

## 完成

按编排器完成标准自检，并告诉用户：情境、意见台账摘要、已写入内容、建议的下一句输入。
