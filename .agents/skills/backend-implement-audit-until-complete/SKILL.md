---
name: backend-implement-audit-until-complete
description: Orchestrate readiness verification, plan implementation, implementation audit, remediation, follow-up verification, and completion acceptance under a bounded persistent goal.
---

# Implement And Audit Until Complete

仅通过下列 skill 执行各阶段：

- `$backend-plan-acceptance-audit`
- `$backend-implement-plan`
- `$backend-implementation-audit`
- `$backend-fix-audit-findings`
- `$backend-follow-up-audit`
- `$backend-implementation-acceptance-audit`

## 输入

- `TARGET=active`：默认选择全部活跃且未归档计划；也接受单个或多个 `PLN-NNNN`。
- `MAX_CYCLES=3`，范围 1–10；`MAX_STAGNANT_CYCLES=2`，范围 1–3。
- 可传递 `IMPLEMENTER`、`AUDITOR` 和 `FOCUS`。

## 闭环

1. 建立或复用匹配的 persistent goal；目标是每个计划最新完成验收 `acceptance_verdict: complete`，且相关 AUD/REM/IMP 链干净。
2. 将 `TARGET` 解析为不可变的计划 ID 集合；先调用 `$backend-plan-acceptance-audit TARGET=<该集合>`。非 `ready` 时停止，要求先运行 `$backend-plan-audit-until-ready TARGET=<该集合>`，不得在实施闭环中绕过或暗中修正计划。
3. 调用 `$backend-implement-plan TARGET=<该集合>`。任何 IMP 为 `partial` 或 `blocked` 时停止并报告恢复条件。
4. 仅对该集合产生的 completed IMP 调用 `$backend-implementation-audit TARGET=<IMP 列表>`。
5. 仅对这些实施 AUD 的 `remediation=required` 队列执行 `$backend-fix-audit-findings TARGET=<AUD 列表>` 和 `$backend-follow-up-audit TARGET=<REM 列表>`，直到该集合链条干净。
6. 调用 `$backend-implementation-acceptance-audit TARGET=<同一计划集合>`。验收为 `complete` 时完成；若产生 finding，则在剩余周期内只对该集合整改、复审并重新验收。

## 停止条件

达到周期上限、连续停滞、外部阻断重复、需要接受风险/削弱测试/修改不可变记录/破坏性操作或缺少用户授权时停止。不得自动归档计划，也不得通过改索引制造完成。

所有记录和最终汇报使用中文；代码、命令、路径、ID 与固定状态值保留原样。
