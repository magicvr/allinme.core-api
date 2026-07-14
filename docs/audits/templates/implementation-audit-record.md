---
status: open
governance_contract: audit-loop/v3
audit_schema: implementation-audit/v1
audit_id: AUD-NNNN
auditor: auditor-name-and-version
execution_context_id: 00000000-0000-4000-8000-000000000000
audit_type: implementation
scope: implementation:IMP-NNNN
subject: implementation audit
baseline: git:full-commit-sha; worktree:clean
started_at: YYYY-MM-DDTHH:MM:SS+08:00
completed_at: pending
last_updated: YYYY-MM-DD
related_audits: none
related_remediations: none
related_implementations: IMP-NNNN
supersedes: none
related_plans: PLN-NNNN
---

# 实施审计

<!-- implementation-audit: IMP-NNNN -->

- Implementation: `../../implementations/records/IMP-NNNN-YYYYMMDD-implementer-plan-pln-nnnn-subject.md`
- Plan: `../../plans/PLN-NNNN-subject.md`
- Checklist: `../../plans/PLN-NNNN-subject-checklist.md`

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_TRACEABILITY | <required:IMP/plan/revision mapping> | pass/fail | none 或 AUD-NNNN-F001 |
| CHECKLIST_EVIDENCE | <required:item/date/command/result evidence> | pass/fail | none 或 finding |
| CODE_CONTRACT | <required:file/symbol/contract evidence> | pass/fail | none 或 finding |
| TEST_FAILURE | <required:commands and failure-path results> | pass/fail | none 或 finding |
| SECURITY_DATA | <required:security/data checks> | pass/fail | none 或 finding |
| MIGRATION_RECOVERY | <required:migration/recovery checks> | pass/fail | none 或 finding |
| DOCS_CI_RELEASE | <required:docs/CI/artifact evidence> | pass/fail | none 或 finding |

## Findings

## 验证结果

## 未执行项与剩余风险

## 关闭结论
