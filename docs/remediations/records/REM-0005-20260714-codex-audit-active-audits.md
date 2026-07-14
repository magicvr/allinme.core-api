---
status: completed
remediation_id: REM-0005
implementer: codex
scope: audit:AUD-0007,AUD-0008
source_audits: AUD-0007, AUD-0008
source_findings: AUD-0007-F001, AUD-0008-F001, AUD-0008-F002
baseline: git:596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1; worktree:dirty (pre-existing staged AUD-0007, AUD-0008, REM-0004, audit/remediation index, and validator changes)
started_at: 2026-07-14T13:24:27+08:00
completed_at: 2026-07-14T13:31:38+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# 当前审计整改

## 范围与边界

本记录整改当前索引中标记为 `remediation=required` 的全部审计：`AUD-0007` 和 `AUD-0008`。范围包括 `WP-Facts` 输出契约以及阶段五治理校验器和自测。已关闭的审计记录及此前已关闭的整改记录保持不变，只有后续复审审计可以验证这些 finding 是否已解决。

## Finding 整改矩阵

| 来源 finding | 根因 | 计划变更 | 验证方式 | 结果 |
|---|---|---|---|---|
| `AUD-0007-F001` | 权威 `WP-Facts` 行声称输出包含四个外部事实源差异，但 P0-1 要求五个（包括 architecture）；校验器没有固定精确输出集合。 | 更新受跟踪行，列出五个外部事实源以及 plan/checklist 变更，并增加精确输出校验契约。 | 仓库校验器，以及移除 `docs/01-architecture.md` 的负向 fixture。 | 本地已完成；等待后续复审 |
| `AUD-0008-F001` | 部署证据扫描在出现任何延期标记时豁免整条子句，导致同一未拆分子句中的后续义务可能被掩盖。 | 增加按子句评估 live-Evidence 极性的逻辑，覆盖英文和中文转折边界，以及否定义务的重叠处理。 | 校验器自测覆盖延期到义务、义务到延期和中文转折 fixture。 | 本地已完成；等待后续复审 |
| `AUD-0008-F002` | 拒绝措辞会豁免整行，关系语法也遗漏了 `depends upon` 等常见变体。 | 将拒绝豁免限定在独立子句，识别 `depends upon`，并对无法识别的多工作包连接词 fail closed。 | 校验器自测覆盖拒绝行污染、`depends upon` 和未知连接词。 | 本地已完成；等待后续复审 |

## 实际变更

已更新 `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md`、`docs/tools/validate.ps1` 和 `docs/tools/validate.tests.ps1`。新增 `WP-Facts` 事实源契约及针对性的负向 fixture，同时保留所有已关闭审计和此前整改记录。

## 验证结果

`powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1` 通过。`powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1` 通过，共校验 50 个 Markdown 文件。`git diff HEAD --check` 已作为仓库校验的一部分通过。

## 未完成项与剩余风险

独立复审仍未完成。产品代码、阶段五二进制文件、迁移、文件系统和线上部署行为不属于本次文档治理 finding 的范围。对于当前 fixture 未覆盖的未来语法，解析器仍有意保持保守策略。

## 复审交接

可使用 `$backend-follow-up-audit TARGET=REM-0005` 进行独立复审。本 REM 仅表示本地整改完成，不代表复审结果。
