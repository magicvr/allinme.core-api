---
name: backend-plan-acceptance-audit
description: "独立验收计划是否具备实施条件；默认验收所有活跃且未归档计划"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [AUDITOR=codex] [CONTEXT_ID=<uuid>] [FOCUS=...]"
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
- `FOCUS` 只能增加深度。`CONTEXT_ID` 必须来自不同于计划审计、整改和 follow-up 的新执行上下文。

## 2. 建立独立验收审计

1. 检查分支、工作树、HEAD 完整 SHA、计划当前 revision、已有计划审计和用户改动。
2. 完整读取计划、事实源和历史审计；从索引递归派生以 `plan-audit/v2` 或既有 plan-readiness 验收为根的计划就绪链，不得手选子集。实施审计/完成验收及仅由它们派生的 REM/follow-up 不属于计划就绪链，不得污染 readiness verdict。
3. `related_audits` 至少包含最新计划审计及清理该就绪链的终端 follow-up；`related_remediations` 列出该就绪链在验收前发生的全部 REM。链内更晚的待处理状态使 Control 失败。
4. 对每个计划先恢复相同计划/baseline 的唯一 open 验收；不存在时才调用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-plan-readiness-plan-id-subject>` 分配记录。
5. `started_at` 固定链条快照；链条在证据运行期间变化时重启。`baseline` 与 `evidence_revision` 必须相等。记录新的 `execution_context_id`、完整 `source_context_ids` 和唯一 `evidence_run_id`。
6. frontmatter 固定 `governance_contract: audit-loop/v3`、`audit_schema: plan-acceptance/v2`、`independence_basis: separate-context`、上下文字段及现有验收字段，并立即索引。

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
- `not-ready`：关闭审计并写 `remediation=required`。
- `blocked`：关闭审计并写 `remediation=decision-required`；记录责任人和恢复条件，不进入自动整改队列。
- 不得自动把计划改为 active、归档计划或开始实施。下一步分别使用 `$backend-fix-audit-findings` 或 `$backend-implement-plan`。
- 全程使用中文；代码、命令、路径、ID、固定状态值和矩阵 Control 名称保留原样。

运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```
