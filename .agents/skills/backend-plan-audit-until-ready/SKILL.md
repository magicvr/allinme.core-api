---
name: backend-plan-audit-until-ready
description: Orchestrate plan audit, remediation, follow-up verification, and independent plan-readiness acceptance under a bounded persistent goal until selected plans are ready to implement.
---

# Plan Audit Until Ready

仅通过下列现有 skill 执行各阶段，不复制或缩减其规范：

- `$backend-plan-audit`
- `$backend-fix-audit-findings`
- `$backend-follow-up-audit`
- `$backend-plan-acceptance-audit`

## 输入

- `TARGET=active`：默认选择全部活跃且未归档计划；也接受单个或多个 `PLN-NNNN`。
- `MAX_CYCLES=3`，范围 1–10；`MAX_STAGNANT_CYCLES=2`，范围 1–3。
- 可传递 `AUDITOR` 和 `FOCUS`，但不得缩小底层 skill 的强制范围。

## 闭环

1. 建立或复用匹配的 persistent goal；目标是最新计划验收 `acceptance_verdict: ready`，且相关 AUD/REM 链无待整改或待复审项。
2. 将 `TARGET` 解析为不可变的计划 ID 集合；调用 `$backend-plan-audit TARGET=<该集合>`，只读取该集合产生的 AUD 和索引。
3. 只对该计划集合关联且当前 `remediation=required` 的 AUD 调用 `$backend-fix-audit-findings TARGET=<AUD 列表>`，再只对这些整改产生且属于该集合的 REM 调用 `$backend-follow-up-audit TARGET=<REM 列表>`。
4. 审计链干净后调用 `$backend-plan-acceptance-audit TARGET=<同一计划集合>`；禁止回退到子 skill 的默认全量范围。
5. 验收为 `ready` 时完成；验收产生 finding 时，以该验收 AUD 进入下一整改/复审周期，并在复审后重新执行独立验收。

## 停止条件

达到周期上限、连续停滞、同一外部阻断连续两轮、需要接受风险/削弱测试/修改不可变记录/扩大未授权范围时停止，保留所有记录并报告决策需求。不得伪造索引状态完成目标。

所有记录和最终汇报使用中文；代码、命令、路径、ID 与固定状态值保留原样。
