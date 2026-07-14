---
status: completed
remediation_id: REM-0003
implementer: codex
scope: audit:AUD-0005
source_audits: AUD-0005
source_findings: AUD-0005-F001, AUD-0005-F002
baseline: git:4d487b4b499d1012ea551960503b75bde9c0cc94; worktree:clean
started_at: 2026-07-14T12:04:56+08:00
completed_at: 2026-07-14T12:14:03+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# AUD-0005 计划契约 validator 整改

## 对象与边界

本记录整改 `AUD-0005` 中两项 `partially-resolved` finding。范围限定为 `PLN-0005` 的 P0 deployment Evidence 边界、tracked work-package dependency DAG、配套 checklist，以及 `docs/tools/validate.ps1` 和 `docs/tools/validate.tests.ps1` 的防语义漂移门禁；不修改已关闭的 `AUD-0005`、历史 AUD 或已关闭 REM，不实现阶段五产品代码，也不自行把 source finding 标记为 resolved。

## Finding 整改矩阵

| Source finding | Root cause | Planned change | Validation | Result |
|---|---|---|---|---|
| `AUD-0005-F001` | validator 只检查 P0-22/P0-23 的必需短语，未验证 P0 条目集合，也未拒绝其他 P0 条款重新要求真实 binary、监督器、cleanup 调度、watchdog/recovery、ENOSPC 或 live profile Evidence。 | 在 plan 中增加机器可解析的 P0 deployment Evidence 单一契约；validator 解析该契约、校验 P0-1..P0-25 的完整唯一集合，并扫描所有 P0 checklist 条款及 P0 契约正文，拒绝附加 live deployment gate；增加已有条款、额外 P0-26 和 plan prose 三类 additive contradiction fixture。 | `docs/tools/validate.tests.ps1` 证明删除合法短语、改写 P0-22、在 P0-20 附加 live gate、新增 P0-26 及在 plan 附加 live gate 均被拒绝；repository validator 与 diff check 通过。 | completed locally; pending follow-up audit |
| `AUD-0005-F002` | validator 只检查固定正向行，未从 tracked table 建图，也未拒绝正文中的反向边或“不依赖”陈述。 | 从 tracked work-package 表解析完整八包 DAG，校验已知 package、精确输入边、自依赖、未知依赖、环和 Release 七包汇聚边；扫描 plan/checklist 的 arrow、depends-on、before/precedes/先于及否定依赖陈述，拒绝不存在、反向或否定 tracked edge；增加 additive contradiction fixture。 | `docs/tools/validate.tests.ps1` 证明删除 `WP-Baseline-Evidence → WP-Facts` 和附加 `WP-Baseline-Evidence may run before WP-Facts and does not depend on it` 均被拒绝；repository validator 与 diff check 通过。 | completed locally; pending follow-up audit |

## 实际变更

- `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md`：增加 `phase5-p0-deployment-evidence-contract` 机器可解析契约，固定 P0 deployment artifact 类型、真实 Evidence gate 和 P0 禁止的 live Evidence 类别。
- `docs/tools/validate.ps1`：从 tracked work-package 表解析完整 dependency DAG，验证八个 package 的精确输入、未知依赖、自依赖、环和 Release 汇聚；拒绝正文/checklist 中不存在、反向或否定 tracked edge 的显式依赖陈述。
- `docs/tools/validate.ps1`：验证 checklist 的 P0-1..P0-25 恰好各出现一次，拒绝额外 P0 项；解析 deployment contract，并扫描 P0 checklist/plan 条款中的附加 live deployment gate。
- `docs/tools/validate.tests.ps1`：合法 fixture 扩展为完整八包 DAG 和 25 个 P0 条目；新增反向 DAG 正文、附加 P0 live gate、额外 P0-26 与附加 plan live gate 的负向 fixture。
- 实际 revision：`4d487b4b499d1012ea551960503b75bde9c0cc94` 上的未提交整改 diff；未修改任何 closed AUD 或 closed REM 正文。

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；输出确认合法治理 fixture 被接受，phase-five DAG/profile 删除与附加矛盾均被拒绝。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过；验证 45 个 Markdown 文件、frontmatter、相对链接和 `git diff HEAD --check`。
- `git diff --check`：通过；仅有工作树 LF→CRLF 转换提示，无 whitespace error。

## 未完成项与剩余风险

- 两项 source finding 均已有本地实现和可证伪负向 fixture，无未完成整改项。
- 依赖正文检查针对显式 `WP-*` arrow/depends/before/否定表达，deployment 检查针对 P0 条款与冻结的 live Evidence 词汇；未来若引入全新的自然语言表达或新 work-package 契约，需同步扩展结构化 contract/validator。
- 阶段五产品 binary、migration、部署 profile 和真实环境 Evidence 仍未实施；本 REM 只修复计划契约治理门禁，不证明后续 5A-D/5B gate 已通过。
- source findings 保持原 disposition；只有新的 follow-up audit 可以确认整改有效。

## Follow-up 交接

整改完成后使用 `$backend-follow-up-audit TARGET=REM-0003` 独立复审。
