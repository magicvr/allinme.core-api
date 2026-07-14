---
name: backend-plan-audit-until-ready
description: Orchestrate set-aware plan audit, remediation, verification-first follow-up, and independent readiness acceptance under a bounded goal. Use when one or more active plans must reach ready without losing cross-plan checks or duplicating partial remediation.
---

# Plan Audit Until Ready

1. 解析仓库根目录，完整读取 `.github/prompts/backend-plan-audit-until-ready.prompt.md`。
2. 将该 prompt 作为唯一闭环状态机；不得重新排序“先复审后整改”，不得把完整计划集合缩成单计划审计而丢失跨计划检查。
3. 完整调用 prompt 指定的底层 skills，并显式传播 `TARGET`、cycle 参数、`GOAL_MODE`、角色参数、`CONTEXT_ID` 和 `FOCUS`。
4. 按计划隔离阻断和最终状态；只有所有目标计划满足 canonical prompt 的 ready 条件时才完成 standalone goal。
5. prompt 缺失或无法读取时停止，不得凭记忆重建简化闭环。记录与最终汇报使用中文，固定状态值保持原样。
