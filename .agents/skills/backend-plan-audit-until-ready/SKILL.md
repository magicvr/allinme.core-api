---
name: backend-plan-audit-until-ready
description: Orchestrate set-aware plan audit, remediation, verification-first follow-up, and independent readiness acceptance under a bounded goal. Use when one or more active plans must reach ready without losing cross-plan checks or duplicating partial remediation.
---

# Plan Audit Until Ready

<!-- evidence-attestation-contract: external-signed-artifact; exact-run-revision-command-result-image; missing-trust-stops -->

1. 解析仓库根目录，完整读取 `.github/prompts/backend-plan-audit-until-ready.prompt.md`。
2. 将该 prompt 作为唯一闭环状态机；不得重新排序“先复审后整改”，不得把完整计划集合缩成单计划审计而丢失跨计划检查。
3. 将 `TARGET` 作为不可扩大的 goal 集合，将 `PEER_SET` 作为完整活跃 peer 集合，将 `ADVANCE_SET` 作为本轮推进子集；standalone 时 `ADVANCE_SET` 必须等于 `TARGET`，child 才允许使用真子集。向计划审计传递 `PEER_SET=<完整 PEER_SET>`，并验证每份计划审计持久化完整 peer 快照。
4. 每个 cycle 重新派生完整活跃 peer 集合；它与启动时 `PEER_SET` 不一致时安全停止并返回使用新 peer 集合、原目标子集重启的精确 handoff，禁止沿过期集合继续推进。
5. `GOAL_MODE=child` 时支持 `STEP_MODE=single-transition MAX_CYCLES=1`，禁止嵌套多 cycle；默认 standalone 使用足以覆盖审计、整改、复审和验收的 `MAX_CYCLES=8`。
6. follow-up 和验收必须由运行时创建真实新 task/agent，再传入其真实 `CONTEXT_REF` 与 evidence correlation `CONTEXT_ID`；不得在当前上下文只换 UUID。每个关闭 AUD 的 child 必须返回 Evidence 已由外部 trust anchor 验签且 worktree 干净的 terminal `governance_revision` 后才能继续；已有未漂移且 Evidence 已验签的 closed ready 直接复用，不创建重复验收 AUD。
7. 使用 `docs/tools/update-loop-run-state.ps1` 创建或恢复 `governance-loop-run/v1` 状态，以 generation 和 previous governance SHA 双 CAS 持久化不可变集合、cycle、每计划 fingerprint/stagnant/blocker；状态不一致时停止。所有 child 严格串行。
8. 要求全部原子 skill 通过 `docs/tools/invoke-governance-transaction.ps1` 的 Git common-dir 锁、精确路径和 HEAD/shared-ref CAS 完成 open、subject/result 与 terminal commit；新记录 open 必须包含自身 `runtime_context_attestation` 文件，关闭 AUD 的 terminal 必须包含 AUD、必要索引及 `evidence.json`/`attestation.json`。禁止裸提交、遗漏 artifact 或混入并行/用户改动。
9. 按 canonical state fingerprint 分计划计算 stagnant counter并隔离阻断；只有完整 `TARGET` 全部满足 ready 条件时才完成 standalone goal，child 只汇报 `ADVANCE_SET` 的本次结果。
10. 最新 REM 为 `blocked` 时，只有恢复条件或授权发生可验证变化才创建新 REM；否则保留 handoff，禁止自动重试风暴。
11. prompt 缺失或无法读取时停止，不得凭记忆重建简化闭环。记录与最终汇报使用中文，固定状态值保持原样。
12. 每个新记录 child 必须取得仓库外签名 `runtime_context_attestation`，signed payload 绑定 exact `record_id`/`record_path`；follow-up/验收还需精确 `source_context_attestations`，且 `source_context_refs` 必须等于已验签 source ref 集合。仓库不得持有私钥，缺失 trust anchor 时安全 handoff。
13. 对关闭 AUD 的 child，验证 frontmatter `evidence_attestation`、artifact 原始 SHA256 与 run/revision/tree/argv/exit/image 绑定，并确认两个 Evidence blob 位于返回的干净 terminal revision；不满足时不得更新 loop state 或派生下一 child。
14. 每个创建新 AUD/REM 的原子阶段必须使用独立的逐记录 child task/context 和全局唯一 `execution_context_id`；同一 child 不得创建多份治理记录，批量只允许严格串行分派。
