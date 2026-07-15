---
status: open
governance_contract: audit-loop/v3
workflow_contract_revision: audit-runtime/v1
audit_schema: plan-acceptance/v2
audit_id: AUD-NNNN
auditor: auditor-name-and-version
execution_context_id: 00000000-0000-4000-8000-000000000000
runtime_context_ref: runtime-task-or-agent-ref
runtime_context_attestation: docs/evidence/runtime-attestations/00000000-0000-4000-8000-000000000000.json
source_context_ids: legacy-unavailable
source_context_refs: legacy-unavailable
source_context_attestations: docs/evidence/runtime-attestations/00000000-0000-4000-8000-000000000000.json
audit_type: acceptance
acceptance_type: plan-readiness
acceptance_verdict: pending
plan_status_at_acceptance: active
independence_basis: separate-context
scope: plan:PLN-NNNN
subject: plan readiness acceptance
baseline: git:full-commit-sha; worktree:clean
evidence_revision: git:full-commit-sha; worktree:clean
evidence_worktree_revision: git:full-commit-sha
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_argv_json: ["<subject-command>", "<arg-1>"]
evidence_run_id: 00000000-0000-4000-8000-000000000000
evidence_artifact: docs/evidence/runs/00000000-0000-4000-8000-000000000000/evidence.json
evidence_attestation: docs/evidence/runs/00000000-0000-4000-8000-000000000000/attestation.json
started_at: YYYY-MM-DDTHH:MM:SS+08:00
completed_at: pending
last_updated: YYYY-MM-DD
related_audits: none
related_remediations: none
supersedes: none
superseded_by: none
supersession_reason: none
related_plans: PLN-NNNN
---

# 计划实施就绪验收

<!-- plan-acceptance-audit: PLN-NNNN -->

- Plan: `../../plans/PLN-NNNN-subject.md`
- Checklist: `../../plans/PLN-NNNN-subject-checklist.md`

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| READY_IDENTITY | <required:file/frontmatter/index evidence> | pass/fail | none 或 AUD-NNNN-F001 |
| READY_SCOPE | <required:scope section evidence> | pass/fail | none 或 finding |
| READY_FACTS | <required:fact-source/code evidence> | pass/fail | none 或 finding |
| READY_DEPENDENCIES | <required:dependency/version evidence> | pass/fail | none 或 finding |
| READY_DESIGN | <required:decision/stop-condition evidence> | pass/fail | none 或 finding |
| READY_EVIDENCE | <required:test/CI/recovery plan evidence> | pass/fail | none 或 finding |
| READY_GATES | <required:entry/release/owner evidence> | pass/fail | none 或 finding |
| PLAN_AUDIT_CHAIN_CLEAN | <required:derived AUD/REM chain evidence> | pass/fail | none 或 finding |

## Findings

## 验证结果

## 未执行项与剩余风险

## 验收结论
