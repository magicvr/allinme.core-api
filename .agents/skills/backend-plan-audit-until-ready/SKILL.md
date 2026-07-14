---
name: backend-plan-audit-until-ready
description: Orchestrate set-aware plan audit, remediation, verification-first follow-up, and independent readiness acceptance under a bounded goal. Use when one or more active plans must reach ready without losing cross-plan checks or duplicating partial remediation.
---

# Plan Audit Until Ready

1. 解析仓库根目录，完整读取 `.github/prompts/backend-plan-audit-until-ready.prompt.md`。
2. 将该 prompt 作为唯一闭环状态机；不得重新排序“先复审后整改”，不得把完整计划集合缩成单计划审计而丢失跨计划检查。
3. 将 `TARGET` 作为完整 peer 集合，将 `ADVANCE_SET` 作为本轮推进子集；向计划审计传递 `PEER_SET=TARGET`，不得为未漂移的 ready peer 创建刷新 AUD。
4. `GOAL_MODE=child` 时支持 `STEP_MODE=single-transition MAX_CYCLES=1`，禁止嵌套多 cycle；默认 standalone 使用足以覆盖审计、整改、复审和验收的 `MAX_CYCLES=8`。
5. follow-up 和验收必须由运行时创建真实新 task/agent，再传入其 `CONTEXT_ID`；不得在当前上下文只换 UUID。每个 child 必须返回 clean terminal `governance_revision` 后才能继续。
6. 按计划隔离阻断和最终状态；只有所有推进目标满足 canonical prompt 的 ready 条件时才完成 standalone goal。
7. prompt 缺失或无法读取时停止，不得凭记忆重建简化闭环。记录与最终汇报使用中文，固定状态值保持原样。
