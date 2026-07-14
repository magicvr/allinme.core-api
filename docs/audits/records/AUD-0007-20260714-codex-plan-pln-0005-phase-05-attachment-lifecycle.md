---
status: closed
audit_schema: plan-audit/v2
audit_id: AUD-0007
auditor: codex
audit_type: targeted
scope: plan:PLN-0005
subject: phase-05-attachment-lifecycle
baseline: git:596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1; worktree:clean
started_at: 2026-07-14T12:44:37+08:00
completed_at: 2026-07-14T13:02:11+08:00
last_updated: 2026-07-14
related_audits: AUD-0002,AUD-0003,AUD-0004,AUD-0005,AUD-0006
related_remediations: REM-0001,REM-0002,REM-0003,REM-0004
supersedes: none
related_plans: PLN-0005
---

# PLN-0005 阶段五附件生命周期计划审计

## 目的与范围

审计 `PLN-0005` 计划及其 checklist 的正确性、完整性、可执行性，以及与路线图、事实源、当前代码和历史审计的兼容性。本记录不是全仓质量审计，也不实施 finding 整改。

## 基线与方法

- 固定基线：`main@596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1`；审计开始时工作树 clean。
- 审计对象：`PLN-0005` plan/checklist、计划与审计索引、阶段五路线图、architecture/domain/target/current API/validation、归档阶段四前置计划、相关源码/测试/migration/配置/CI、同主题 `AUD-0002..AUD-0006` 与 `REM-0001..REM-0003`。
- 内容方法：完整读取选中 plan/checklist，按 plan §3 的 41 项冻结决策、§4-§7 数据/HTTP/运行义务、§9 gate、§11 风险和 checklist 95 个父项建立双向覆盖；抽查并验证当前 v6 migration、订单幂等、applock、app/admin 生命周期、CI 与 phase-five 代码缺失基线。
- 并发变更：审计开始后出现未提交 `REM-0004` 及 validator/index diff；该整改明确以本审计记录为 pre-existing change 并未改写它。固定 baseline 不吸收这些后发变更；仅在“未执行项与剩余风险”记录其待独立 follow-up 状态。

## 历史关系

- `AUD-0002-F001/F002/F003/F004` 与 `AUD-0003-F001/F002`：`REM-0001..REM-0003` 和 `AUD-0004..AUD-0006` 已确认 P0/M1A 主循环、5B capability、ORDER_DELETE/idempotency 生命周期、architecture 两阶段边界及 tracked DAG 当前文本得到修正；本轮在 baseline 上未复现这些已解决的计划内容缺陷。
- `AUD-0006-F001/F002`：在固定 baseline 上仍是当前 validator 实现缺陷，不是本轮新计划 finding；它们不改变当前 plan/checklist 的正确语义，但会阻塞 P0 requirements validator 的可信完成。审计进行中出现的未提交 `REM-0004` 声称加入条款级 P0 scanner 与完整 dependency relation parser，状态为 `verification=pending`，不能在本记录中提前写成 resolved。
- 本轮新 finding `AUD-0007-F001` 不重复历史结论：历史审计关注部署 Evidence 循环、DAG 边和 parser；本项关注 authoritative `WP-Facts` 出口 Evidence 对五份强制事实源少计一份。

## Plan/Checklist 审计矩阵

<!-- plan-checklist-audit: PLN-0005 -->
### PLN-0005 Plan/Checklist 审计

- Plan: [阶段五：附件生命周期开发计划](../../plans/PLN-0005-phase-05-attachment-lifecycle.md)
- Checklist: [阶段五：附件生命周期执行清单](../../plans/PLN-0005-phase-05-attachment-lifecycle-checklist.md)

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| PAIRING | plan/checklist frontmatter 均为 `status: active`、`plan_id: PLN-0005`、相同 owner/日期/applies_to；文件名同号同主题，plan §9.3 与 checklist 顶部双向链接，`docs/plans/README.md` 当前索引同时列出两文件 | pass | none |
| PLAN_TO_CHECKLIST | 按 plan §3 的 41 项冻结决策、§4-§7 数据/HTTP/运行义务、§9 gate 与 §11 风险逐项抽取；checklist P0-1..P0-25、M1A/M2/M3A/5A-I/5A-D/M1B/M3B/M4/5B/R 共 95 个父项覆盖实现、失败注入、安全、并发、migration/recovery、回退、CI/平台和发布 Evidence | pass | none |
| CHECKLIST_TO_PLAN | checklist 的 `phase5_test`/四发布 tag、schema v7、五状态、三 endpoint、5A/5B 边界、ORDER_DELETE、integrity pool、锁/启动/恢复与 Evidence 约束均可追溯到 plan §2-§9 或直接事实源；未发现私自新增外部 HTTP 契约 | pass | none |
| CHECKED_EVIDENCE | 全量统计 95 个 checkbox：`[x]=0`、`[ ]=95`；checklist 顶部明确未完成父项不得勾选，plan §10 也未把未来 phase 5 Evidence 写成已通过 | not-applicable | none |
| GATE_COMPLETENESS | plan §3 第 26/31 项与 checklist P0-1 要求 architecture/domain/target API/roadmap/validation 五份外部事实源先同步，但 authoritative tracked 表 `WP-Facts` 出口只要求“四份事实源 diff”，无法唯一判定 P0 No-Go 所需 Evidence 集合 | fail | AUD-0007-F001 |
| ARCHIVE_CLOSURE | plan §9.3 与 checklist R6 均要求完成报告、未执行项/剩余风险、实际 Evidence 和用户明确确认后同步归档 plan/checklist；没有把计划完成等同于审计关闭 | pass | none |

