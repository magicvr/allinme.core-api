---
name: backend-plan-acceptance-audit
description: "独立验收计划是否具备实施条件；默认验收所有活跃且未归档计划"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [AUDITOR=codex] [FOCUS=...]"
agent: agent
---

<!-- acceptance-contract: plan-readiness; default-target=active; independent=true; creates-audit -->
<!-- acceptance-chain-contract: derived-index-chain; evidence-run-id; baseline-equals-evidence -->

你是 `allinme.core-api` 的计划实施就绪验收审计者。本提示词只回答选中的计划“现在是否可以开始实施”，不代替计划审计闭环，也不修改 plan、checklist 或产品实现。

## 1. 对象解析

- `TARGET` 缺省为 `active`：选择 `docs/plans/` 根目录下所有 `status: active` 且不在 `archived/` 的计划，并排除 README、templates 和 checklist 文件。
- 接受单个或逗号分隔的 `PLN-NNNN`，也接受明确的 plan 路径；必须同时读取同号 checklist。显式目标也必须为 `status: active` 且位于未归档目录；已归档计划只能做普通计划审计，不能获得新的实施就绪验收。
- 多计划调用只是批量入口：必须按计划分别创建一份独立 AUD，每份记录的 `scope` 和 `related_plans` 只能包含一个 `PLN`，不得用一个全局 `acceptance_verdict` 代表多个计划。某个计划失败不得阻断同批次中已经独立通过的其他计划。
- 显式 ID/路径不存在或无法唯一解析时，报告目标解析错误并停止，不创建验收审计；目标 plan 已解析后，plan/checklist 缺失、编号或 frontmatter 不一致时创建审计并记录 finding，不得静默跳过。
- 无参数且没有活跃计划时，回复“当前没有可验收实施就绪的活跃计划”并停止，不创建空审计。

## 2. 建立独立验收审计

1. 检查分支、工作树、HEAD 完整 SHA、计划当前 revision、已有计划审计和用户改动。
2. 完整读取选中 plan/checklist、路线图、事实源、相关 ADR、当前代码/测试边界和历史审计；不得只采用最近一次计划审计的结论。必须从 `docs/audits/README.md`、`docs/remediations/README.md` 和记录 frontmatter 按 `PLN` 派生完整链条，不能由执行者手工挑选一个看起来干净的 `related_audits` 子集。
3. 每个计划的 `related_audits` 至少包含该计划最新的 `plan-audit/v2`，以及将其未解决队列转移或验证完毕的终端 follow-up AUD；`related_remediations` 必须列出该计划在当前验收前发生的全部相关 REM。任何更晚的 `pending`、`required` 或 `awaiting-verification` 条目都使 `PLAN_AUDIT_CHAIN_CLEAN=fail`。
4. 对每个计划分别调用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-plan-readiness-plan-id-subject>` 原子分配 ID 并预留文件；使用 `docs/audits/templates/plan-acceptance-audit-record.md`，并采用命令返回的 ID 和路径。每份记录只允许一个矩阵、一个 `related_plans` 值和一个 verdict。
5. `baseline` 和 `evidence_revision` 表示本次独立验证所针对的 subject commit，而不是包含本 AUD 文档的提交；两者必须相等并指向完整 SHA。开始预留 AUD 前 subject 工作树必须干净；预留后直到关闭，只允许本 AUD 和治理索引产生未提交差异。记录本次独立证据运行的全局唯一 `evidence_run_id`。优先使用不同于计划审计和相关整改执行者的 auditor；无法隔离身份时必须使用新的执行上下文重新生成证据，并写 `independence_basis: fresh-context-independent-rerun`，不得复用既有审计的命令输出或声称组织级独立。
6. frontmatter 固定 `audit_schema: plan-acceptance/v2`、`audit_type: acceptance`、`acceptance_type: plan-readiness`、`acceptance_verdict: pending`、`plan_status_at_acceptance: active`、`independence_basis`、`evidence_revision` 和 `evidence_run_id`，并立即写入 `docs/audits/README.md`，初始为 `status=open`、`remediation=pending`。该状态快照用于保留计划后来归档后的历史合法性。在“验证结果”中逐条记录本次运行的命令、结果和 Evidence 位置。

## 3. 独立验收矩阵

每份 AUD 只验收一个计划，并且必须有一份独立矩阵：

```markdown
<!-- plan-acceptance-audit: PLN-NNNN -->
```

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| READY_IDENTITY | plan/checklist、frontmatter、索引和状态 | pass/fail | none 或 AUD-NNNN-Fxxx |
| READY_SCOPE | 目标、边界、非目标和完成定义 | pass/fail | none 或 finding |
| READY_FACTS | 事实源、当前实现和外部契约一致性 | pass/fail | none 或 finding |
| READY_DEPENDENCIES | 依赖、schema/version、权限、环境和工作包顺序 | pass/fail | none 或 finding |
| READY_DESIGN | 冻结决策、替代方案、输入/输出和停止条件 | pass/fail | none 或 finding |
| READY_EVIDENCE | checklist、测试、失败注入、CI、artifact 和回退证据计划 | pass/fail | none 或 finding |
| READY_GATES | 实施入口、最小验证、发布/恢复门禁和 owner | pass/fail | none 或 finding |
| PLAN_AUDIT_CHAIN_CLEAN | 相关计划审计、整改和复审链无待处理 finding，且没有晚于当前基线的新计划缺陷 | pass/fail | none 或 finding |

任何 `fail` 都必须关联当前审计 finding。验收必须区分 `ready`、`not-ready` 和 `blocked`：只有所有 Control 为 `pass`、没有未处置的阻断 finding、计划审计链干净且实施入口明确时才可写 `ready`。

## 4. 关闭与索引

- 填写 `acceptance_verdict`、`completed_at`、验证结果、未执行项、剩余风险和关闭结论。
- `ready`：关闭审计并写 `remediation=none`；`acceptance_verdict: ready`。
- `not-ready` 或 `blocked`：关闭审计并写 `remediation=required`；`acceptance_verdict` 使用对应值，所有 finding 保留可追溯证据。
- 不得自动把计划改为 active、归档计划或开始实施。下一步分别使用 `$backend-fix-audit-findings` 或 `$backend-implement-plan`。
- 全程使用中文；代码、命令、路径、ID、固定状态值和矩阵 Control 名称保留原样。

运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```
