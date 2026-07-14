# 实施计划管理

`plans/` 只存放实施计划及其配套 checklist。计划描述要交付什么、边界和决策；checklist 记录执行门禁与实际 Evidence。审计发现和审计结论必须写入 [`../audits/`](../audits/README.md)，不得再用计划文件替代审计记录。

## 标识与命名

- 计划 ID：`PLN-NNNN`，在当前目录和 `archived/` 中全局递增，永不重置或复用。
- 计划文件：`PLN-NNNN-<subject>.md`。
- 配套清单：`PLN-NNNN-<subject>-checklist.md`。
- `<subject>` 使用小写 ASCII kebab-case，表达稳定业务对象或交付范围，例如 `phase-05-attachment-lifecycle`；不写状态、负责人或日期。
- 创建日期、负责人和范围写入 frontmatter。文件名不带日期，因为计划会持续修订，稳定路径比初始日期更适合作为长期引用。
- plan 与 checklist 必须同号、同主题并互相链接；一个计划默认只有一份 checklist。

## 必需元数据

计划与 checklist 至少包含：

```yaml
status: active
plan_id: PLN-0006
owner: 后端团队
created: 2026-07-14
last_updated: 2026-07-14
applies_to: implementation roadmap phase 6 pages
```

`plan_id` 必须与文件名一致。活跃区只允许 `status: active`，归档区只允许 `status: archived`。若计划被替代，不覆盖历史文件；归档旧计划，并在正文链接替代它的新 `PLN-NNNN`。

## 生命周期

1. 创建前扫描 `plans/` 与 `plans/archived/` 的最大编号并加一。
2. 从 [`templates/plan.md`](./templates/plan.md) 和 [`templates/checklist.md`](./templates/checklist.md) 创建同号文件，补齐范围、非目标、依赖、风险、回退和验收证据。
3. 执行期间只在 checklist 勾选已完成且有实际 Evidence 的条目；计划中的契约或边界变化先修改计划和事实源，再更新 checklist。
4. 完成后汇报已完成项、未执行项和剩余风险。只有用户明确确认后，才将同号 plan/checklist 一起移入 [`archived/`](./archived/README.md)，并把 `status` 改为 `archived`。
5. 归档后计划作为实施历史保留，不再改写为当前规范；后续纠错或新增工作创建新计划并显式关联。

## 当前活跃计划

- `PLN-0005`：[阶段五附件生命周期开发计划](./PLN-0005-phase-05-attachment-lifecycle.md) / [checklist](./PLN-0005-phase-05-attachment-lifecycle-checklist.md)

## 关联审计

- 审计可关联一个或多个计划，但审计 ID 与计划 ID 分别递增，不能共用编号。
- 审计发现需要大规模整改时，新建计划并在审计的 `related_plans` 中引用；小范围修复可直接记录在审计中。
- 计划完成不自动关闭审计。审计者必须复核修复或明确记录接受风险后，才能关闭审计。

## 计划审计闭环与验收

计划审计闭环使用 `$backend-plan-audit-until-ready`：先执行 `$backend-plan-audit`，再通过 `$backend-fix-audit-findings` 和 `$backend-follow-up-audit` 迭代清理 findings，最后运行独立的 `$backend-plan-acceptance-audit`。

计划验收审计不依赖闭环运行上下文，可单独执行，但必须独立重建并检查完整计划 AUD/REM/follow-up 链；无 `TARGET` 时选择所有活跃且未归档计划，并对每个计划给出 `ready`、`not-ready` 或 `blocked`。只有 `PLAN_AUDIT_CHAIN_CLEAN=pass` 且 verdict 为 `ready` 的计划才允许进入 `$backend-implement-plan`。
