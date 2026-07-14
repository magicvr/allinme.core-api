---
name: backend-implementation-acceptance-audit
description: Verify implementation completion in a separate execution context using the latest IMP and linear verified REM revision chain.
---

# Backend Implementation Acceptance Audit

1. 解析仓库根目录，并在验收前完整读取 `.github/prompts/backend-implementation-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行目标解析、独立完成验收矩阵、AUD 创建和 IMP/AUD 索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 只能增加深度。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析到计划、该计划最新 IMP 和完整审计链。没有当前有效 ready 计划验收时停止、不创建完成验收记录，并 handoff 到 `$backend-plan-audit-until-ready`。多计划调用必须拆成每个计划一份独立 AUD。
5. 必须在不同于 implementer、实施审计、整改和 follow-up 的新执行上下文运行；缺少新的 `CONTEXT_ID` 时停止并输出 handoff。`baseline` 固定治理快照，`evidence_revision` 固定 subject revision；记录上下文、线性派生的 `effective_result_revision` 和唯一 `evidence_run_id`。
6. 必须从索引派生完整计划/实施 AUD、REM 和 follow-up 链；`related_audits` 必须包含最新 ready 计划验收、最新实施审计和终端复审，`related_remediations` 必须包含影响有效实施 revision 的全部 REM，不得漏列较新的失败记录或使用不存在的状态引用。
7. 没有 IMP、IMP 未完成或缺少实施审计时仍创建结构合法的负向 AUD；没有 IMP 时使用 `related_implementations: none`、`effective_result_revision: none`，并以 `acceptance_next_action` 把队列路由到 implement、implementation-audit、remediate 或 decision，不得伪造 IMP 索引或用 REM 代替 IMP。
8. 只有全部 Control 通过、计划与实施审计链均干净且计划验收未漂移时才能写 `acceptance_verdict: complete`。
9. 预留前恢复相同计划/effective revision/治理 baseline 的唯一 open 验收；漂移时终止为 superseded 后新建。生成的记录和最终汇报必须使用中文；固定状态值保持原样。

```text
$backend-implementation-acceptance-audit
$backend-implementation-acceptance-audit TARGET=PLN-0005
```
