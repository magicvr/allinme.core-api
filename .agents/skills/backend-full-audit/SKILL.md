---
name: backend-full-audit
description: Execute the repository's formal, traceable full backend audit. Use only when explicitly invoked to audit all of allinme.core-api; optional focus text may deepen checks but must never narrow repository coverage.
---

# Backend Full Audit

1. Resolve the repository root and read `.github/prompts/backend-full-audit.prompt.md` completely before taking any audit action.
2. Treat that Copilot prompt as the canonical workflow and execute every mandatory full-repository scope and audit-record requirement in it.
3. Interpret invocation text after `$backend-full-audit` as optional `AUDITOR`, `FOCUS`, and `MODE` input. Default to `MODE=audit-only`.
4. Never use focus text to narrow the audit to a plan, feature, diff, directory, or PR. Suggest `$backend-plan-audit` when the user actually wants plan-scoped review.
5. Stop and report the missing canonical prompt if the file cannot be read; do not reconstruct a reduced workflow from memory.

Example invocations:

```text
$backend-full-audit
$backend-full-audit FOCUS=security
$backend-full-audit AUDITOR=codex MODE=remediate
```
