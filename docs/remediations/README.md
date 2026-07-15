# 整改记录管理

`records/` 保存审计 finding 的实际整改。REM 说明修复了哪些 finding、根因是什么、改了什么、如何验证，以及还剩什么风险。

## 规则

- 文件名：`REM-NNNN-YYYYMMDD-<implementer>-<scope-kind>-<subject>.md`。
- 必需字段：`status`、`remediation_id`、`implementer`、`scope`、`source_audits`、`source_findings`、`baseline`、`started_at`、`last_updated`。
- `status` 使用 `in-progress`、`completed`、`partial` 或 `blocked`。终态 REM 不再改写；继续工作创建新的 REM。
- 同一根因可以合并，但必须保留 source finding 的逐项映射；不得跨无关计划或实施链合并。
- `completed` 只表示整改者声称完成，必须由不同上下文的 follow-up AUD 验证。
- REM 不得修改 source AUD 的 finding disposition。

## 当前索引

- [`REM-0006`](./records/REM-0006-20260714-codex-audit-aud-0009-wp-facts-exact-output.md)：`status=completed`；`verification=retired-by-scope`。
- [`REM-0005`](./records/REM-0005-20260714-codex-audit-active-audits.md)：`status=completed`；`verification=historical`。
- [`REM-0004`](./records/REM-0004-20260714-codex-audit-aud-0006-contract-clause-parsers.md)：`status=completed`；`verification=retired-by-scope`。
- [`REM-0003`](./records/REM-0003-20260714-codex-audit-aud-0005-contract-validators.md)：`status=completed`；`verification=retired-by-scope`。
- [`REM-0002`](./records/REM-0002-20260714-codex-audit-aud-0004-rem-0001-follow-up.md)：`status=completed`；`verification=historical`。
- [`REM-0001`](./records/REM-0001-20260714-codex-audit-active-audits.md)：`status=completed`；`verification=historical`。

`retired-by-scope` 仅用于已删除的自然语言/专用治理 validator，不可用于产品或闭环缺陷。

模板：[templates/remediation-record.md](./templates/remediation-record.md)。整改入口：`$backend-fix-audit-findings`；独立复审：`$backend-follow-up-audit`。