## Findings

### AUD-0007-F001 - WP-Facts 出口少计一份强制事实源

- Affected plan: PLN-0005
- Severity: medium
- Category: 计划缺陷 / 工作分解与门禁完整性
- Evidence: plan §3 第 26 项要求 P0-1 先同步“架构、路线图、领域模型、目标 API 和验证矩阵”，第 31 项列出 domain/target API/roadmap/validation，且 plan §3 开头与 checklist P0-1 明确把 `docs/01-architecture.md` 一并列入，共五份外部事实源；但 plan tracked work-package 表的 `WP-Facts` 出口只写“四份事实源 diff、行号、独立 revision”。该表又被 plan §8 和 checklist P0-21 定义为 dependency DAG 与工作包出口的唯一事实源。
- Impact: `WP-Facts` owner 可以按表提交四份 diff 并声称出口满足，而 P0-1/No-Go 仍要求第五份；最可能被遗漏的是此前历史 finding 专门要求前移的 architecture 事务边界。P0 完成判定、输入 revision 和 reviewer 批准因此存在两种合法解释。
- Recommendation: 把 `WP-Facts` 出口改为明确列出五份外部事实源及 plan/checklist 变更，或将数量改为与 P0-1 机器可解析集合一致；为 requirements validator 增加事实源集合精确匹配和缺任一文件的负向 fixture。
- Owner: 阶段五协议 owner / domain+API reviewer / Evidence tooling owner
- Disposition: open

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：在并发 `REM-0004` 完成本地修改后通过，验证 48 个 Markdown 文件、frontmatter、相对链接和 `git diff HEAD --check`。
- 固定 baseline validator 复核：从 `git show HEAD:docs/tools/validate.ps1` 加载不可变脚本并对当前 `docs/` 显式传入 `DocsRoot`，通过，验证 48 个 Markdown 文件；证明 `AUD-0007` 的 v2 矩阵、双链接、finding 引用和 closed 元数据满足 baseline 治理合同。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：并发 `REM-0004` 工作树版本通过；这是后发整改的本地自测，不替代 follow-up audit。
- `go version`：`go1.26.0 windows/amd64`，与 plan §10 基线一致。
- `go test ./... -count=1`：通过，wall 29.9s；现有 v6 全部 package 通过。
- `go vet ./...`：通过。
- `go test -race ./... -count=1`：通过，wall 85.5s；现有共享状态基线未报告 race。
- 结构检索：当前代码无 `internal/files`、附件 route、phase-five capability tag、`cleanup-attachments`、`verify-attachments`、`StartupError` 或 `RunResult`；migration 停在 v6，`internal/applock.Acquire(path)` 仍为单模式且 `Close` 删除锁文件。这与 plan 将相关实现放入 P0/M1A 及后续的基线假设一致。

## 未执行项与剩余风险

- 未执行任何 phase-five target package、v7 migration、四发布 tag build/test/vet、真实 upload/download、文件系统故障、crash harness、Windows reparse/share-mode、Linux 监督器/调度、ENOSPC、artifact retention 或部署 profile 验证：相应实现、fixture、远端系统和 artifact 尚未交付，不能写成已满足。
- 全仓 v6 test/vet/race 通过只证明当前基线健康，不证明 v7 schema、durable recovery、跨平台文件安全或部署门禁可行。
- `AUD-0006` 的两项 validator finding 在固定 baseline 上仍等待整改验证；后发 `REM-0004` 仅为未提交、`verification=pending` 的本地声明，必须用新的 follow-up audit 独立复核。
- 当前只有一个活跃计划，没有跨计划 dependency/schema/file ownership/release boundary 冲突。若后续发现 plan-specific validator 问题扩展为通用文档治理或 CI 可靠性问题，建议另行执行 `$backend-full-audit`，本记录不声称全仓保证。

## 关闭结论

本轮计划审计发现 1 项新问题：medium 1，属于 `WP-Facts` 工作分解与 P0 gate Evidence 集合不一致。六项 checklist 审计矩阵完整，唯一 `fail` 已关联 `AUD-0007-F001`；finding disposition 为 `open`。审计记录关闭并标记 `remediation=required`，关闭表示审计过程完成，不表示 finding 已整改、`PLN-0005` 已完成或 phase-five 产品能力已验证。整改应使用 `$backend-fix-audit-findings` 创建独立 REM，复核使用新的 follow-up audit。
