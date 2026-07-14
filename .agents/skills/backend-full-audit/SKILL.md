---
name: backend-full-audit
description: Execute the repository's formal, traceable full backend audit. Use only when explicitly invoked to audit all of allinme.core-api; optional focus text may deepen checks but must never narrow repository coverage.
---

# Backend Full Audit

1. Resolve the repository root and read `.github/prompts/backend-full-audit.prompt.md` completely before taking any audit action.
2. Treat that Copilot prompt as the canonical workflow and execute every mandatory full-repository scope and audit-record requirement in it.
3. Interpret invocation text after `$backend-full-audit` as optional `AUDITOR` and `FOCUS` input.
4. Never use focus text to narrow the audit to a plan, feature, diff, directory, or PR. Suggest `$backend-plan-audit` when the user actually wants plan-scoped review.
5. Never remediate findings in this command. Direct remediation to `$backend-fix-audit-findings` after the indexed audit record is complete.
6. Stop and report the missing canonical prompt if the file cannot be read; do not reconstruct a reduced workflow from memory.
7. 生成的审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。

Example invocations:

```text
$backend-full-audit
$backend-full-audit FOCUS=security
$backend-full-audit AUDITOR=codex FOCUS=protocol
```
