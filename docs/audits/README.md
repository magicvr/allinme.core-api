# 审计记录管理

`records/` 保存实际发生过的审计。审计记录回答：在什么基线上审了什么、发现了什么、执行了哪些验证、哪些风险仍然存在。计划、实施和整改分别位于 `../plans/`、`../implementations/` 和 `../remediations/`。

## 核心原则

1. 每次正式审计创建独立 `AUD-NNNN`；即使没有 finding，也保留范围、基线、验证和剩余风险。
2. 一份计划或一份 IMP 对应一份 AUD；批量调用可以做交叉检查，但不能共享 verdict 或 remediation 状态。
3. 审计记录绑定 `baseline` 和被检查的 `evidence_revision`。对象漂移后，旧结论不能继续作为当前验收依据。
4. 实施审计、整改复审和两类验收必须在不同于被审对象作者/执行者的运行时上下文中执行，并记录真实 `runtime_context_ref`。该字段用于追踪，不是密码学证明。
5. 审计必须检查实际 subject，并按风险执行可证伪验证；治理脚本通过不等于 subject 正确。
6. 审计只记录结论，不直接整改。整改创建 REM，复审再创建新的 follow-up AUD。
7. `status: closed|superseded` 的记录不可改写；新证据或相反结论通过新记录和关联字段表达。
8. 未执行验证必须明确记录，不得写成通过。

## 命名与最小元数据

- 文件名：`AUD-NNNN-YYYYMMDD-<auditor>-<scope-kind>-<subject>.md`。
- 必需字段：`status`、`audit_id`、`auditor`、`audit_type`、`scope`、`subject`、`baseline`、`started_at`、`last_updated`；关闭时增加 `completed_at`。
- 计划/实施审计和验收还应记录 `evidence_revision`；独立流程记录 `independence_basis: separate-context` 和真实上下文引用。
- finding 使用 `AUD-NNNN-F001`，并记录 severity、evidence、impact、recommendation、owner、disposition。

## 当前索引

- [`AUD-0010`](./records/AUD-0010-20260716-claude-governance-goal-drift.md)：`status=open`；目标漂移专项；关联 `PLN-0006`。
- [`AUD-0009`](./records/AUD-0009-20260714-codex-follow-up-rem-0005-active-audits.md)：`status=closed`；`remediation=retired-by-scope`。
- [`AUD-0008`](./records/AUD-0008-20260714-codex-follow-up-rem-0004-contract-clause-parsers.md)：`status=closed`；`remediation=retired-by-scope`。
- [`AUD-0007`](./records/AUD-0007-20260714-codex-plan-pln-0005-phase-05-attachment-lifecycle.md)：`status=closed`；`remediation=retired-by-scope`。
- [`AUD-0006`](./records/AUD-0006-20260714-codex-follow-up-rem-0003-contract-validators.md)：`status=closed`；`remediation=retired-by-scope`。
- [`AUD-0005`](./records/AUD-0005-20260714-codex-follow-up-rem-0002-contracts.md)：`status=closed`；`remediation=retired-by-scope`。
- [`AUD-0004`](./records/AUD-0004-20260714-codex-follow-up-rem-0001-active-audits.md)：`status=closed`；历史整改链已结束。
- [`AUD-0003`](./records/AUD-0003-20260714-github-copilot-plan-pln-0005-phase-05-attachment-lifecycle.md)：`status=closed`；历史整改链已结束。
- [`AUD-0002`](./records/AUD-0002-20260714-codex-plan-phase-05-attachment-lifecycle.md)：`status=closed`；历史整改链已结束。
- [`AUD-0001`](./records/AUD-0001-20260714-codex-repository-docs-governance.md)：`status=closed`；`remediation=none`。

`retired-by-scope` 表示 finding 仅针对已经删除的目标外自然语言门禁；它保留历史，但不再驱动整改。它不能用于仍影响计划、代码、测试或闭环正确性的 finding。

## 工作流入口

以下入口是按风险选用的 agent 辅助工具，不是每个产品变更的默认前置或 CI 门禁。默认产品门禁以 [`../04-validation.md`](../04-validation.md) 为准；只有维护 prompt/skill 时才额外运行 `validate-audit-workflows.ps1`。

| 工作类型 | Prompt | Skill |
|---|---|---|
| 计划审计 | [prompt](../../.github/prompts/backend-plan-audit.prompt.md) | [skill](../../.agents/skills/backend-plan-audit/SKILL.md) |
| 计划就绪验收 | [prompt](../../.github/prompts/backend-plan-acceptance-audit.prompt.md) | [skill](../../.agents/skills/backend-plan-acceptance-audit/SKILL.md) |
| 计划审计闭环 | [prompt](../../.github/prompts/backend-plan-audit-until-ready.prompt.md) | [skill](../../.agents/skills/backend-plan-audit-until-ready/SKILL.md) |
| 计划实施 | [prompt](../../.github/prompts/backend-implement-plan.prompt.md) | [skill](../../.agents/skills/backend-implement-plan/SKILL.md) |
| 实施审计 | [prompt](../../.github/prompts/backend-implementation-audit.prompt.md) | [skill](../../.agents/skills/backend-implementation-audit/SKILL.md) |
| 实施完成验收 | [prompt](../../.github/prompts/backend-implementation-acceptance-audit.prompt.md) | [skill](../../.agents/skills/backend-implementation-acceptance-audit/SKILL.md) |
| 实施审计闭环 | [prompt](../../.github/prompts/backend-implement-audit-until-complete.prompt.md) | [skill](../../.agents/skills/backend-implement-audit-until-complete/SKILL.md) |
| 整改 | [prompt](../../.github/prompts/backend-fix-audit-findings.prompt.md) | [skill](../../.agents/skills/backend-fix-audit-findings/SKILL.md) |
| 整改复审 | [prompt](../../.github/prompts/backend-follow-up-audit.prompt.md) | [skill](../../.agents/skills/backend-follow-up-audit/SKILL.md) |

Prompt 是唯一流程正文；skill 只负责读取并执行对应 prompt。模板位于 [`templates/`](./templates/)。
