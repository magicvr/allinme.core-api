---
name: backend-implementation-acceptance-audit
description: Verify implementation completion in a separate execution context using the latest IMP and linear verified REM revision chain.
---

# Backend Implementation Acceptance Audit

1. 解析仓库根目录，并在验收前完整读取 `.github/prompts/backend-implementation-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行目标解析、独立完成验收矩阵、AUD 创建和 IMP/AUD 索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 只能增加深度。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析到计划、该计划最新 IMP 和完整审计链。多计划调用必须拆成每个计划一份独立 AUD。
5. 必须在不同于 implementer、实施审计、整改和 follow-up 的新执行上下文运行；缺少新的 `CONTEXT_ID` 时停止并输出 handoff。记录 `execution_context_id`、`source_context_ids`、`independence_basis: separate-context`、`evidence_revision`、线性派生的 `effective_result_revision` 和唯一 `evidence_run_id`。
6. 必须从索引派生完整计划/实施 AUD、REM 和 follow-up 链；`related_audits` 必须包含最新 ready 计划验收、最新实施审计和终端复审，`related_remediations` 必须包含影响有效实施 revision 的全部 REM，不得漏列较新的失败记录或使用不存在的状态引用。
7. 只有全部 Control 通过、计划与实施审计链均干净且计划验收未漂移时才能写 `acceptance_verdict: complete`。
8. 预留前恢复相同计划/effective revision 的唯一 open 验收，不得重复创建。生成的记录和最终汇报必须使用中文；固定状态值保持原样。

```text
$backend-implementation-acceptance-audit
$backend-implementation-acceptance-audit TARGET=PLN-0005
```
