# 实施记录管理

`implementations/records/` 保存按 `PLN-NNNN` 执行实际交付的 `IMP-NNNN` 记录。它连接计划、checklist、代码/测试变更和实施审计：IMP 说明实施做了什么以及证据在哪里，AUD 说明是否符合计划，独立验收 AUD 才能确认是否完成。

## 标识与命名

- 实施 ID：`IMP-NNNN`，扫描全部 `records/` 后全局递增，永不复用。
- 文件名：`IMP-NNNN-YYYYMMDD-<implementer>-plan-<plan-id-subject>.md`。
- 一个计划的一次实施尝试对应一份 IMP；后续继续实施创建新的 IMP，不改写已关闭记录。
- `started_at`、`completed_at` 和 `last_updated` 记录跨日实施；文件名日期使用开始日期。

## 生命周期与索引

1. 实施前必须存在已关闭的独立计划验收 AUD，且最新 `acceptance_verdict: ready`、`PLAN_AUDIT_CHAIN_CLEAN=pass`，验收后没有计划 revision 或审计链漂移。
2. 修改代码、测试、文档或 checklist 前先创建 IMP、加入本索引并提交独立 open checkpoint governance commit；未取得干净 checkpoint 不得开始实施。
3. `status=in-progress`：正在实施；`audit=not-ready`；`acceptance=not-ready`。只允许原 `runtime_context_ref` 的可恢复 task 续跑；context 丢失时必须由全局唯一新 context 的新 IMP 原子替代，禁止接管。
4. `status=completed`：计划范围已实施且本地 Evidence 齐全；`audit=pending`；`acceptance=pending`。
5. `status=partial` 或 `status=blocked`：必须逐项记录未完成内容、阻断原因和恢复条件；不得被当作完成，但可以进入负向完成验收，由验收在恢复条件满足时路由新的 IMP，或在仍需授权/决策时路由 `decision`。
6. `status=superseded`：原 in-progress runtime task/ref 已丢失；旧 IMP 与新 in-progress IMP、索引和新 attestation 必须在同一精确事务中双向记录 `superseded_by`/`supersedes` 与 `supersession_reason: context-loss`。索引固定 `audit=not-ready; acceptance=not-ready`。
7. 实施审计使用 `implementation-audit/v2`，必须在不同于 implementer 的执行上下文中运行，且 `evidence_revision` 精确等于 IMP `result_revision`；完成后索引写 `audit=audited-by:AUD-NNNN`。
8. 完成验收必须为每个计划分别创建 AUD，从索引派生完整计划与实施审计链，且只能验收该计划最新 IMP。实施 REM 通过 `parent_result_revision -> result_revision` 形成线性 Git 祖先链；`effective_result_revision` 是唯一链尾并必须等于 `evidence_revision`。`baseline` 则是包含 IMP/AUD/REM source records 的后继治理快照。

`completed`、`partial`、`blocked`、`superseded` 的 IMP 记录不可改写。针对已完成 IMP 的窄范围整改由 REM 记录新的 `result_revision` 并进入 effective revision 链；需要重新执行计划工作包、恢复 partial/blocked 实施、改变计划范围或无法由原 finding 限定的工作必须创建新的 IMP。当前 ready 未漂移且恢复证据已变化时可直接新建 IMP；只有计划/peer/subject 漂移才重新走计划审计/验收。不得通过改写历史 IMP、重放已消费的 `implement` 路由或遗漏 REM 伪造闭环完成。

新实施记录固定使用 `governance_contract: audit-loop/v3`、`implementation_schema: implementation/v2`、全局唯一 `execution_context_id`、逐记录 `runtime_context_ref` 和 `plan_evidence_revision`。每份新 IMP 必须由真实、独立的 runtime task/context 创建；批量入口不得复用一个 ID/ref。ready 前置必须同时保持完整 peer 快照未漂移。实施期间只允许更新 checklist 的实际 Evidence；计划契约或范围变化必须关闭当前 IMP 并重新走计划审计/验收，不能在同一 IMP 中事后改变验收标准。

若实施由失败完成验收的 `acceptance_next_action: implement` 触发，IMP 必须在 `trigger_audits` 记录该 AUD，并把源 AUD 索引流转为 `remediation=implemented-by:IMP-NNNN`。这表示路由动作已被新的实施尝试消费，不等于提前宣告新 IMP 完成。

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
