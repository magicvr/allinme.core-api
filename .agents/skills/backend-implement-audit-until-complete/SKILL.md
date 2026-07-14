---
name: backend-implement-audit-until-complete
description: Orchestrate readiness verification, plan implementation, implementation audit, remediation, follow-up verification, and completion acceptance under a bounded persistent goal.
---

# Implement And Audit Until Complete

<!-- implementation-loop-contract: acceptance-remediation; controlled-implementation-reentry -->
<!-- fresh-plan-contract: plan-audit-before-readiness-acceptance -->

仅通过下列 skill 执行各阶段：

- `$backend-plan-acceptance-audit`
- `$backend-plan-audit-until-ready`
- `$backend-implement-plan`
- `$backend-implementation-audit`
- `$backend-fix-audit-findings`
- `$backend-follow-up-audit`
- `$backend-implementation-acceptance-audit`

## 输入

- `TARGET=active`：默认选择全部活跃且未归档计划；也接受单个或多个 `PLN-NNNN`。
- `MAX_CYCLES=3`，范围 1–10；`MAX_STAGNANT_CYCLES=2`，范围 1–3。
- 可传递 `IMPLEMENTER`、`AUDITOR`、`FOLLOW_UP_AUDITOR`、`ACCEPTANCE_AUDITOR` 和 `FOCUS`；`FOCUS` 只能增加深度，不得缩小任何强制 Control。

## 闭环

1. 仅由本外层 skill 建立或复用 persistent goal；目标是每个计划最新完成验收 `acceptance_verdict: complete`，且相关 AUD/REM/IMP 链干净。任何被调用的闭环都必须使用 child 模式，不得嵌套管理 goal。
2. 将 `TARGET` 解析为不可变集合并确定阶段；优先恢复 revision 未漂移的 open AUD、in-progress IMP 和待处理 REM；stale open 按底层 prompt 终止为 superseded。存在关联 blocked/not-ready REM 时停止并报告恢复条件，不得重复整改。
3. 无论是否已有 IMP，都先复用最新、已关闭、未漂移的 `ready` 计划验收。若当前 revision 没有已关闭的 `plan-audit/v2`，或计划审计/整改链不干净，直接调用 `$backend-plan-audit-until-ready TARGET=<单一计划> GOAL_MODE=child`；只有已存在干净的当前计划审计链但缺少 ready 验收时，才在独立上下文调用 `$backend-plan-acceptance-audit TARGET=<单一计划>`。已有 IMP 的计划还必须判断原 IMP/REM 链是否覆盖新计划范围。
4. 对“无 IMP 且 ready”或完成验收明确写 `acceptance_next_action: implement` 的计划调用 `$backend-implement-plan TARGET=<计划列表>`。任何最新 IMP 为 `in-progress` 时恢复同一 IMP；为 `partial` 或 `blocked` 时停止并报告恢复条件；为 `completed` 时禁止再次调用实施入口，除非验收/复审明确要求新的实施尝试。
5. 对 `status=completed` 且 `audit=pending`，或完成验收写 `acceptance_next_action: implementation-audit` 的 IMP，逐个在不同于 implementer 的新执行上下文调用 `$backend-implementation-audit TARGET=<单一 IMP>`；显式传递新 `CONTEXT_ID`，禁止多个 IMP 共用一个实施审计 AUD。
6. 仅从索引派生该集合 `remediation=required` 且 `acceptance_next_action: remediate`（若为完成验收）的 AUD，包括实施审计、follow-up 和可整改的失败完成验收；调用 `$backend-fix-audit-findings TARGET=<AUD 列表>`。`implementation-required` 和 `audit-required` 分别回到步骤 4/5，`decision-required` 立即停止。随后在新上下文逐个调用 `$backend-follow-up-audit TARGET=<单一 REM>`。实施 REM 必须形成线性祖先链。
7. 对链条干净且尚未 complete 的计划，逐个交给不同于 implementer、实施审计和整改/复审上下文的新执行上下文调用 `$backend-implementation-acceptance-audit TARGET=<单一计划>`。必须显式传递新的 `CONTEXT_ID`；无法隔离时停止并输出 handoff。若产生 finding，下一 cycle 从步骤 6 处理该验收 AUD，不得无条件重新调用 `$backend-implement-plan`。
8. 一个 cycle 定义为一次“阶段解析、待处理队列整改/复审、完成验收”的完整尝试；只有队列状态、revision、finding 或 verdict 均未变化时才计为 stagnant cycle。

## 停止条件

达到周期上限、连续停滞、外部阻断重复、需要接受风险/削弱测试/修改不可变记录/破坏性操作或缺少用户授权时停止。不得自动归档计划，也不得通过改索引制造完成。

所有记录和最终汇报使用中文；代码、命令、路径、ID 与固定状态值保留原样。
