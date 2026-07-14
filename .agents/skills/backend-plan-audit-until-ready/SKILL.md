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
2. 将 `TARGET` 解析为不可变的计划 ID 集合，并根据索引识别当前阶段；已有待整改 AUD 或待复审 REM 时先恢复该队列，不得每轮无条件创建新的计划审计。
3. 从索引派生该计划集合当前全部 `remediation=required` AUD，包括计划审计、follow-up 和先前失败的计划验收 AUD；调用 `$backend-fix-audit-findings TARGET=<AUD 列表>`，再对这些 AUD 已有或新产生且 `verification=pending` 的 REM 调用 `$backend-follow-up-audit TARGET=<REM 列表>`。
4. 仅当某计划在当前 revision 上不存在已关闭的 `plan-audit/v2`，或上次审计后 plan/checklist/事实源发生漂移时，调用 `$backend-plan-audit TARGET=<计划列表>`；新审计产生 finding 时返回步骤 3。已有当前且链条干净的计划不得重复审计制造记录噪音。
5. 审计链干净后调用 `$backend-plan-acceptance-audit TARGET=<同一计划集合>`；禁止回退到子 skill 的默认全量范围。
6. 验收为 `ready` 时完成；验收产生 finding 时，以该验收 AUD 进入下一整改/复审周期，并在复审后重新执行独立验收，不重复创建计划审计，除非整改改变了计划 revision。一个 cycle 定义为一次“阶段解析、派生待处理队列、整改/复审、必要审计、验收”的完整尝试；只有队列状态、revision、finding 或 verdict 均未变化时才计为 stagnant cycle。

## 停止条件

达到周期上限、连续停滞、同一外部阻断连续两轮、需要接受风险/削弱测试/修改不可变记录/扩大未授权范围时停止，保留所有记录并报告决策需求。不得伪造索引状态完成目标。

所有记录和最终汇报使用中文；代码、命令、路径、ID 与固定状态值保留原样。
