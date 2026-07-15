---
name: backend-follow-up-audit
description: Verify pending or selected REM records in a separate execution context, creating or resuming one follow-up AUD per remediation.
---

# Follow-up Audit

<!-- evidence-attestation-contract: external-signed-artifact; exact-run-revision-command-result-image; missing-trust-stops -->

1. Resolve the repository root and read `.github/prompts/backend-follow-up-audit.prompt.md` completely before reviewing remediation work.
2. Treat that prompt as the canonical workflow, including pending-target selection, independent tests, new follow-up AUD creation, and AUD/REM index transitions.
3. Interpret invocation text as optional `TARGET`, `AUDITOR`, `CONTEXT_ID`, `CONTEXT_REF`, and `FOCUS`. Default to `TARGET=pending`; FOCUS may deepen but never narrow review.
4. Accept REM IDs, paths, lists, topics, or an AUD ID that resolves to its latest pending REM. Never review an in-progress or unindexed remediation as completed work.
5. Require the runtime to create a new task/agent distinct from the REM implementer and source audit contexts and supply its real `CONTEXT_REF`; use `CONTEXT_ID` only as an evidence correlation UUID, never as proof of isolation.
6. Resume a matching open follow-up only when the current real `CONTEXT_REF` matches the recorded ref and the original task is recoverable. Otherwise supersede it with `context-loss` and create a new AUD; use `baseline-drift` for revision/chain drift and never rebind an open AUD to a new runtime ref.
7. Commit the open checkpoint before review and the terminal governance transition before handoff; return a clean `governance_revision`. Run safe independent checks through the detached evidence runner at the REM result revision.
8. Stop and report the missing canonical prompt if it cannot be read.
9. 生成的后续复审审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。
10. 所有实际提交必须调用 `docs/tools/invoke-governance-transaction.ps1`：以完整 HEAD 作为 `ExpectedHead`，只列出本阶段精确 `Paths`，在 Git common-dir 锁内对 HEAD 和 `refs/allinme/governance-head` 做 CAS。open checkpoint 必须精确包含 AUD、必要索引及本记录自身的 `runtime_context_attestation` 文件；`source_context_attestations` 只引用既有文件。terminal transaction 必须精确包含 AUD、必要索引、`evidence.json` 和 `attestation.json`，不得留下 artifact 脏文件。各提交必须线性分开并返回干净 `governance_revision`；禁止裸 `git add`/`git commit`、预暂存或混入用户/并行改动。
11. 当前及全部 source contexts 必须有仓库外运行时签发的单次 attestation；当前签名绑定 exact `record_id`/`record_path`，记录 `runtime_context_attestation` 与精确 `source_context_attestations`，令 `source_context_refs` 等于已验签 source refs 集合，并验证 signed task/ref 均不同。仓库不得持有私钥，缺失 trust anchor 时停止。
12. runner 必须按唯一 `evidence_run_id` 生成 `docs/evidence/runs/<run-id>/evidence.json`，frontmatter 写精确 `evidence_artifact`/`evidence_attestation`；仓库外可信 runtime/CI signer 再签发同目录 `attestation.json`，绑定 artifact 原始 SHA256、run/revision/tree/argv/exit/image 等。主 evidence 命令可非零，但必须形成一致 verdict/finding；签名或外部 trust anchor 缺失时不得关闭 AUD。
13. 每份新 AUD 必须使用真实、独立的逐记录 child task/context 和全局唯一 `execution_context_id`；多 REM 入口严格串行分派，禁止复用一个 `CONTEXT_ID`/`CONTEXT_REF`。当前合同不支持外部风险批准 attestation，因此不得自行写 `accepted-risk`；需要接受风险时保持 finding 未解决并路由 `decision-required`。

Examples:

```text
$backend-follow-up-audit
$backend-follow-up-audit TARGET=REM-0001 CONTEXT_REF=<runtime-task-ref>
$backend-follow-up-audit TARGET=AUD-0002 FOCUS=regression
```
