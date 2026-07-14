---
name: backend-implement-audit-until-complete
description: Orchestrate readiness verification, plan implementation, implementation audit, remediation, follow-up verification, and completion acceptance under a bounded persistent goal.
---

# Implement And Audit Until Complete

<!-- implementation-loop-contract: acceptance-remediation; no-implementation-restart -->

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
- 可传递 `IMPLEMENTER`、`AUDITOR` 和 `FOCUS`。

## 闭环

1. 建立或复用匹配的 persistent goal；目标是每个计划最新完成验收 `acceptance_verdict: complete`，且相关 AUD/REM/IMP 链干净。
2. 将 `TARGET` 解析为不可变的计划 ID 集合，并为每个计划根据索引确定当前阶段；不得每轮无条件从实施入口重新开始。
3. 对尚无实施尝试的计划，逐个调用 `$backend-plan-acceptance-audit TARGET=<单一计划>`；非 `ready` 时调用 `$backend-plan-audit-until-ready TARGET=<单一计划>`。已有 IMP 的计划必须验证其记录的 ready 验收与当前 plan/checklist revision 是否仍有效；无漂移时不得为了刷新门禁重复创建验收，有漂移时必须先调用计划审计闭环重新取得单计划 ready 验收，再判断原 IMP/REM 链是否仍覆盖新计划范围。
4. 仅对“无 IMP 且 ready”的计划调用 `$backend-implement-plan TARGET=<计划列表>`。任何最新 IMP 为 `in-progress` 时恢复同一 IMP；为 `partial` 或 `blocked` 时停止并报告恢复条件；为 `completed` 时禁止再次调用实施入口，除非后续复审明确要求新的实施尝试。
5. 仅对 `status=completed` 且 `audit=pending` 的 IMP 调用 `$backend-implementation-audit TARGET=<IMP 列表>`。
6. 从索引派生该集合全部 `remediation=required` AUD，包括实施审计、follow-up 和失败的实施完成验收 AUD；调用 `$backend-fix-audit-findings TARGET=<AUD 列表>`，再对关联且 `verification=pending` 的 REM 调用 `$backend-follow-up-audit TARGET=<REM 列表>`，直到该集合链条干净。影响实施结果的已验证 REM 通过其 `result_revision` 延伸原 IMP 的 effective revision 链。
7. 对链条干净且尚未 complete 的计划，逐个调用 `$backend-implementation-acceptance-audit TARGET=<单一计划>`。验收必须选择该计划最新 IMP，并由 IMP 和已验证 REM 派生 `effective_result_revision`。若产生 finding，下一 cycle 从步骤 6 处理该验收 AUD，不得无条件重新调用 `$backend-implement-plan`；只有整改超出原 finding、计划范围改变或复审明确判定原 IMP 不可继续时，才创建新 IMP。
8. 一个 cycle 定义为一次“阶段解析、待处理队列整改/复审、完成验收”的完整尝试；只有队列状态、revision、finding 或 verdict 均未变化时才计为 stagnant cycle。

## 停止条件

达到周期上限、连续停滞、外部阻断重复、需要接受风险/削弱测试/修改不可变记录/破坏性操作或缺少用户授权时停止。不得自动归档计划，也不得通过改索引制造完成。

所有记录和最终汇报使用中文；代码、命令、路径、ID 与固定状态值保留原样。
