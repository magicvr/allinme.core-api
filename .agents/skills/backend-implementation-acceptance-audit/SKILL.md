---
name: backend-implementation-acceptance-audit
description: Verify implementation completion in a separate execution context using the latest IMP and linear verified REM revision chain.
---

# Backend Implementation Acceptance Audit

<!-- evidence-attestation-contract: external-signed-artifact; exact-run-revision-command-result-image; missing-trust-stops -->

1. 解析仓库根目录，并在验收前完整读取 `.github/prompts/backend-implementation-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行目标解析、独立完成验收矩阵、AUD 创建和 IMP/AUD 索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID`、`CONTEXT_REF` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 只能增加深度。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析到计划、该计划最新 IMP 和完整审计链。没有当前有效 ready 计划验收时停止、不创建完成验收记录，并 handoff 到 `$backend-plan-audit-until-ready`。多计划调用必须拆成每个计划一份独立 AUD。
5. 必须由运行时创建不同于 implementer、实施审计、整改和 follow-up 的新 task/agent，并提供真实 `CONTEXT_REF`；缺失时停止。`CONTEXT_ID` 只作为 evidence correlation UUID。记录 source refs、线性派生的 `effective_result_revision` 和唯一 `evidence_run_id`。
6. 必须从索引派生完整计划/实施 AUD、REM 和 follow-up 链；`related_audits` 必须包含最新 ready 计划验收、最新实施审计和终端复审，`related_remediations` 必须包含影响有效实施 revision 的全部 REM，不得漏列较新的失败记录或使用不存在的状态引用。
7. 没有 IMP、IMP 未完成或缺少实施审计时仍创建结构合法的负向 AUD；没有 IMP 时使用 `related_implementations: none`、`effective_result_revision: none`。`partial`/`blocked` IMP 的恢复条件已满足时必须路由 `implement`，仍需授权/决策时才路由 `decision`；不得伪造 IMP 索引或用 REM 代替 IMP。
8. 只有全部 Control 通过、计划与实施审计链均干净且计划验收未漂移时才能写 `acceptance_verdict: complete`。
9. `complete` 前必须通过 detached evidence runner 在 effective revision 独立重跑至少一条与计划风险匹配、且不属于治理 validator 的 subject-specific 验证命令；不能只引用历史结果或在治理 HEAD 上代跑。
10. 只有当前真实 `CONTEXT_REF` 与 open AUD 已记录 ref 相同且原 task 可恢复时才能续跑；否则以 `context-loss` supersede 旧记录并新建 AUD。revision/baseline 漂移使用 `baseline-drift`，禁止重绑 runtime ref。提交 open checkpoint 后再验收，关闭后提交 terminal governance commit，并返回 clean `governance_revision`。生成的记录和最终汇报必须使用中文；固定状态值保持原样。
11. 所有实际提交必须调用 `docs/tools/invoke-governance-transaction.ps1`：以完整 HEAD 作为 `ExpectedHead`，只列出本阶段精确 `Paths`，在 Git common-dir 锁内对 HEAD 和 `refs/allinme/governance-head` 做 CAS。open checkpoint 必须精确包含 AUD、必要索引及本记录自身的 `runtime_context_attestation` 文件；`source_context_attestations` 只引用既有文件。terminal transaction 必须精确包含 AUD、必要索引、`evidence.json` 和 `attestation.json`，不得留下 artifact 脏文件。各提交必须线性分开并返回干净 `governance_revision`；禁止裸 `git add`/`git commit`、预暂存或混入用户/并行改动。
12. 当前及全部 source contexts 必须有仓库外运行时签发的单次 attestation；当前签名绑定 exact `record_id`/`record_path`，记录 `runtime_context_attestation` 与精确 `source_context_attestations`，令 `source_context_refs` 等于已验签 source refs 集合，并验证 signed task/ref 均不同。仓库不得持有私钥，缺失 trust anchor 时停止。
13. runner 必须按唯一 `evidence_run_id` 生成 `docs/evidence/runs/<run-id>/evidence.json`，frontmatter 写精确 `evidence_artifact`/`evidence_attestation`；仓库外可信 runtime/CI signer 再签发同目录 `attestation.json`，绑定 artifact 原始 SHA256、run/revision/tree/argv/exit/image 等。只有主 evidence 命令 `exit_code: 0` 才能写 `complete`；非零只能支撑带一致 finding 的负向验收。签名或外部 trust anchor 缺失时不得关闭 AUD。
14. 每份新 AUD 必须使用真实、独立的逐记录 child task/context 和全局唯一 `execution_context_id`；批量入口严格串行分派，禁止复用一个 `CONTEXT_ID`/`CONTEXT_REF`。

```text
$backend-implementation-acceptance-audit
$backend-implementation-acceptance-audit TARGET=PLN-0005 CONTEXT_REF=<runtime-task-ref>
```
