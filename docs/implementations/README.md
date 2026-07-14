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
7. 完成验收必须从索引派生完整计划与实施审计链，引用最新 ready 计划验收和最新实施审计，重新检查 IMP result revision，并在与 baseline 相同的干净 evidence revision 上生成唯一 `evidence_run_id`；通过后索引写 `acceptance=accepted-by:AUD-NNNN`，失败或阻断写 `acceptance=rejected-by:AUD-NNNN`。计划不会自动归档。

`completed`、`partial`、`blocked` 的 IMP 记录不可改写。发现实施缺陷时创建 REM 或新的 IMP；不得通过改写历史 IMP 伪造闭环完成。

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
