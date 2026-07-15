---
name: backend-implementation-audit
description: Independently audit completed implementations at their exact result revision with one resumable AUD per IMP, checking traceability, code, tests, evidence, CI, recovery, and release gates.
---

# Backend Implementation Audit

<!-- evidence-attestation-contract: external-signed-artifact; exact-run-revision-command-result-image; missing-trust-stops -->

1. 解析仓库根目录，并在审计前完整读取 `.github/prompts/backend-implementation-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行 IMP 选择、实施审计矩阵、AUD 创建、finding 和索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID`、`CONTEXT_REF` 和 `FOCUS`；默认 `TARGET=pending`。`FOCUS` 只能增加深度；`PLN` 目标只能解析到最新 eligible IMP。
4. 只审计已关闭为 `completed` 且具有可 checkout `result_revision` 的 IMP；Evidence 缺失或不完整必须形成负向 finding，不得因此拒绝创建审计。不得把 `in-progress`、`partial` 或 `blocked` 实施视为完成。
5. 不得修改实现、plan/checklist 或 IMP 来消除 finding；整改交给 `$backend-fix-audit-findings`。
6. 必须由运行时创建不同于 implementer 的新 task/agent 并提供真实 `CONTEXT_REF`；`CONTEXT_ID` 只作为 evidence correlation UUID。使用 `implementation-audit/v2`，令 `evidence_revision` 等于 IMP `result_revision`，并记录完整 source context IDs/refs。
7. 多 IMP 调用仅作为分派入口；每个 IMP 必须创建或恢复一份独立 AUD。revision 漂移时按 canonical prompt 终止 stale open 记录。
8. 只有当前真实 `CONTEXT_REF` 与 open AUD 已记录 ref 相同且原 task 可恢复时才能续跑；否则以 `context-loss` supersede 旧记录并新建 AUD，禁止重绑 runtime ref。
9. 每个失败 Control 必须关联当前 AUD finding；通过 detached evidence runner 在 IMP exact result revision 至少重跑一条产品命令和一条主要风险的负向/失败路径。提交 open checkpoint 后再审计，关闭后提交 terminal governance commit，并返回 clean `governance_revision`。
10. 生成的审计记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。
11. 所有实际提交必须调用 `docs/tools/invoke-governance-transaction.ps1`：以完整 HEAD 作为 `ExpectedHead`，只列出本阶段精确 `Paths`，在 Git common-dir 锁内对 HEAD 和 `refs/allinme/governance-head` 做 CAS。open checkpoint 必须精确包含 AUD、必要索引及本记录自身的 `runtime_context_attestation` 文件；`source_context_attestations` 只引用既有文件。terminal transaction 必须精确包含 AUD、必要索引、`evidence.json` 和 `attestation.json`，不得留下 artifact 脏文件。各提交必须线性分开并返回干净 `governance_revision`；禁止裸 `git add`/`git commit`、预暂存或混入用户/并行改动。
12. 当前及全部 source contexts 必须有仓库外运行时签发的单次 attestation；当前签名绑定 exact `record_id`/`record_path`，记录 `runtime_context_attestation` 与精确 `source_context_attestations`，令 `source_context_refs` 等于已验签 source refs 集合，并验证 signed task/ref 均不同。仓库不得持有私钥，缺失 trust anchor 时停止。
13. runner 必须按唯一 `evidence_run_id` 生成 `docs/evidence/runs/<run-id>/evidence.json`，frontmatter 写精确 `evidence_artifact`/`evidence_attestation`；仓库外可信 runtime/CI signer 再签发同目录 `attestation.json`，绑定 artifact 原始 SHA256、run/revision/tree/argv/exit/image 等。主 evidence 命令可非零，但必须形成一致 finding；签名或外部 trust anchor 缺失时不得关闭 AUD。
14. 每份新 AUD 必须使用真实、独立的逐记录 child task/context 和全局唯一 `execution_context_id`；多 IMP 调用只做严格串行 dispatcher，不得复用一个 `CONTEXT_ID`/`CONTEXT_REF`。

```text
$backend-implementation-audit
$backend-implementation-audit TARGET=IMP-0001 CONTEXT_REF=<runtime-task-ref>
```
