---
status: open
governance_contract: audit-loop/v3
workflow_contract_revision: audit-runtime/v1
audit_schema: implementation-acceptance/v2
audit_id: AUD-NNNN
auditor: auditor-name-and-version
execution_context_id: 00000000-0000-4000-8000-000000000000
runtime_context_ref: runtime-task-or-agent-ref
source_context_ids: legacy-unavailable
source_context_refs: legacy-unavailable
audit_type: acceptance
acceptance_type: implementation-completion
acceptance_verdict: pending
acceptance_next_action: pending
plan_status_at_acceptance: active
independence_basis: separate-context
scope: plan:PLN-NNNN
subject: implementation completion acceptance
baseline: git:full-commit-sha; worktree:clean
evidence_revision: git:full-commit-sha; worktree:clean
evidence_worktree_revision: git:full-commit-sha
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_run_id: 00000000-0000-4000-8000-000000000000
effective_result_revision: none
started_at: YYYY-MM-DDTHH:MM:SS+08:00
completed_at: pending
last_updated: YYYY-MM-DD
related_audits: none
related_remediations: none
related_implementations: none
supersedes: none
superseded_by: none
supersession_reason: none
related_plans: PLN-NNNN
---

# 实施完成验收

<!-- implementation-acceptance-audit: PLN-NNNN -->

- Implementation: `none`（存在 IMP 时替换为稳定路径）
- Plan: `../../plans/PLN-NNNN-subject.md`
- Checklist: `../../plans/PLN-NNNN-subject-checklist.md`

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_PRESENT | <required:IMP and linear effective revision chain> | pass/fail | none 或 AUD-NNNN-F001 |
| SCOPE_COMPLETE | <required:plan-to-change evidence> | pass/fail | none 或 finding |
| CHECKLIST_COMPLETE | <required:item/date/revision evidence> | pass/fail | none 或 finding |
| VALIDATION_GATES | <required:commands/results/artifacts> | pass/fail | none 或 finding |
| AUDIT_CHAIN_CLEAN | <required:derived AUD/REM/follow-up evidence> | pass/fail | none 或 finding |
| RESIDUAL_RISK | <required:risk owner/decision evidence> | pass/fail | none 或 finding |
| ARCHIVE_READY | <required:closure/approval prerequisites> | pass/fail | none 或 finding |

## Findings

## 验证结果

## 未执行项与剩余风险

## 验收结论
