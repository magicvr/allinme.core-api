---
status: open
audit_schema: implementation-acceptance/v2
audit_id: AUD-NNNN
auditor: auditor-name-and-version
audit_type: acceptance
acceptance_type: implementation-completion
acceptance_verdict: pending
plan_status_at_acceptance: active
independence_basis: separate-auditor
scope: plan:PLN-NNNN
subject: implementation completion acceptance
baseline: git:full-commit-sha; worktree:clean
evidence_revision: git:full-commit-sha; worktree:clean
evidence_run_id: 00000000-0000-4000-8000-000000000000
effective_result_revision: git:full-commit-sha
started_at: YYYY-MM-DDTHH:MM:SS+08:00
completed_at: pending
last_updated: YYYY-MM-DD
related_audits: none
related_remediations: none
related_implementations: IMP-NNNN
supersedes: none
related_plans: PLN-NNNN
---

# 实施完成验收

<!-- implementation-acceptance-audit: PLN-NNNN -->

- Implementation: `../../implementations/records/IMP-NNNN-YYYYMMDD-implementer-plan-pln-nnnn-subject.md`
- Plan: `../../plans/PLN-NNNN-subject.md`
- Checklist: `../../plans/PLN-NNNN-subject-checklist.md`

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_PRESENT | 最新 IMP 与 IMP/REM effective revision 链证据 | pass/fail | none 或 AUD-NNNN-F001 |
| SCOPE_COMPLETE | 具体证据 | pass/fail | none 或 finding |
| CHECKLIST_COMPLETE | 具体证据 | pass/fail | none 或 finding |
| VALIDATION_GATES | 具体证据 | pass/fail | none 或 finding |
| AUDIT_CHAIN_CLEAN | 计划/实施 AUD、REM、follow-up 链和验收基线证据 | pass/fail | none 或 finding |
| RESIDUAL_RISK | 具体证据 | pass/fail | none 或 finding |
| ARCHIVE_READY | 具体证据 | pass/fail | none 或 finding |

## Findings

## 验证结果

## 未执行项与剩余风险

## 验收结论
