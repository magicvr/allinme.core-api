# 整改记录管理

`remediations/records/` 保存审计 finding 的实际整改记录。整改记录连接不可变的源审计与后续 follow-up audit：它说明针对哪些 findings 做了什么、在哪个 baseline 上实施、执行了哪些验证，以及哪些事项仍未完成。

## 标识与命名

- 整改 ID：`REM-NNNN`，扫描全部 records 后全局递增，永不复用。
- 文件名：`REM-NNNN-YYYYMMDD-<implementer>-<scope-kind>-<subject>.md`。
- `<scope-kind>` 使用 `audit`、`repository`、`plan` 或 `feature`。
- 日期是整改开始日；跨日通过 `started_at` / `completed_at` 表示，不重命名文件。

## 必需元数据

```yaml
status: in-progress
remediation_schema: remediation/v2
governance_contract: audit-loop/v3
remediation_id: REM-0001
implementer: codex
execution_context_id: 00000000-0000-4000-8000-000000000000
runtime_context_ref: runtime-unavailable
scope: audit:AUD-0002,AUD-0003
source_audits: AUD-0002, AUD-0003
source_findings: AUD-0002-F001, AUD-0003-F001
baseline: git:<full-sha>; worktree:clean
result_revision: pending
parent_result_revision: none
affects_implementation: false
related_implementations: none
started_at: 2026-07-14T04:00:00+08:00
completed_at: pending
last_updated: 2026-07-14
related_plans: PLN-0005
```

`status` 使用：

- `in-progress`：正在实施，可以继续更新。
- `completed`：所有选中 findings 都已声称完成，等待独立复审。
- `partial`：只完成部分 findings，必须逐项说明。
- `blocked`：因外部依赖、权限或停止条件无法继续。

`completed`、`partial` 或 `blocked` 的整改记录视为关闭且不可改写；继续整改创建新的 REM。

## 整改内容

整改记录至少包含：

1. source audits/findings 与当前 baseline；
2. finding 到 root cause、修改、测试和结果的逐项映射；
3. 修改 subject 前提交 in-progress REM/索引的 open checkpoint；实际代码/文档/计划/CI 变更形成完整 `result_revision` 纯 subject commit；最终 REM 状态与索引流转由 terminal governance commit 固化，并以干净 `governance_revision` 交给 follow-up；
4. 命令、Evidence、未执行项、失败和剩余风险；
5. 是否影响已有 IMP、关联哪些 implementation，以及是否已具备 follow-up audit 条件。

`affects_implementation: true` 表示 REM 修改了产品代码、测试、migration、运行配置或发布 artifact。此时必须列出 `related_implementations`；REM 通过 follow-up 后，其 `result_revision` 进入对应 IMP 的 effective revision 链。历史 remediation/v1 记录保持不可变，不补写这些字段。

新合同 REM 不得跨计划或跨 IMP 合并。实施 REM 的 `parent_result_revision` 必须等于创建时的 effective 链尾，关闭时 `result_revision` 必须为其 Git 后代；并行分叉必须先合并，才能进入复审。

follow-up 的 `baseline` 必须是已经包含 completed/partial REM 和索引状态的干净治理快照，`evidence_revision` 才等于 REM `result_revision`。这样审计链快照与实际被测 subject revision 可分别复现。

整改不能把源审计 finding 改写为 resolved。只有新的 follow-up audit 可以独立确认修复结果。

## 当前索引

- [`REM-0006`](./records/REM-0006-20260714-codex-audit-aud-0009-wp-facts-exact-output.md)：`status=completed`；`verification=pending`；整改 `AUD-0009-F001` 的 `WP-Facts` exact-output allowlist gap，等待独立复审。
- [`REM-0005`](./records/REM-0005-20260714-codex-audit-active-audits.md)：`status=completed`；`verification=partial-by:AUD-0009`；整改 `AUD-0007-F001`、`AUD-0008-F001` 与 `AUD-0008-F002`。
- [`REM-0004`](./records/REM-0004-20260714-codex-audit-aud-0006-contract-clause-parsers.md)：`status=completed`；`verification=partial-by:AUD-0008`；五个精确复现已修正，但同 clause deferral mask、整行 rejection 豁免和未识别关系词仍可绕过 validator。
- [`REM-0003`](./records/REM-0003-20260714-codex-audit-aud-0005-contract-validators.md)：`status=completed`；`verification=partial-by:AUD-0006`；当前结构化 contract/DAG 正确，但两类 additive contradiction 仍可绕过 validator。
- [`REM-0002`](./records/REM-0002-20260714-codex-audit-aud-0004-rem-0001-follow-up.md)：`status=completed`；`verification=partial-by:AUD-0005`；当前计划文本已修正，但两项防语义漂移 validator 仍可被附加矛盾条款绕过。
- [`REM-0001`](./records/REM-0001-20260714-codex-audit-active-audits.md)：`status=completed`；`verification=partial-by:AUD-0004`；6 个 source findings 中 4 个已复审确认，2 个转入 `AUD-0004` 继续整改。

每份 REM 必须在创建时立即加入本索引，并且只能出现一次。索引格式必须记录 `status` 与 `verification`：

- `verification=not-ready`：仍在进行或 blocked，不能复审。
- `verification=pending`：整改已 completed/partial，是 follow-up audit 默认对象。
- `verification=verified-by:AUD-NNNN`：复审全部通过。
- `verification=partial-by:AUD-NNNN`：复审只确认部分修正。
- `verification=failed-by:AUD-NNNN`：复审未确认有效修正。

新记录从 [`templates/remediation-record.md`](./templates/remediation-record.md) 创建。整改使用 `/backend-fix-audit-findings` 或 `$backend-fix-audit-findings`，复审使用 `/backend-follow-up-audit` 或 `$backend-follow-up-audit`。
