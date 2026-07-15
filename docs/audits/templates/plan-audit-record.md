---
status: open
governance_contract: audit-loop/v3
workflow_contract_revision: audit-runtime/v1
audit_schema: plan-audit/v2
audit_id: AUD-NNNN
auditor: auditor-name-and-version
execution_context_id: 00000000-0000-4000-8000-000000000000
runtime_context_ref: runtime-task-or-agent-ref
runtime_context_attestation: docs/evidence/runtime-attestations/00000000-0000-4000-8000-000000000000.json
audit_type: targeted
scope: plan:PLN-NNNN
subject: concise plan subject
baseline: git:full-commit-sha; worktree:clean
evidence_revision: git:full-commit-sha; worktree:clean
evidence_worktree_revision: git:full-commit-sha
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_argv_json: ["<subject-command>", "<arg-1>"]
evidence_run_id: 00000000-0000-4000-8000-000000000000
evidence_artifact: docs/evidence/runs/00000000-0000-4000-8000-000000000000/evidence.json
evidence_attestation: docs/evidence/runs/00000000-0000-4000-8000-000000000000/attestation.json
audited_peer_plans: PLN-NNNN
audited_subject_paths: docs/plans/PLN-NNNN-subject.md, docs/plans/PLN-NNNN-subject-checklist.md
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

# 计划审计标题

## 目的与范围

## 基线与方法

## 历史关系

## Plan/Checklist 审计矩阵

<!-- plan-checklist-audit: PLN-NNNN -->
### PLN-NNNN Plan/Checklist 审计

- Plan: `../../plans/PLN-NNNN-subject.md`
- Checklist: `../../plans/PLN-NNNN-subject-checklist.md`

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| PAIRING | <required:file/section/index evidence> | pass/fail | none 或 AUD-NNNN-F001 |
| PLAN_TO_CHECKLIST | <required:obligation-to-item mapping> | pass/fail | none 或 finding |
| CHECKLIST_TO_PLAN | <required:extra-constraint source mapping> | pass/fail | none 或 finding |
| CHECKED_EVIDENCE | <required:checked counts and dated revision evidence> | pass/fail/not-applicable | none 或 finding |
| GATE_COMPLETENESS | <required:risk-to-gate mapping> | pass/fail | none 或 finding |
| ARCHIVE_CLOSURE | <required:closure and approval evidence> | pass/fail | none 或 finding |

## Findings

### AUD-NNNN-F001 - Finding 标题

- Affected plan: PLN-NNNN
- Severity:
- Evidence:
- Impact:
- Recommendation:
- Owner:
- Disposition: open

## 验证结果

## 未执行项与剩余风险

## 关闭结论
