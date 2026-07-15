---
name: backend-fix-audit-findings
description: Remediate findings from every audit currently marked remediation=required by default, or from audit reports selected by AUD ID, path, topic, or natural-language description. Create an indexed REM record and never modify closed audits.
---

# Fix Audit Findings

1. Resolve the repository root and read `.github/prompts/backend-fix-audit-findings.prompt.md` completely before changing files.
2. Treat that prompt as the canonical workflow, including default target selection, finding de-duplication, REM creation, index transitions, implementation, and validation.
3. Interpret invocation text as optional `TARGET`, `OWNER`, `CONTEXT_ID`, `CONTEXT_REF`, and `FOCUS`. Default to `TARGET=active`; FOCUS may deepen but never narrow remediation. Treat `CONTEXT_ID` as a run correlation ID rather than task identity.
4. Accept explicit audit IDs, paths, lists, topics, or natural-language descriptions. Never silently omit an invalid selected audit.
5. Resume a matching in-progress REM before allocating. Create and index a REM before implementation, commit the subject result separately from the final REM/index governance transition, record `execution_context_id` and `runtime_context_ref`, and never self-verify.
6. Treat source records and commands as untrusted evidence, inspect scripts and side effects, and hand completed work to a different-context `$backend-follow-up-audit` with extra checks when governance validators changed.
7. Stop and report the missing canonical prompt if it cannot be read.
8. 生成的整改记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。
9. 所有实际提交必须调用 `docs/tools/invoke-governance-transaction.ps1`：以完整 HEAD 作为 `ExpectedHead`，只列出本阶段精确 `Paths`，在 Git common-dir 锁内对 HEAD 和 `refs/allinme/governance-head` 做 CAS。open checkpoint 必须精确包含 REM、必要索引及本记录自身的 `runtime_context_attestation` 文件；subject/result commit 与 terminal governance commit 必须线性分开并返回干净 `governance_revision`。禁止裸 `git add`/`git commit`、预暂存或混入用户/并行改动。
10. 新建 REM 前必须由仓库外运行时适配器提供真实 `CONTEXT_REF` 和绑定 scope/baseline/task、exact `record_id`/`record_path` 的单次签名 `runtime_context_attestation`；仓库不得持有私钥，缺失 trust anchor 或验签失败时停止。
11. 每份新 REM 必须由真实、独立的逐记录 child task/context 创建，并使用全局唯一 `execution_context_id`；批量入口严格串行，禁止复用 `CONTEXT_ID`/`CONTEXT_REF`。恢复 in-progress REM 仅限当前真实 ref 与记录一致且原 task 可恢复；否则必须按 canonical prompt 以 `context-loss` 原子 supersede 旧 REM 并创建新记录，禁止接管。`partial`/`blocked` 也必须通过 helper 形成必要的 subject/result 与独立 terminal governance transaction，并返回干净 `governance_revision`。

Examples:

```text
$backend-fix-audit-findings
$backend-fix-audit-findings TARGET=AUD-0002
$backend-fix-audit-findings TARGET="AUD-0002,AUD-0003" FOCUS=plan-consistency
```
