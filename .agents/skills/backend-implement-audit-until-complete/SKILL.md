---
name: backend-implement-audit-until-complete
description: Orchestrate set-aware readiness, implementation, audit, verification-first remediation, and independent completion acceptance under a bounded goal. Use when active plans must reach complete with per-plan routing and controlled implementation re-entry.
---

# Implement And Audit Until Complete

1. 解析仓库根目录，完整读取 `.github/prompts/backend-implement-audit-until-complete.prompt.md`。
2. 将该 prompt 作为唯一闭环状态机；不得重新排序“先复审后整改”，不得丢失跨计划就绪检查、per-plan 阻断或显式 `acceptance_next_action` 路由。
3. 将 `TARGET` 保持为完整 peer 集合；计划就绪子流程固定使用 `ADVANCE_SET=<需要就绪子集> GOAL_MODE=child STEP_MODE=single-transition MAX_CYCLES=1`，禁止嵌套多 cycle。
4. 实施审计、follow-up 和完成验收必须由运行时创建真实新 task/agent，再传入其 `CONTEXT_ID`；不得在当前上下文只换 UUID。每个 child 必须返回 clean terminal `governance_revision` 后才能继续。
5. 一个外层 cycle 对每个计划至多推进一个持久化状态；默认 `MAX_CYCLES=12`，并按计划隔离阻断和恢复入口。
6. 仅当全部目标计划满足 canonical prompt 的 complete 条件时完成 persistent goal；达到本地停止条件时遵循运行时 goal 状态规则并保留精确 handoff。
7. prompt 缺失或无法读取时停止，不得凭记忆重建简化闭环。记录与最终汇报使用中文，固定状态值保持原样。
