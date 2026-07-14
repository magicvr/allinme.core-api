---
name: backend-plan-acceptance-audit
description: Verify plan readiness in a separate execution context, creating one resumable acceptance AUD per active or selected plan without modifying it.
---

# Backend Plan Acceptance Audit

1. 解析仓库根目录，并在执行任何验收前完整读取 `.github/prompts/backend-plan-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，完整执行对象解析、独立证据检查、验收矩阵、AUD 创建和索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 只能增加深度。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析，不得静默遗漏。多计划调用必须拆成每个计划一份独立 AUD，禁止共享一个 `acceptance_verdict`。
5. 必须在不同于计划审计、整改和 follow-up 的新执行上下文中运行；缺少新的 `CONTEXT_ID` 时停止并输出 handoff。不得采用既有结论代替复核；必须记录 `execution_context_id`、`source_context_ids`、`independence_basis: separate-context`、`evidence_revision` 和唯一 `evidence_run_id`。
6. 必须递归派生以计划审计/计划就绪验收为根的 AUD/REM/follow-up 链；排除纯实施审计链。当前 revision 没有已关闭 `plan-audit/v2` 时停止、不创建验收记录，并 handoff 到 `$backend-plan-audit-until-ready`；其他相关状态引用必须真实匹配，不得漏列或伪造。
7. 只有所有强制 Control 通过且 `PLAN_AUDIT_CHAIN_CLEAN` 通过时才能写 `acceptance_verdict: ready`。
8. `baseline` 固定包含 source records 的治理快照，`evidence_revision` 固定被验收 subject；漂移时把旧 open 验收终止为 superseded 后新建。生成的记录和最终汇报必须使用中文；固定状态值保持原样。

```text
$backend-plan-acceptance-audit
$backend-plan-acceptance-audit TARGET=PLN-0005
```
