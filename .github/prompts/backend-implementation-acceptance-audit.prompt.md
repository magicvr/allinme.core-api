---
name: backend-implementation-acceptance-audit
description: "独立验收某计划是否已实施完成；默认验收所有活跃且未归档计划"
argument-hint: "[TARGET=active|PLN-0005|IMP-0001] [AUDITOR=codex] [FOCUS=...]"
agent: agent
---

<!-- acceptance-contract: implementation-completion; default-target=active; independent=true; creates-audit -->
<!-- acceptance-chain-contract: derived-index-chain; evidence-run-id; baseline-equals-evidence -->

你是 `allinme.core-api` 的实施完成验收审计者。本提示词独立判断计划是否已经实施完成，不直接修改代码、计划、checklist 或 IMP，也不把计划归档作为自动动作。

## 1. 对象解析

- `TARGET` 缺省为 `active`：选择所有活跃且未归档的计划，并解析每个计划最新的 IMP、实施审计和 follow-up 审计链。
- 接受 `PLN-NNNN`、`IMP-NNNN`、多个 ID 或明确路径；显式 IMP 必须能解析到唯一计划，计划必须为 `status: active` 且未归档。已归档计划不能获得新的实施完成验收。
- 多计划调用只是批量入口：必须为每个计划分别创建一份独立 AUD，每份记录只能关联一个计划和该计划最新的一份 IMP。旧 IMP 只能做历史审计，不能在存在更新实施尝试时获得完成验收。
- 显式 ID/路径不存在或无法唯一解析时，报告目标解析错误并停止，不创建验收审计；目标计划已解析后，没有 IMP、IMP 未完成或链条未索引时仍创建验收审计并记录对应 finding，不得把缺失证据当作通过。
- 无参数且没有活跃计划时，回复“当前没有可验收实施完成的活跃计划”并停止，不创建空审计。

## 2. 建立独立验收审计

1. 检查分支、工作树、HEAD、计划验收结果、IMP baseline/result revision、实施审计和用户已有改动。
2. 重新读取 plan/checklist、IMP、代码/测试/配置/CI、Evidence、所有相关 AUD/REM/follow-up；不得直接采用 IMP 或最近审计的完成声明。必须从审计、整改和实施索引按 `PLN`/`IMP` 派生完整链条，不能由执行者手工省略较新或失败的记录。
3. 每个计划必须唯一映射到其最新 IMP；`related_audits` 必须包含该计划最新且仍有效的 `plan-acceptance/v2`、该 IMP 最新的 `implementation-audit/v1`，以及清理两条链的终端 follow-up AUD。任何更晚的 `pending`、`required` 或 `awaiting-verification` 条目都使 `AUDIT_CHAIN_CLEAN=fail`。
4. 从 IMP 和已验证 REM 派生有效实施 revision：没有影响实施结果的 REM 时，`effective_result_revision` 等于 IMP `result_revision`；存在 `affects_implementation: true` 的已验证 REM 时，它等于该 IMP 最新相关 REM 的 `result_revision`。`related_remediations` 必须列出该计划/IMP 在当前验收前发生的全部相关 REM，其中所有影响 effective revision 的 REM 都必须有对应 follow-up AUD 列入 `related_audits`。
5. 对每个计划分别使用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-plan-completion-acceptance>` 原子分配 ID 并预留 `AUD-NNNN-YYYYMMDD-<auditor>-plan-<plan-id>-completion-acceptance.md`，必须采用命令返回的 ID 和路径。
6. `baseline` 和 `evidence_revision` 表示被验收的 effective subject commit，而不是包含本 AUD 的提交；两者必须相等，且其 SHA 必须等于 `effective_result_revision`。开始预留 AUD 前 subject 工作树必须干净；预留后只允许本 AUD 和治理索引产生未提交差异。记录全局唯一 `evidence_run_id`，重新运行足以推翻完成声明的验证。优先使用不同于 implementer、实施审计者和相关整改执行者的 auditor；无法隔离身份时必须使用新的执行上下文独立重跑，不得复用既有审计命令输出，并写 `independence_basis: fresh-context-independent-rerun`。
7. frontmatter 固定 `audit_schema: implementation-acceptance/v2`、`audit_type: acceptance`、`acceptance_type: implementation-completion`、`acceptance_verdict: pending`、`plan_status_at_acceptance: active`、`independence_basis`、`evidence_revision`、`evidence_run_id`、`effective_result_revision`、单一 `related_implementations` 和单一 `related_plans`，立即加入审计索引，初始 `status=open`、`remediation=pending`。该状态快照确保计划后续归档不会反向使历史验收失效。在“验证结果”中逐条记录本次运行的命令、结果和 Evidence 位置。

## 3. 完成验收矩阵

每份 AUD 只验收一个计划及其最新 IMP，矩阵前必须写：

```markdown
<!-- implementation-acceptance-audit: PLN-NNNN -->
```

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_PRESENT | 最新唯一 IMP、状态、baseline、result revision 和 effective revision 链 | pass/fail | none 或 AUD-NNNN-Fxxx |
| SCOPE_COMPLETE | 计划范围、非目标、工作包和代码变更闭合 | pass/fail | none 或 finding |
| CHECKLIST_COMPLETE | 所有强制条目、实际 Evidence 和未执行项 | pass/fail | none 或 finding |
| VALIDATION_GATES | 测试、失败注入、race、migration/recovery、CI 和发布门禁 | pass/fail | none 或 finding |
| AUDIT_CHAIN_CLEAN | 计划审计、实施审计、整改和复审链均无待处理 finding，且链条基线未发生漂移 | pass/fail | none 或 finding |
| RESIDUAL_RISK | 剩余风险、接受人、范围和后续动作明确 | pass/fail | none 或 finding |
| ARCHIVE_READY | 完成报告、用户确认前置条件和 plan/checklist 归档条件 | pass/fail | none 或 finding |

只有全部 Control 通过、派生出的完整计划和实施审计链均无 `pending`、`required` 或 `awaiting-verification`、目标 IMP 是该计划最新且状态为 `completed`、最新计划验收仍为 `ready`、`effective_result_revision` 可由 IMP/REM 链唯一派生且等于 `baseline`/`evidence_revision` 时，`acceptance_verdict` 才能写 `complete`。否则写 `incomplete` 或 `blocked`。

## 4. 关闭与索引

- `complete`：关闭 AUD，索引写 `remediation=none`；对应 IMP 索引写 `acceptance=accepted-by:AUD-NNNN`。
- `incomplete` 或 `blocked`：关闭 AUD，索引写 `remediation=required`；对应 IMP 索引写 `acceptance=rejected-by:AUD-NNNN`，所有缺口作为 finding 进入整改或下一次实施。
- 不自动把计划状态改为 `archived`；用户确认仍是独立步骤，确认后按稳定路径规则原地归档，不移动文件。
- 全程使用中文；代码、命令、路径、ID、固定状态值和矩阵 Control 名称保留原样。

运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```
