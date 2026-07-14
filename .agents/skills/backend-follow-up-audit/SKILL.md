---
name: backend-follow-up-audit
description: Independently verify every remediation record marked verification=pending by default, or selected REM/AUD records. Review the REM and its source audits, create a new indexed follow-up AUD, and never append to closed records.
---

# Follow-up Audit

1. Resolve the repository root and read `.github/prompts/backend-follow-up-audit.prompt.md` completely before reviewing remediation work.
2. Treat that prompt as the canonical workflow, including pending-target selection, independent tests, new follow-up AUD creation, and AUD/REM index transitions.
3. Interpret invocation text after `$backend-follow-up-audit` as optional `TARGET`, `AUDITOR`, and `FOCUS`. Default to `TARGET=pending`.
4. Accept REM IDs, paths, lists, topics, or an AUD ID that resolves to its latest pending REM. Never review an in-progress or unindexed remediation as completed work.
5. Always create a new follow-up AUD. On partial or failed remediation, record new open findings and move the active remediation queue to that follow-up AUD; never append to the source AUD or REM.
6. Stop and report the missing canonical prompt if it cannot be read.
7. 生成的后续复审审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。

Examples:

```text
$backend-follow-up-audit
$backend-follow-up-audit TARGET=REM-0001
$backend-follow-up-audit TARGET=AUD-0002 FOCUS=regression
```
