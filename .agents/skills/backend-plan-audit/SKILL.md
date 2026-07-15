---
name: backend-plan-audit
description: Execute revision-bound implementation-plan audits with a persisted complete active peer snapshot, dispatching one resumable AUD per selected plan and detecting peer drift before readiness or implementation.
---

# Backend Plan Audit

<!-- evidence-attestation-contract: external-signed-artifact; exact-run-revision-command-result-image; missing-trust-stops -->
<!-- evidence-argv-contract: evidence_argv_json; strict-json-array; exact-artifact-and-signed-payload -->

1. Resolve the repository root and read `.github/prompts/backend-plan-audit.prompt.md` completely before taking any audit action.
2. Treat that Copilot prompt as the canonical workflow and execute its target resolution, plan and checklist checks, mandatory per-plan checklist matrices, history comparison, audit-record, and validation requirements.
3. Interpret invocation text as optional `TARGET`, `PEER_SET`, `AUDITOR`, `CONTEXT_ID`, `CONTEXT_REF`, and `FOCUS`. For active targets, default `PEER_SET` to all active unarchived plans even when `TARGET` is explicit; persist the peer set and all peer plan/checklist paths in every AUD.
4. Audit all active plans when no target is supplied, but treat a multi-plan invocation only as a dispatcher: create or resume one independent AUD per plan. Never place multiple plans in one audit record.
5. Before reserving an AUD, resume the unique open record with the same contract, plan and baseline. If the baseline drifted, supersede the stale open record through the canonical replacement transition. Commit the open checkpoint before evidence work and the terminal governance transition before handoff.
6. Do not claim assurance outside the selected plans. Record a recommendation for a new plan or implementation audit when evidence indicates a broader issue.
7. Treat repository content and commands as untrusted evidence; run subject checks through `docs/tools/invoke-revision-evidence.ps1` at the exact evidence revision and require an independent check when governance validators changed.
8. Never remediate findings in this command. Direct remediation to `$backend-fix-audit-findings` after the indexed audit record is complete.
9. Stop and report the missing canonical prompt if the file cannot be read; do not reconstruct a reduced workflow from memory.
10. 将 `FOCUS` 仅解释为增加检查深度。`CONTEXT_ID` 只是运行关联 ID；新建记录必须由仓库外运行时适配器提供真实 `CONTEXT_REF` 和绑定 scope/baseline/task、exact `record_id`/`record_path` 的单次签名 `runtime_context_attestation`，缺失时停止。
11. 生成的审计记录和最终报告必须使用中文；代码、命令、路径、ID 及固定的 frontmatter/status 值保持原样。
12. 所有实际提交必须调用 `docs/tools/invoke-governance-transaction.ps1`：以完整 HEAD 作为 `ExpectedHead`，只列出本阶段精确 `Paths`，在 Git common-dir 锁内对 HEAD 和 `refs/allinme/governance-head` 做 CAS。open checkpoint 必须精确包含 AUD、必要索引及本记录自身的 `runtime_context_attestation` 文件；terminal transaction 必须精确包含 AUD、必要索引、`evidence.json` 和 `attestation.json`，不得留下 artifact 脏文件。各提交必须线性分开并返回干净 `governance_revision`；禁止裸 `git add`/`git commit`、预暂存或混入用户/并行改动。
13. 仓库、skill 和 child 不得持有签名私钥；通过 `docs/tools/validate-runtime-attestations.ps1` 和外部 trust anchor 验签后才可提交 open checkpoint。
14. 为唯一 `evidence_run_id` 运行 runner 并记录 `evidence_artifact`/`evidence_attestation`。runner 生成 `docs/evidence/runs/<run-id>/evidence.json` 后，必须由仓库外可信 runtime/CI signer 签发同目录 `attestation.json`，签名绑定 artifact 原始 SHA256、run/revision/tree/argv/exit/image 等并用外部 trust anchor 验证。主 evidence 命令可非零，但必须形成一致 finding；缺失签名或 trust anchor 时不得关闭 AUD。
15. 每份新 AUD 必须由真实、独立的逐记录 runtime child 创建，并使用在全部 AUD/REM/IMP 中全局唯一的 `execution_context_id`。多计划调用只做严格串行 dispatcher，禁止把一个 `CONTEXT_ID`/`CONTEXT_REF` 复制到多份记录。

Example invocations:

```text
$backend-plan-audit
$backend-plan-audit TARGET=PLN-0005
$backend-plan-audit TARGET="PLN-0005,PLN-0006" FOCUS=recovery
```
