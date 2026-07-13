---
name: backend-plan-audit
description: Execute the repository's formal implementation-plan audit. Use when explicitly invoked to audit every active plan by default or one or more plans selected by PLN ID or path; do not represent it as a full repository audit.
---

# Backend Plan Audit

1. Resolve the repository root and read `.github/prompts/backend-plan-audit.prompt.md` completely before taking any audit action.
2. Treat that Copilot prompt as the canonical workflow and execute its target resolution, plan checks, history comparison, audit-record, and validation requirements.
3. Interpret invocation text after `$backend-plan-audit` as optional `TARGET`, `AUDITOR`, and `FOCUS` input. Default to `TARGET=active`.
4. Audit all active plans when no target is supplied. For explicit targets, resolve every requested `PLN` ID or plan path and its checklist; never silently omit an invalid target.
5. Do not claim repository-wide assurance. Recommend `$backend-full-audit` when evidence indicates a systemic issue outside the selected plans.
6. Never remediate findings in this command. Direct remediation to `$backend-fix-audit-findings` after the indexed audit record is complete.
7. Stop and report the missing canonical prompt if the file cannot be read; do not reconstruct a reduced workflow from memory.

Example invocations:

```text
$backend-plan-audit
$backend-plan-audit TARGET=PLN-0005
$backend-plan-audit TARGET="PLN-0005,PLN-0006" FOCUS=recovery
```
