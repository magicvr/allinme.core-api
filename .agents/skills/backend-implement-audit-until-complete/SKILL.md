---
name: backend-implement-audit-until-complete
description: Orchestrate set-aware readiness, implementation, audit, verification-first remediation, and independent completion acceptance under a bounded goal. Use when active plans must reach complete with per-plan routing and controlled implementation re-entry.
---

# Implement And Audit Until Complete

1. 解析仓库根目录，完整读取 `.github/prompts/backend-implement-audit-until-complete.prompt.md`。
2. 将该 prompt 作为唯一闭环状态机；不得重新排序“先复审后整改”，不得丢失跨计划就绪检查、per-plan 阻断或显式 `acceptance_next_action` 路由。
3. 完整调用 prompt 指定的底层 skills，并显式传播 `TARGET`、cycle 参数、角色参数、`CONTEXT_ID`、`GOAL_MODE=child` 和 `FOCUS`。
4. 仅当全部目标计划满足 canonical prompt 的 complete 条件时完成 persistent goal；达到本地停止条件时遵循运行时 goal 状态规则并保留精确 handoff。
5. prompt 缺失或无法读取时停止，不得凭记忆重建简化闭环。记录与最终汇报使用中文，固定状态值保持原样。
