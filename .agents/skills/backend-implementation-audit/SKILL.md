---
name: backend-implementation-audit
description: Independently audit completed implementations at their exact result revision with one resumable AUD per IMP, checking traceability, code, tests, evidence, CI, recovery, and release gates.
---

# Backend Implementation Audit

1. 解析仓库根目录，并在审计前完整读取 `.github/prompts/backend-implementation-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行 IMP 选择、实施审计矩阵、AUD 创建、finding 和索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR`、`CONTEXT_ID` 和 `FOCUS`；默认 `TARGET=pending`。`FOCUS` 只能增加深度；`PLN` 目标只能解析到最新 eligible IMP。
4. 只审计已关闭为 `completed` 且具备可复核 Evidence 的 IMP；不得把 `in-progress`、`partial` 或 `blocked` 实施视为完成。
5. 不得修改实现、plan/checklist 或 IMP 来消除 finding；整改交给 `$backend-fix-audit-findings`。
6. 必须由运行时创建不同于 implementer 的新 task/agent 并提供真实 `CONTEXT_ID`；禁止本地生成 UUID 冒充隔离。使用 `implementation-audit/v2`，令 `evidence_revision` 等于 IMP `result_revision`，并记录完整 source contexts。
7. 多 IMP 调用仅作为分派入口；每个 IMP 必须创建或恢复一份独立 AUD。revision 漂移时按 canonical prompt 终止 stale open 记录。
8. 每个失败 Control 必须关联当前 AUD finding，并保留独立验证证据。提交 open checkpoint 后再审计，关闭后提交 terminal governance commit，并返回 clean `governance_revision`。
9. 生成的审计记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。

```text
$backend-implementation-audit
$backend-implementation-audit TARGET=IMP-0001
```
