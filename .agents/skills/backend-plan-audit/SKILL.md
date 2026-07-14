---
name: backend-plan-audit
description: Execute the repository's formal implementation-plan audit. Use when explicitly invoked to audit every active plan by default or one or more plans selected by PLN ID or path; do not claim assurance outside the selected plans.
---

# Backend Plan Audit

1. Resolve the repository root and read `.github/prompts/backend-plan-audit.prompt.md` completely before taking any audit action.
2. Treat that Copilot prompt as the canonical workflow and execute its target resolution, plan and checklist checks, mandatory per-plan checklist matrices, history comparison, audit-record, and validation requirements.
3. Interpret invocation text after `$backend-plan-audit` as optional `TARGET`, `AUDITOR`, and `FOCUS` input. Default to `TARGET=active`.
4. Audit all active plans when no target is supplied. For explicit targets, resolve every requested `PLN` ID or plan path and its checklist; never silently omit an invalid target.
5. Do not close a plan audit unless every selected plan has its own `plan-audit/v2` checklist matrix with all required controls, both file links, concrete evidence, and findings for every failed control.
6. Do not claim assurance outside the selected plans. Record a recommendation for a new plan or implementation audit when evidence indicates a broader issue.
7. Never remediate findings in this command. Direct remediation to `$backend-fix-audit-findings` after the indexed audit record is complete.
8. Stop and report the missing canonical prompt if the file cannot be read; do not reconstruct a reduced workflow from memory.
9. 生成的审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。

Example invocations:

```text
$backend-plan-audit
$backend-plan-audit TARGET=PLN-0005
$backend-plan-audit TARGET="PLN-0005,PLN-0006" FOCUS=recovery
```
