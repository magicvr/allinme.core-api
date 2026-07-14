---
name: backend-fix-audit-findings
description: Remediate findings from every audit currently marked remediation=required by default, or from audit reports selected by AUD ID, path, topic, or natural-language description. Create an indexed REM record and never modify closed audits.
---

# Fix Audit Findings

1. Resolve the repository root and read `.github/prompts/backend-fix-audit-findings.prompt.md` completely before changing files.
2. Treat that prompt as the canonical workflow, including default target selection, finding de-duplication, REM creation, index transitions, implementation, and validation.
3. Interpret invocation text as optional `TARGET`, `OWNER`, `CONTEXT_ID`, and `FOCUS`. Default to `TARGET=active`; FOCUS may deepen but never narrow remediation.
4. Accept explicit audit IDs, paths, lists, topics, or natural-language descriptions. Never silently omit an invalid selected audit.
5. Resume a matching in-progress REM before allocating. Create and index a REM before implementation, record `execution_context_id`, and never self-verify; hand completed work to a different-context `$backend-follow-up-audit`.
6. Stop and report the missing canonical prompt if it cannot be read.
7. 生成的整改记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。

Examples:

```text
$backend-fix-audit-findings
$backend-fix-audit-findings TARGET=AUD-0002
$backend-fix-audit-findings TARGET="AUD-0002,AUD-0003" FOCUS=plan-consistency
```
