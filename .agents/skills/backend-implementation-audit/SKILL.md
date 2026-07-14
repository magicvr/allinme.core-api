---
name: backend-implementation-audit
description: Audit completed implementation records selected from pending IMP entries or by IMP/PLN ID, checking plan traceability, code, tests, evidence, CI, recovery, and release gates.
---

# Backend Implementation Audit

1. 解析仓库根目录，并在审计前完整读取 `.github/prompts/backend-implementation-audit.prompt.md`。
2. 将该 prompt 视为唯一规范正文，执行 IMP 选择、实施审计矩阵、AUD 创建、finding 和索引流转。
3. 将调用文本解释为可选 `TARGET`、`AUDITOR` 和 `FOCUS`；默认 `TARGET=pending`。
4. 只审计已关闭为 `completed` 且具备可复核 Evidence 的 IMP；不得把 in-progress/blocked 实施视为完成。
5. 不得修改实现、plan/checklist 或 IMP 来消除 finding；整改交给 `$backend-fix-audit-findings`。
6. 每个失败 Control 必须关联当前 AUD finding，并保留独立验证证据。
7. 生成的审计记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。

```text
$backend-implementation-audit
$backend-implementation-audit TARGET=IMP-0001
```
