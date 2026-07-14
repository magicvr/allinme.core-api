---
status: closed
audit_id: AUD-0009
auditor: codex
audit_type: follow-up
scope: follow-up:REM-0005
subject: rem-0005-active-audits
baseline: git:fb787629ce892900e8fe806a6f527d8b839ee5ef; worktree:clean
started_at: 2026-07-14T13:47:32+08:00
completed_at: 2026-07-14T13:58:55+08:00
last_updated: 2026-07-14
related_audits: AUD-0007,AUD-0008
related_remediations: REM-0005
supersedes: none
related_plans: PLN-0005
---

# REM-0005 后续复审审计

## 范围与方法

本审计针对来源 finding `AUD-0007-F001`、`AUD-0008-F001` 和 `AUD-0008-F002`，对 `REM-0005` 进行独立验证。基线为当前干净的 `HEAD`；复审范围包括受跟踪的 `PLN-0005` 工作包契约、`docs/tools/validate.ps1`、其自测，以及整改和来源审计记录。产品二进制文件、迁移和线上部署行为不属于来源 finding 范围，仍未验证。

## 验证矩阵

| 来源 finding | 声称的整改 | 检查的代码/Evidence | 独立测试 | Verdict |
|---|---|---|---|---|
| `AUD-0007-F001` | 列出五个外部事实源及 plan/checklist 变更，并校验输出契约。 | `PLN-0005` 的 `WP-Facts` 行；`docs/tools/validate.ps1` 中的 `Test-PhaseFiveFactsOutput`；`docs/tools/validate.tests.ps1` 中缺失 architecture 的 fixture。 | 增加无法识别的 `docs/99-untracked-fact.md` 输出的临时 fixture 后，校验器仍以 0 退出；必需事实源的负向 fixture 仍按预期失败。 | `partially-resolved`；必需事实源已固定，但声称的精确输出集合尚未强制执行。见 `AUD-0009-F001`。 |
| `AUD-0008-F001` | 按子句评估 live-Evidence 极性，包括英文和中文转折边界及重叠处理。 | `Get-PhaseFiveStatementClauses`、`Get-PhaseFiveLiveEvidencePattern` 和 `Test-PhaseFiveDeploymentClauses`；整改中的双向 fixture。 | 在延期与 live supervisor 义务之间使用实际中文转折 `但是` 的临时 fixture 被拒绝，并返回 `P0 deployment evidence clause must stop at contract fixtures`。 | `resolved` |
| `AUD-0008-F002` | 将拒绝豁免限定到独立子句，识别 `depends upon`，并对未知多工作包连接词 fail closed。 | `Get-PhaseFiveDependencyClauses` 和 `Test-PhaseFiveDependencyStatements`；拒绝污染及 `depends upon` fixture。 | 使用未列出的连接词 `WP-Release is coupled to WP-Unknown` 的临时 fixture 被拒绝，并返回 `dependency statement could not be fully parsed`；声明的 `depends upon` 和拒绝污染检查也通过。 | `resolved` |

## Findings

### AUD-0009-F001 - WP-Facts 接受无法识别的额外输出

- 映射至：`AUD-0007-F001`
- 严重度：medium
- Evidence：`Test-PhaseFiveFactsOutput` 检查 `WP-Facts` Evidence 单元格中是否出现每个必需路径，但没有将单元格解析为允许列表，也没有拒绝额外路径。增加 `docs/99-untracked-fact.md` 的独立临时 fixture 仍以 0 退出并通过校验器。
- 影响：工作包可以增加未经复核的事实源，同时仍满足校验器。五个 P0-1 必需事实源不能省略，但整改所声称的输出契约并不精确。
- 建议：将 `WP-Facts` Evidence 单元格解析为规范 token，拒绝五个必需事实源及明确允许的 plan/checklist 差异 token 之外的任何输出；为未知额外事实源以及重复或含糊的事实源拼写增加负向 fixture。
- 负责人：阶段五协议负责人 / Evidence 工具负责人
- Disposition：open

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，共校验 50 个 Markdown 文件。
- `git diff HEAD --check`：通过仓库校验器。
- 独立的中文同子句 fixture 和替代未知连接词 fixture 均产生预期的非零校验结果。

## 未完成项与剩余风险

`AUD-0009-F001` 仍为 open。当前整改队列是精确输出契约缺口；本记录不对阶段五二进制文件、迁移、文件系统行为或线上部署 profile 作任何保证。

## 关闭结论

`AUD-0009` 作为审计流程已关闭，但其新 finding 需要整改。`REM-0005` 已部分验证。`AUD-0007` 因新 finding 继续跟踪；`AUD-0008` 已完整验证。后续整改和复审审计必须处理精确的 `WP-Facts` 输出集合，且不得向任一已关闭的来源审计追加内容。
