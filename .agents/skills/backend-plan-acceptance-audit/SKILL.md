---
name: backend-plan-acceptance-audit
description: Verify plan readiness in a separate execution context, creating one resumable acceptance AUD per active or selected plan without modifying it.
---

# Backend Plan Acceptance Audit

1. 解析仓库根目录，并在执行任何验收前完整读取 `.github/prompts/backend-plan-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，完整执行对象解析、独立证据检查、验收矩阵、AUD 创建和索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID`、`CONTEXT_REF` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 只能增加深度。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析，不得静默遗漏。多计划调用必须拆成每个计划一份独立 AUD，禁止共享一个 `acceptance_verdict`。
5. 必须由运行时创建不同于计划审计、整改和 follow-up 的新 task/agent，并提供其真实 `CONTEXT_REF`；缺失时停止。`CONTEXT_ID` 只是 evidence correlation UUID，不能冒充 task identity。记录完整 source refs 和唯一 `evidence_run_id`。
6. 必须递归派生以当前 revision-bound、`governance_contract: audit-loop/v3` 的计划审计/计划就绪验收为根的 AUD/REM/follow-up 链；重新派生完整 active peer 集合并检查 `audited_peer_plans` 与全部 peer plan/checklist path 漂移；排除纯实施审计链。缺少当前有效 `plan-audit/v2` 时 handoff 到 `$backend-plan-audit-until-ready`。
7. 只有所有强制 Control 通过且 `PLAN_AUDIT_CHAIN_CLEAN` 通过时才能写 `acceptance_verdict: ready`。
8. `ready` 前必须通过 detached evidence runner 在 `evidence_revision` 独立执行至少一条不属于治理 validator 的 subject-specific 验证命令；仅复述历史 Evidence 或在治理 HEAD 上代跑不得通过。
9. 只有当前真实 `CONTEXT_REF` 与 open AUD 已记录 ref 相同且原 task 可恢复时才能续跑；否则以 `context-loss` supersede 旧记录并新建 AUD，禁止重绑 runtime ref。
10. `baseline` 固定包含 source records 的治理快照，`evidence_revision` 固定被验收 subject；提交 open checkpoint 后再验收，关闭后提交 terminal governance commit，并返回 clean `governance_revision`。生成的记录和最终汇报必须使用中文；固定状态值保持原样。

```text
$backend-plan-acceptance-audit
$backend-plan-acceptance-audit TARGET=PLN-0005 CONTEXT_REF=<runtime-task-ref>
```
