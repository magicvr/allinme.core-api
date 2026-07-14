---
status: completed
remediation_id: REM-0006
implementer: codex
scope: audit:AUD-0009
source_audits: AUD-0009
source_findings: AUD-0009-F001
baseline: git:fb787629ce892900e8fe806a6f527d8b839ee5ef; worktree:dirty (pre-existing AUD-0009 and remediation index changes)
started_at: 2026-07-14T00:00:00+08:00
completed_at: 2026-07-14T14:19:41+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# WP-Facts 精确输出整改

## 范围与边界

本记录整改 `AUD-0009-F001`：该 finding 发现受跟踪的 `WP-Facts` Evidence 单元格虽然检查了必需路径，却接受了无法识别的额外输出。已关闭的审计记录保持不变；独立验证由后续复审审计完成。

## Finding 整改矩阵

| 来源 finding | 根因 | 计划变更 | 验证方式 | 结果 |
|---|---|---|---|---|
| `AUD-0009-F001` | `Test-PhaseFiveFactsOutput` 只搜索必需子字符串，没有把输出契约解析为有界集合。 | 将 Evidence 单元格解析为规范输出 token，强制要求五个事实源及 `plan/checklist`，拒绝未知、重复或含糊 token；为每条拒绝路径增加负向 fixture。 | `docs/tools/validate.tests.ps1`；`docs/tools/validate.ps1`；仓库校验器；`git diff HEAD --check`。 | 本地已完成；等待后续复审 |

## 实施与证据

已更新 `docs/tools/validate.ps1`，将类似路径的 WP-Facts 输出分词，拒绝未知或非规范 token、重复 token，并要求全部六个规范输出。已在 `docs/tools/validate.tests.ps1` 中增加缺失、未知、重复和含糊输出的自测 fixture。本整改记录不会将任何来源审计 finding 标记为已解决。

验证证据：

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，共校验 52 个 Markdown 文件。
- `git diff HEAD --check`：通过。

## 未完成项与剩余风险

必须进行独立后续复审。本范围仅覆盖阶段五文档校验器和 fixture；产品二进制文件、迁移、文件系统和线上部署行为仍不属于该 finding 的范围。

## 复审交接

可使用 `$backend-follow-up-audit TARGET=REM-0006` 进行独立复审。本 REM 仅表示本地整改完成，不代表复审结果。
