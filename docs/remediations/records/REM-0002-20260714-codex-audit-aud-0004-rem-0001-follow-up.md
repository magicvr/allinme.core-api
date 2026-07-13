---
status: completed
remediation_id: REM-0002
implementer: codex
scope: audit:AUD-0004
source_audits: AUD-0004
source_findings: AUD-0004-F001, AUD-0004-F002
baseline: git:00ff9cbe9034efdd4b9c46700d39dfaa8435ed31; worktree:clean
started_at: 2026-07-14T05:03:56+08:00
completed_at: 2026-07-14T05:10:21+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# AUD-0004 剩余计划缺陷整改

## 对象与边界

本记录整改 `AUD-0004` 中两项 `partially-resolved` finding。范围限定为 `PLN-0005` 的 P0 部署 Evidence 边界、tracked work-package dependency DAG、配套 checklist 和用于防止相同语义漂移的文档治理验证；不修改已关闭的 `AUD-0004` 或 `REM-0001`，不实现阶段五产品代码，也不把 finding 自行标记为 resolved。

## Finding 整改矩阵

| Source finding | Root cause | Implemented change | Validation | Result |
|---|---|---|---|---|
| `AUD-0004-F001` | P0-22 仍要求实测完整单机部署 profile，P0-23 又要求其真实 run；真实 supervisor、cleanup、watchdog/recovery 与阶段五 binary 只能在 M1A/5A-D/5B 产生，继续形成 P0→M1A 循环门禁。 | P0-22 改为冻结目标 profile、环境 owner、监督器/timer 配置模板、命令/退出码/告警 acceptance、恢复/ENOSPC 演练步骤并只生成 `artifactKind=contract-fixture`；P0-23 明确真实 profile、调度、watchdog/recovery、ENOSPC 与部署 Evidence 只由 5A-D/5B gate 产生。 | plan/checklist 边界检索；治理 validator 正反 fixture；`docs/tools/validate.ps1`；`docs/tools/validate.tests.ps1`；`git diff HEAD --check`。 | completed locally; pending follow-up audit |
| `AUD-0004-F002` | tracked 表漏写 `WP-Baseline-Evidence` 对 `WP-Facts` 的依赖，但正文声明该边存在；所谓唯一 DAG 同时存在两种执行顺序，validator 也没有拒绝该漂移。 | 唯一 tracked 表的 `WP-Baseline-Evidence` 输入加入 `WP-Facts`；P0-21 明确 requirements validator 必须拒绝缺失该边；通用 docs validator 和负向 fixture 同步覆盖表/正文/清单漂移。 | 同上；表与正文解析结果一致，缺失边 fixture 被拒绝。 | completed locally; pending follow-up audit |

## 实际变更

- `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md`：将 P0-22/P0-23 Evidence 边界收窄为 profile contract fixture，把真实监督器、cleanup 调度、watchdog/recovery、ENOSPC 与部署 run 留给 5A-D/5B；`WP-Baseline-Evidence` 明确依赖 `WP-Facts`。
- `docs/plans/PLN-0005-phase-05-attachment-lifecycle-checklist.md`：同步 P0-21/22/23 的可执行验收条件，明确 live profile gate 不属于 P0。
- `docs/tools/validate.ps1`：增加阶段五 DAG 表/正文一致性、P0-21 缺边拒绝和 P0-22/P0-23 contract-fixture 边界检查。
- `docs/tools/validate.tests.ps1`：增加合法 phase-five fixture，并证明缺失 `WP-Baseline-Evidence → WP-Facts` 与恢复 live P0 profile gate 时 validator 失败。
- 实际 revision：`00ff9cbe9034efdd4b9c46700d39dfaa8435ed31` 上的未提交整改 diff；未修改任何 closed AUD 或 closed REM 正文。

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，验证 43 个 Markdown 文件、frontmatter、相对链接、阶段五 contract 和 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；合法治理 fixture 被接受，缺失阶段五 DAG edge、live P0 profile gate、不完整 checklist 矩阵、未索引 audit、断链、孤立 plan 和不完整 closed audit 均被拒绝。
- `git diff HEAD --check`：通过；仅有 LF→CRLF 工作树转换警告，无 whitespace error。

## 未完成项与剩余风险

- 阶段五 binary、migration、部署 profile 和真实环境 Evidence 尚不存在；本整改只消除计划门禁循环与 dependency DAG 漂移，不证明未来 5A-D/5B deployment gate 已通过。
- source findings 保持原 disposition；只有新的 follow-up audit 可以确认整改有效。

## Follow-up 交接

完成后使用 `$backend-follow-up-audit TARGET=REM-0002` 独立复审。
