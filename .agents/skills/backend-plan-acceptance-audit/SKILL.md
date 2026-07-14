---
name: backend-plan-acceptance-audit
description: Independently verify whether every active unarchived plan, or plans selected by PLN ID/path, are ready to implement. Create indexed plan-readiness acceptance AUD records without modifying the plans.
---

# Backend Plan Acceptance Audit

1. 解析仓库根目录，并在执行任何验收前完整读取 `.github/prompts/backend-plan-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，完整执行对象解析、独立证据检查、验收矩阵、AUD 创建和索引流转。
3. 将 `$backend-plan-acceptance-audit` 后的文本解释为可选 `TARGET`、`AUDITOR` 和 `FOCUS`；默认 `TARGET=active`。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析，不得静默遗漏。
5. 不得采用已有计划审计的结论代替独立复核，也不得修改 plan/checklist 消除 finding。
6. 只有所有强制 Control 通过时才能写 `acceptance_verdict: ready`。
7. 生成的审计记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。

```text
$backend-plan-acceptance-audit
$backend-plan-acceptance-audit TARGET=PLN-0005
```
