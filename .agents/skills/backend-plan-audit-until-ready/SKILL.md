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
- `GOAL_MODE=standalone|child`，默认 `standalone`；只有 standalone 可以创建或完成 persistent goal。
- 分别接受 `AUDITOR`、`FOLLOW_UP_AUDITOR`、`ACCEPTANCE_AUDITOR` 和 `FOCUS`；三个审计阶段必须使用不同执行上下文，`FOCUS` 不得缩小底层 skill 的强制范围。

## 闭环

1. `GOAL_MODE=standalone` 时建立或复用匹配的 persistent goal；`child` 时复用外层目标但不得创建、完成或阻塞它。目标是最新计划验收 `acceptance_verdict: ready`，且相关 AUD/REM 链无待整改、待复审或执行中的 open AUD。
2. 将 `TARGET` 解析为不可变集合并识别阶段；优先恢复唯一且 revision 未漂移的 open AUD、待整改 AUD 或待复审 REM。stale open 必须按底层 prompt 终止为 superseded，不能永久占据队列。若源 AUD 已有关联 `status=blocked; verification=not-ready` REM，停止并报告恢复条件，不得重复创建 REM。
3. 从索引派生该计划集合当前全部 `remediation=required` AUD，包括计划审计、follow-up 和先前失败的计划验收 AUD；调用 `$backend-fix-audit-findings TARGET=<AUD 列表>`。`remediation=decision-required` 不得进入自动整改。
4. 仅当某计划在当前 revision 上不存在已关闭的当前合同计划审计，或上次审计后 plan/checklist/事实源发生漂移时，逐个调用 `$backend-plan-audit TARGET=<单一计划>`；新审计产生 finding 时返回步骤 3。禁止用共享 AUD 合并多个计划。
5. 整改完成后，把每个待复审 REM 交给不同于整改上下文的新执行上下文调用 `$backend-follow-up-audit TARGET=<单一 REM>`。审计链干净后，再把每个计划交给另一个新的执行上下文调用 `$backend-plan-acceptance-audit TARGET=<单一计划>`。必须显式传递新的 `CONTEXT_ID`；无法创建独立上下文时停止并输出精确 handoff，不得在当前上下文自我复审或填写独立声明。
6. 验收为 `ready` 时完成；验收产生 finding 时，以该验收 AUD 进入下一整改/复审周期，并在复审后重新执行独立验收，不重复创建计划审计，除非整改改变了计划 revision。一个 cycle 定义为一次“阶段解析、派生待处理队列、整改/复审、必要审计、验收”的完整尝试；只有队列状态、revision、finding 或 verdict 均未变化时才计为 stagnant cycle。

## 停止条件

达到周期上限、连续停滞、同一外部阻断连续两轮、需要接受风险/削弱测试/修改不可变记录/扩大未授权范围时停止，保留所有记录并报告决策需求。不得伪造索引状态完成目标。

所有记录和最终汇报使用中文；代码、命令、路径、ID 与固定状态值保留原样。
