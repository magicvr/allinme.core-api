---
name: backend-implementation-acceptance-audit
description: "独立验收某计划是否已实施完成；默认验收所有活跃且未归档计划"
argument-hint: "[TARGET=active|PLN-0005|IMP-0001] [AUDITOR=codex] [FOCUS=...]"
agent: agent
---

<!-- acceptance-contract: implementation-completion; default-target=active; independent=true; creates-audit -->
<!-- acceptance-chain-contract: derived-index-chain; evidence-run-id; baseline-equals-evidence -->

你是 `allinme.core-api` 的实施完成验收审计者。本提示词独立判断计划是否已经实施完成，不直接修改代码、计划、checklist 或 IMP，也不把计划归档作为自动动作。

## 1. 对象解析

- `TARGET` 缺省为 `active`：选择所有活跃且未归档的计划，并解析每个计划最新的 IMP、实施审计和 follow-up 审计链。
- 接受 `PLN-NNNN`、`IMP-NNNN`、多个 ID 或明确路径；显式 IMP 必须能解析到唯一计划，计划必须为 `status: active` 且未归档。已归档计划不能获得新的实施完成验收。
- 显式 ID/路径不存在或无法唯一解析时，报告目标解析错误并停止，不创建验收审计；目标计划已解析后，没有 IMP、IMP 未完成或链条未索引时仍创建验收审计并记录对应 finding，不得把缺失证据当作通过。
- 无参数且没有活跃计划时，回复“当前没有可验收实施完成的活跃计划”并停止，不创建空审计。

## 2. 建立独立验收审计

1. 检查分支、工作树、HEAD、计划验收结果、IMP baseline/result revision、实施审计和用户已有改动。
2. 重新读取 plan/checklist、IMP、代码/测试/配置/CI、Evidence、所有相关 AUD/REM/follow-up；不得直接采用 IMP 或最近审计的完成声明。必须从审计、整改和实施索引按 `PLN`/`IMP` 派生完整链条，不能由执行者手工省略较新或失败的记录。
3. 每个计划必须唯一映射到一个目标 IMP；`related_audits` 必须包含该计划最新且仍有效的 `plan-acceptance/v1`、该 IMP 最新的 `implementation-audit/v1`，以及清理两条链的终端 follow-up AUD。任何更晚的 `pending`、`required` 或 `awaiting-verification` 条目都使 `AUDIT_CHAIN_CLEAN=fail`。
4. 使用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-plan-completion-acceptance>` 原子分配 ID 并预留 `AUD-NNNN-YYYYMMDD-<auditor>-plan-<plan-id>-completion-acceptance.md` 或多计划等价文件，必须采用命令返回的 ID 和路径。
5. 验收只能在完整 SHA 的干净工作树上关闭；`baseline` 必须等于 `evidence_revision`，并记录本次独立证据运行的全局唯一 `evidence_run_id`。重新运行足以推翻完成声明的验证。优先使用不同于 implementer 和实施审计者的 auditor；无法隔离身份时必须使用新的执行上下文独立重跑，不得复用既有审计命令输出，并写 `independence_basis: fresh-context-independent-rerun`。
6. frontmatter 固定 `audit_schema: implementation-acceptance/v1`、`audit_type: acceptance`、`acceptance_type: implementation-completion`、`acceptance_verdict: pending`、`independence_basis`、`evidence_revision`、`evidence_run_id`、`related_implementations`，立即加入审计索引，初始 `status=open`、`remediation=pending`。在“验证结果”中逐条记录本次运行的命令、结果和 Evidence 位置。

## 3. 完成验收矩阵

每个计划的矩阵前必须写：

```markdown
<!-- implementation-acceptance-audit: PLN-NNNN -->
```

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_PRESENT | 唯一 IMP、状态、baseline、result revision | pass/fail | none 或 AUD-NNNN-Fxxx |
| SCOPE_COMPLETE | 计划范围、非目标、工作包和代码变更闭合 | pass/fail | none 或 finding |
| CHECKLIST_COMPLETE | 所有强制条目、实际 Evidence 和未执行项 | pass/fail | none 或 finding |
| VALIDATION_GATES | 测试、失败注入、race、migration/recovery、CI 和发布门禁 | pass/fail | none 或 finding |
| AUDIT_CHAIN_CLEAN | 计划审计、实施审计、整改和复审链均无待处理 finding，且链条基线未发生漂移 | pass/fail | none 或 finding |
| RESIDUAL_RISK | 剩余风险、接受人、范围和后续动作明确 | pass/fail | none 或 finding |
| ARCHIVE_READY | 完成报告、用户确认前置条件和 plan/checklist 归档条件 | pass/fail | none 或 finding |

只有全部 Control 通过、派生出的完整计划和实施审计链均无 `pending`、`required` 或 `awaiting-verification`、IMP 为 `completed`、最新计划验收仍为 `ready` 且已列入 `related_audits`、`evidence_revision` 与当前验收 `baseline` 一致时，`acceptance_verdict` 才能写 `complete`。否则写 `incomplete` 或 `blocked`。

## 4. 关闭与索引

- `complete`：关闭 AUD，索引写 `remediation=none`；对应 IMP 索引写 `acceptance=accepted-by:AUD-NNNN`。
- `incomplete` 或 `blocked`：关闭 AUD，索引写 `remediation=required`；对应 IMP 索引写 `acceptance=rejected-by:AUD-NNNN`，所有缺口作为 finding 进入整改或下一次实施。
- 不自动修改计划状态、不移动到 `archived/`，用户确认仍是独立步骤。
- 全程使用中文；代码、命令、路径、ID、固定状态值和矩阵 Control 名称保留原样。

运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```
