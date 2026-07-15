---
status: open
governance_contract: audit-loop/v3
workflow_contract_revision: audit-runtime/v1
audit_schema: general-audit/v1
audit_id: AUD-NNNN
auditor: auditor-name-and-version
execution_context_id: 00000000-0000-4000-8000-000000000000
runtime_context_ref: runtime-task-or-agent-ref
runtime_context_attestation: docs/evidence/runtime-attestations/00000000-0000-4000-8000-000000000000.json
audit_type: targeted
scope: feature:subject
subject: concise subject
baseline: git:full-commit-sha; worktree:clean
evidence_revision: git:full-commit-sha; worktree:clean
evidence_worktree_revision: git:full-commit-sha
evidence_runner: docs/tools/invoke-revision-evidence.ps1
evidence_argv_json: ["go", "test", "./..."]
evidence_run_id: 00000000-0000-4000-8000-000000000000
evidence_artifact: docs/evidence/runs/00000000-0000-4000-8000-000000000000/evidence.json
evidence_attestation: docs/evidence/runs/00000000-0000-4000-8000-000000000000/attestation.json
started_at: YYYY-MM-DDTHH:MM:SS+08:00
completed_at: pending
last_updated: YYYY-MM-DD
related_audits: none
related_remediations: none
supersedes: none
related_plans: none
---

# 审计标题

## 目的与范围

## 基线与方法

## 历史关系

## Findings

### AUD-NNNN-F001 - Finding 标题

- Severity:
- Evidence:
- Impact:
- Recommendation:
- Owner:
- Disposition: open

## 验证结果

## 未执行项与剩余风险

## 关闭结论
