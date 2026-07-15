---
name: backend-implement-audit-until-complete
description: Orchestrate set-aware readiness, implementation, audit, verification-first remediation, and independent completion acceptance under a bounded goal. Use when active plans must reach complete with per-plan routing and controlled implementation re-entry.
---

# Implement And Audit Until Complete

<!-- evidence-attestation-contract: external-signed-artifact; exact-run-revision-command-result-image; missing-trust-stops -->

1. 解析仓库根目录，完整读取 `.github/prompts/backend-implement-audit-until-complete.prompt.md`。
2. 将该 prompt 作为唯一闭环状态机；不得重新排序“先复审后整改”，不得丢失跨计划就绪检查、per-plan 阻断或显式 `acceptance_next_action` 路由。
3. 将 `TARGET` 保持为不可扩大的 complete goal 集合，将 `PEER_SET` 保持为完整活跃 peer 集合；计划就绪子流程固定使用 `TARGET=<goal 集合> PEER_SET=<完整 peer 集合> ADVANCE_SET=<需要就绪子集> GOAL_MODE=child STEP_MODE=single-transition MAX_CYCLES=1`，禁止嵌套多 cycle。
4. 每个 cycle 重新派生完整活跃 peer 集合；它与启动时 `PEER_SET` 不一致时安全停止并返回使用新 peer 集合、原目标子集重启的精确 handoff，禁止 readiness、implementation、audit 或 acceptance 沿过期集合继续推进。
5. 实施审计、follow-up 和完成验收必须由运行时创建真实新 task/agent，再传入其真实 `CONTEXT_REF` 与 evidence correlation `CONTEXT_ID`；不得在当前上下文只换 UUID。每个关闭 AUD 的 child 必须返回 Evidence 已由外部 trust anchor 验签且 worktree 干净的 terminal `governance_revision` 后才能继续。
6. 使用 `docs/tools/update-loop-run-state.ps1` 创建或恢复 `governance-loop-run/v1` 状态，以 generation 和 previous governance SHA 双 CAS 持久化不可变集合、cycle、每计划 fingerprint/stagnant/blocker；状态不一致时停止。计划就绪 child 使用独立且可恢复的 child run id，所有 child 严格串行。
7. 要求全部原子 skill 通过 `docs/tools/invoke-governance-transaction.ps1` 的 Git common-dir 锁、精确路径和 HEAD/shared-ref CAS 完成 open、subject/result 与 terminal commit；新记录 open 必须包含自身 `runtime_context_attestation` 文件，关闭 AUD 的 terminal 必须包含 AUD、必要索引及 `evidence.json`/`attestation.json`。禁止裸提交、遗漏 artifact 或混入并行/用户改动。
8. 一个外层 cycle 对每个计划至多推进一个持久化状态；按 canonical state fingerprint 分计划计算 stagnant counter，默认 `MAX_CYCLES=12`，并隔离阻断和恢复入口。
9. 最新 REM 为 `blocked` 时仅在恢复证据变化后新建 REM；最新 IMP 为 `partial`/`blocked` 时，当前 ready 未漂移且恢复证据变化即可新建 IMP，只有计划/peer/subject 漂移才要求新 ready revision。完成验收的 `implement` 路由一经 `implemented-by:IMP-NNNN` 消费不得重放，禁止改写终止记录或自动重试。
10. 仅当全部目标计划满足 canonical prompt 的 complete 条件时完成 persistent goal；达到本地停止条件时遵循运行时 goal 状态规则并保留精确 handoff。
11. prompt 缺失或无法读取时停止，不得凭记忆重建简化闭环。记录与最终汇报使用中文，固定状态值保持原样。
12. 每个新记录 child 必须取得仓库外签名 `runtime_context_attestation`，signed payload 绑定 exact `record_id`/`record_path`；独立审计/验收还需精确 `source_context_attestations`，且 `source_context_refs` 必须等于已验签 source ref 集合。仓库不得持有私钥，缺失 trust anchor 时安全 handoff。
13. 对关闭 AUD 的 child，验证 frontmatter `evidence_attestation`、artifact 原始 SHA256 与 run/revision/tree/argv/exit/image 绑定，并确认两个 Evidence blob 位于返回的干净 terminal revision；不满足时不得更新 loop state、进入下一阶段或完成 goal。
14. 每个创建新 AUD/REM/IMP 的原子阶段必须使用独立的逐记录 child task/context 和全局唯一 `execution_context_id`；同一 child 不得创建多份治理记录，批量只允许严格串行分派。
