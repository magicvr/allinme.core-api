---
status: closed
audit_id: AUD-0001
auditor: OpenAI Codex
audit_type: governance
scope: repository:allinme.core-api/docs
subject: docs-governance
baseline: git:a88a246523bd1a04f1a82834e1d4d4457fcbbb4a; worktree:clean
started_at: 2026-07-14T03:05:21+08:00
completed_at: 2026-07-14T03:17:37+08:00
last_updated: 2026-07-14
related_audits: none
supersedes: none
related_plans: PLN-0001, PLN-0002, PLN-0003, PLN-0004, PLN-0005
---

# 文档治理结构专项审计

## 目的与范围

检查 `docs` 中实施计划、checklist、审计记录、归档索引、验证脚本和全量审计 prompt 的职责边界，重点验证多轮审计是否能够保留历史结论并解释意见变化。

## 基线与方法

- 基线：`main@a88a246`，开始时工作树干净。
- 检查范围：`docs/`、`.github/prompts/backend-full-audit-cycle.prompt.md`、`.github/workflows/ci.yml` 及所有相关链接。
- 方法：目录盘点、frontmatter 和命名检查、全仓引用检索、归档内容抽样、验证脚本与 CI 调用链检查。

## 历史关系

这是新审计账本的首条记录。旧 `docs/audit/archived/` 中的文件实际是阶段实施 plan/checklist，不是独立审计报告，因此作为 `PLN-0001` 至 `PLN-0004` 迁移，不追认或伪造历史审计记录。

## Findings

### AUD-0001-F001 - 实施计划与审计记录共用目录和生命周期

- Severity: high
- Evidence: 旧 `docs/audit/` 同时保存活跃 plan/checklist、归档 plan/checklist、验证脚本和审计规则；历史目录没有独立 review 记录。
- Impact: 多轮审计无法稳定保留审计者、对象、基线和历史关系，容易重复审计或产生无法解释的相反意见。
- Recommendation: 将实施文件迁入独立 `plans/`，建立只追加的 `audits/records/`，并分别使用 `PLN` / `AUD` ID。
- Owner: 后端团队
- Disposition: resolved。现有阶段文件已迁入 `docs/plans/`，计划与审计分别使用 `PLN` / `AUD` ID；验证脚本和 Evidence 也已移出审计目录。

### AUD-0001-F002 - 审计归档采用移动和覆盖式索引

- Severity: medium
- Evidence: 旧全量审计 prompt 要求将 review/checklist/plan 从活跃区移动到 archived，且只有发现新问题时才创建记录。
- Impact: 审计路径不稳定，无发现的审计完全没有记录，后续无法证明审计发生过或核对当时基线。
- Recommendation: 每次正式审计开始即创建记录；关闭后保持原路径不动，无发现也保存验证与剩余风险。
- Owner: 后端团队
- Disposition: resolved。新审计 prompt 要求每轮开始即创建记录，无 finding 也保留；审计关闭后不移动，意见变化通过新审计关联。

### AUD-0001-F003 - 自动化未强制计划与审计的命名和元数据

- Severity: medium
- Evidence: 旧验证器只检查通用 frontmatter 和相对链接，不校验 plan/checklist 配对、稳定 ID、审计者、范围、基线或关闭时间。
- Impact: 即使文档说明存在，后续新增文件仍可能回到混用、缺字段或无法关联的状态。
- Recommendation: 在文档验证器和自测中加入目录专属规则，并同步 CI、总纲和审计 prompt。
- Owner: 后端团队
- Disposition: resolved。验证器已强制命名、frontmatter、plan/checklist 配对、审计日期/范围、关闭时间和已提交 closed 审计不可修改或移动。

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，验证 35 个 Markdown 文件的 frontmatter、相对链接、计划/审计治理规则和 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；合法治理 fixture 被接受，缺失链接、孤立 plan 和缺少关闭时间的 closed audit 被拒绝。
- `git diff --check`：通过。
- 全仓检索确认活跃规则、CI、路线图和阶段五计划不再引用旧 `docs/audit` 路径；历史 checklist 中作为当时执行事实保存的旧命令文本未被改写。

## 未执行项与剩余风险

- 未运行 Go test/vet/race：本次只调整文档、PowerShell 文档验证器和 CI 中的脚本路径，没有修改 Go 源码或依赖。
- 未执行远端 CI：当前工作树尚未提交；CI 已改为串行调用新脚本路径。
- 历史 plan/checklist 不是历史审计报告，因此没有补造审计者或 findings。审计追溯从 `AUD-0001` 开始，这是刻意保留的真实性边界。

## 关闭结论

三个 finding 均已解决。计划、审计、Evidence 和工具具有独立目录与编号空间；无 finding 审计、历史关系、矛盾意见说明和 closed 记录不可变规则已经写入文档、prompt、验证器和 CI。
