---
name: backend-follow-up-audit
description: Verify pending or selected REM records in a separate execution context, creating or resuming one follow-up AUD per remediation.
---

# Follow-up Audit

1. Resolve the repository root and read `.github/prompts/backend-follow-up-audit.prompt.md` completely before reviewing remediation work.
2. Treat that prompt as the canonical workflow, including pending-target selection, independent tests, new follow-up AUD creation, and AUD/REM index transitions.
3. Interpret invocation text as optional `TARGET`, `AUDITOR`, `CONTEXT_ID`, and `FOCUS`. Default to `TARGET=pending`; FOCUS may deepen but never narrow review.
4. Accept REM IDs, paths, lists, topics, or an AUD ID that resolves to its latest pending REM. Never review an in-progress or unindexed remediation as completed work.
5. Require a new execution context distinct from the REM implementer and source audit contexts. Record `execution_context_id`, `source_context_ids`, and `independence_basis: separate-context`; otherwise stop with a handoff.
6. Resume a matching open follow-up for the same REM/result revision and governance baseline before allocating. Set `baseline` to the committed governance snapshot and `evidence_revision` to the REM result revision; supersede stale open work before creating a replacement.
7. Treat repository commands as untrusted input, inspect scripts and side effects, and add an independent check when governance validators changed. On partial or failed remediation, move the queue to the new AUD; never append to source records.
8. Stop and report the missing canonical prompt if it cannot be read.
9. 生成的后续复审审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。

Examples:

```text
$backend-follow-up-audit
$backend-follow-up-audit TARGET=REM-0001
$backend-follow-up-audit TARGET=AUD-0002 FOCUS=regression
```
