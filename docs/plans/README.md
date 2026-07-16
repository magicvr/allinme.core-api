# 实施计划管理

`plans/` 只存放实施计划及其配套 checklist。计划描述要交付什么、边界和决策；checklist 记录执行门禁与实际 Evidence。审计发现和审计结论必须写入 [`../audits/`](../audits/README.md)，不得再用计划文件替代审计记录。

## 标识与命名

- 计划 ID：`PLN-NNNN`，在当前目录和 `archived/` 中全局递增，永不重置或复用。
- 计划文件：`PLN-NNNN-<subject>.md`。
- 配套清单：`PLN-NNNN-<subject>-checklist.md`。
- `<subject>` 使用小写 ASCII kebab-case，表达稳定业务对象或交付范围，例如 `phase-05-attachment-lifecycle`；不写状态、负责人或日期。
- 创建日期、负责人和范围写入 frontmatter。文件名不带日期，且新计划归档后也保持原路径；稳定路径保证 closed AUD、REM 和 IMP 中的历史链接不会因归档失效。
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

`plan_id` 必须与文件名一致。`plans/` 根目录中的新计划允许 `status: active|archived`，`archived/` 只保存旧版目录迁移规则下的历史计划且必须为 `status: archived`。若计划被替代，不覆盖历史文件；把旧计划原地改为 archived，并在正文链接替代它的新 `PLN-NNNN`。

## 生命周期

1. 创建前扫描 `plans/` 与 `plans/archived/` 的最大编号并加一。
2. 从 [`templates/plan.md`](./templates/plan.md) 和 [`templates/checklist.md`](./templates/checklist.md) 创建同号文件，补齐范围、非目标、依赖、风险、回退和验收证据。
3. 执行期间只在 checklist 勾选已完成且有实际 Evidence 的条目；计划中的契约或边界变化先修改计划和事实源，再更新 checklist。
4. 完成后汇报已完成项、未执行项和剩余风险。只有用户明确确认后，才把同号 plan/checklist 的 `status` 一起改为 `archived`；文件保留在原路径，不再移动到 `archived/`。该目录仅兼容既有历史计划。
5. 归档后计划作为实施历史保留，不再改写为当前规范；后续纠错或新增工作创建新计划并显式关联。

## 当前活跃计划

- `PLN-0007`：[阶段五附件 MVP 实施计划](./PLN-0007-phase-05-attachment-mvp.md) / [checklist](./PLN-0007-phase-05-attachment-mvp-checklist.md)

## 稳定路径归档计划

- `PLN-0006`：[目标漂移纠偏与交付重心恢复计划](./PLN-0006-goal-drift-governance-realignment.md) / [checklist](./PLN-0006-goal-drift-governance-realignment-checklist.md)；治理工作已完成，产品关闭证据由 `PLN-0007` 承接。
- `PLN-0005`：[阶段五附件生命周期历史规格](./PLN-0005-phase-05-attachment-lifecycle.md) / [checklist](./PLN-0005-phase-05-attachment-lifecycle-checklist.md)；未实施，由 `PLN-0007` 替代。

旧版移动式归档记录见 [`archived/`](./archived/README.md)。

## 关联审计

- 新合同审计严格一计划一 AUD；多计划调用只负责集合级交叉检查和分派，每个受影响计划拥有独立 finding 与 remediation 状态。旧版多计划审计保持不可变。
- 审计发现需要大规模整改时，新建计划并在审计中引用；任何实际修复都必须通过独立 REM 或新的实施 IMP 完成，审计记录本身不得直接整改。
- 计划完成不自动关闭审计。审计者必须复核修复或明确记录接受风险后，才能关闭审计。

## 可选计划审计与验收

计划审计、整改复审和实施验收按变更风险选用，不是每个产品计划的默认前置。需要正式审计时仍遵守一计划一 AUD、独立基线、整改与复审分离及终态历史不可改写；工作流入口见 [`../audits/README.md`](../audits/README.md)。

产品计划进入实施的默认条件是：范围与非目标清晰、事实源一致、最小可证伪测试和回退边界已写入 plan/checklist。高风险或用户明确要求时，再使用 `$backend-plan-audit-until-ready` 或独立就绪验收；不得用治理闭环代替产品验证。
