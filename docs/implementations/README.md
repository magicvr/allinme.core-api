# 实施记录管理

`implementations/records/` 保存按 `PLN-NNNN` 执行实际交付的 `IMP-NNNN` 记录。它连接计划、checklist、代码/测试变更和实施审计：IMP 说明实施做了什么以及证据在哪里，AUD 说明是否符合计划，独立验收 AUD 才能确认是否完成。

## 标识与命名

- 实施 ID：`IMP-NNNN`，扫描全部 `records/` 后全局递增，永不复用。
- 文件名：`IMP-NNNN-YYYYMMDD-<implementer>-plan-<plan-id-subject>.md`。
- 一个计划的一次实施尝试对应一份 IMP；后续继续实施创建新的 IMP，不改写已关闭记录。
- `started_at`、`completed_at` 和 `last_updated` 记录跨日实施；文件名日期使用开始日期。

## 生命周期与索引

1. 实施前必须存在已关闭的独立计划验收 AUD，且最新 `acceptance_verdict: ready`、`PLAN_AUDIT_CHAIN_CLEAN=pass`，验收后没有计划 revision 或审计链漂移。
2. 修改代码、测试、文档或 checklist 前先创建 IMP 并加入本索引。
3. `status=in-progress`：正在实施；`audit=not-ready`；`acceptance=not-ready`。
4. `status=completed`：计划范围已实施且本地 Evidence 齐全；`audit=pending`；`acceptance=pending`。
5. `status=partial` 或 `status=blocked`：必须逐项记录未完成内容、阻断原因和恢复条件；不得进入完成验收。
6. 实施审计完成后，索引写 `audit=audited-by:AUD-NNNN`；整改和复审仍以 `docs/audits/` 与 `docs/remediations/` 为准。
7. 完成验收必须为每个计划分别创建 AUD，从索引派生完整计划与实施审计链，且只能验收该计划最新 IMP。没有影响实施结果的 REM 时，`effective_result_revision` 等于 IMP `result_revision`；存在已验证的 `affects_implementation: true` REM 时，等于最新相关 REM 的 `result_revision`。验收的 baseline/evidence revision 必须等于该 effective revision；通过后索引写 `acceptance=accepted-by:AUD-NNNN`，失败或阻断写 `acceptance=rejected-by:AUD-NNNN`。计划不会自动归档。

`completed`、`partial`、`blocked` 的 IMP 记录不可改写。针对已完成 IMP 的窄范围整改由 REM 记录新的 `result_revision` 并进入 effective revision 链；需要重新执行计划工作包、改变计划范围或无法由原 finding 限定的工作必须创建新的 IMP。不得通过改写历史 IMP 或遗漏 REM 伪造闭环完成。

新实施记录固定使用 `implementation_schema: implementation/v2`，并记录 `plan_evidence_revision`，其值必须等于实施开始时引用的最新 ready `plan-acceptance/v2` 的 `evidence_revision`。这用于区分“计划验收后的治理记录提交”与真正的 plan/checklist 内容漂移。

## 必需内容

每份 IMP 至少记录：

1. 计划和 checklist 的精确链接、计划验收 AUD 与 baseline；
2. 工作包/条目到实际代码、测试、文档和 Evidence 的映射；
3. 实际 revision、命令、结果、未执行项和剩余风险；
4. `status`、实施审计状态、完成验收状态和下一步交接。

模板：[`templates/implementation-record.md`](./templates/implementation-record.md)。

## 当前索引

暂无实施记录。

实施入口：`$backend-implement-plan`；实施审计：`$backend-implementation-audit`；完成验收：`$backend-implementation-acceptance-audit`。
