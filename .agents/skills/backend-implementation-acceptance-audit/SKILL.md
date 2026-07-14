---
name: backend-implementation-acceptance-audit
description: Independently verify whether active unarchived plans, or selected PLN/IMP targets, are fully implemented and ready for explicit user-approved archival.
---

# Backend Implementation Acceptance Audit

1. 解析仓库根目录，并在验收前完整读取 `.github/prompts/backend-implementation-acceptance-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行目标解析、独立完成验收矩阵、AUD 创建和 IMP/AUD 索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR` 和 `FOCUS`；默认 `TARGET=active`。
4. 无参数时验收全部活跃且未归档计划；显式目标必须逐个解析到计划、IMP 和完整审计链。
5. 不得采用 IMP 或既有审计的完成声明代替独立复核，不得自动归档计划；必须记录 `independence_basis` 和干净的 `evidence_revision`。
6. 只有全部 Control 通过、计划与实施审计链均干净且计划验收未漂移时才能写 `acceptance_verdict: complete`。
7. 生成的审计记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。

```text
$backend-implementation-acceptance-audit
$backend-implementation-acceptance-audit TARGET=PLN-0005
```
