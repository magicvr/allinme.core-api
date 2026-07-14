---
name: backend-follow-up-audit
description: Verify pending or selected REM records in a separate execution context, creating or resuming one follow-up AUD per remediation.
---

# Follow-up Audit

1. Resolve the repository root and read `.github/prompts/backend-follow-up-audit.prompt.md` completely before reviewing remediation work.
2. Treat that prompt as the canonical workflow, including pending-target selection, independent tests, new follow-up AUD creation, and AUD/REM index transitions.
3. Interpret invocation text as optional `TARGET`, `AUDITOR`, `CONTEXT_ID`, `CONTEXT_REF`, and `FOCUS`. Default to `TARGET=pending`; FOCUS may deepen but never narrow review.
4. Accept REM IDs, paths, lists, topics, or an AUD ID that resolves to its latest pending REM. Never review an in-progress or unindexed remediation as completed work.
5. Require the runtime to create a new task/agent distinct from the REM implementer and source audit contexts and supply its real `CONTEXT_REF`; use `CONTEXT_ID` only as an evidence correlation UUID, never as proof of isolation.
6. Resume a matching open follow-up only when the current real `CONTEXT_REF` matches the recorded ref and the original task is recoverable. Otherwise supersede it with `context-loss` and create a new AUD; use `baseline-drift` for revision/chain drift and never rebind an open AUD to a new runtime ref.
7. Commit the open checkpoint before review and the terminal governance transition before handoff; return a clean `governance_revision`. Run safe independent checks through the detached evidence runner at the REM result revision.
8. Stop and report the missing canonical prompt if it cannot be read.
9. 生成的后续复审审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。

Examples:

```text
$backend-follow-up-audit
$backend-follow-up-audit TARGET=REM-0001 CONTEXT_REF=<runtime-task-ref>
$backend-follow-up-audit TARGET=AUD-0002 FOCUS=regression
```
