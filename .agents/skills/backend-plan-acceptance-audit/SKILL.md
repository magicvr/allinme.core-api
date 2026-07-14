---
name: backend-plan-acceptance-audit
description: Verify plan readiness in a separate execution context, creating one resumable acceptance AUD per active or selected plan without modifying it.
---

# Backend Plan Acceptance Audit

1. 解析仓库根目录，并在执行任何验收前完整读取 `.github/prompts/backend-plan-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，完整执行对象解析、独立证据检查、验收矩阵、AUD 创建和索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 只能增加深度。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析，不得静默遗漏。多计划调用必须拆成每个计划一份独立 AUD，禁止共享一个 `acceptance_verdict`。
5. 必须由运行时创建不同于计划审计、整改和 follow-up 的新 task/agent，并提供其真实 `CONTEXT_ID`；缺失时停止，禁止本地生成 UUID 冒充隔离。记录完整上下文字段和唯一 `evidence_run_id`。
6. 必须递归派生以当前 revision-bound、`governance_contract: audit-loop/v3` 的计划审计/计划就绪验收为根的 AUD/REM/follow-up 链；排除纯实施审计链。当前 revision 没有满足 `evidence_revision` 与 `audited_subject_paths` 约束的已关闭 `plan-audit/v2` 时停止、不创建验收记录，并 handoff 到 `$backend-plan-audit-until-ready`；其他相关状态引用必须真实匹配，不得漏列或伪造。
7. 只有所有强制 Control 通过且 `PLAN_AUDIT_CHAIN_CLEAN` 通过时才能写 `acceptance_verdict: ready`。
8. `ready` 前必须独立执行至少一条不属于治理 validator 或 `git diff --check` 的 subject-specific 验证命令；仅复述历史 Evidence 不得通过。
9. `baseline` 固定包含 source records 的治理快照，`evidence_revision` 固定被验收 subject；提交 open checkpoint 后再验收，关闭后提交 terminal governance commit，并返回 clean `governance_revision`。生成的记录和最终汇报必须使用中文；固定状态值保持原样。

```text
$backend-plan-acceptance-audit
$backend-plan-acceptance-audit TARGET=PLN-0005
```
