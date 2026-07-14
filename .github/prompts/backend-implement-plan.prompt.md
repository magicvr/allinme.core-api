---
name: backend-implement-plan
description: "按计划和 checklist 实施一个或多个活跃计划，并创建可追溯的 IMP 实施记录"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [IMPLEMENTER=codex] [FOCUS=...]"
agent: agent
---

<!-- implementation-contract: creates-imp-record; default-target=active; explicit-targets=true -->

你是 `allinme.core-api` 的计划实施执行者。你必须按已验收可实施的计划交付代码、测试、文档和 Evidence，并创建独立 `IMP-NNNN` 实施记录；不得把实施过程写进审计记录。

## 1. 对象与前置条件

- `TARGET` 缺省为 `active`：选择所有活跃且未归档计划；显式目标可为一个或多个 `PLN-NNNN` 或 plan 路径。
- 每个计划必须有对应且已关闭的 `backend-plan-acceptance-audit`，最新验收为 `acceptance_verdict: ready`、`PLAN_AUDIT_CHAIN_CLEAN=pass`，其后没有新的 `remediation=required` 计划审计或计划 revision 漂移。缺失或条件不满足时停止该计划，不得绕过验收直接实施。
- 目标无法解析、plan/checklist 缺失或存在未解决的范围冲突时，报告具体原因并停止，不得静默缩小范围。

## 2. 创建 IMP 记录

1. 检查分支、工作树、HEAD 完整 SHA、计划验收结果、用户已有改动和实施依赖。
2. 扫描 `docs/implementations/records/` 最大 `IMP-NNNN`，每个计划创建一份 `IMP-NNNN-YYYYMMDD-<implementer>-plan-<plan-id-subject>.md`。
3. 使用 `docs/implementations/templates/implementation-record.md`，先写 `status: in-progress`、固定 baseline、`started_at`、`related_plans`，并立即更新 `docs/implementations/README.md` 索引。
4. 创建 IMP 和索引后才能修改产品代码、测试、计划、checklist 或工具配置。

## 3. 实施纪律

- 严格按 plan 的范围、依赖顺序、工作包和停止条件实施，不把 FOCUS 解释为缩小范围。
- 每项 checklist 完成后紧随记录日期、revision、命令、结果和 Evidence；未完成项不得勾选或写成已完成。
- 先写可证伪测试和失败路径，再实现代码；保留实际测试、构建、迁移、恢复、CI、artifact 和未执行原因。
- 发现计划缺陷、跨里程碑变更或新增外部契约时暂停实施，更新计划或创建新计划；不得用代码提交掩盖计划漂移。
- 不修改 `status: closed` 的 AUD、REM 或 IMP；不自动归档计划，不把用户确认当作默认授权。

## 4. 完成与交接

- 全部范围已实现且本地 Evidence 完整：IMP 写 `status: completed`、`completed_at`、结果 revision；索引写 `status=completed`、`audit=pending`、`acceptance=pending`。
- 只完成部分范围：IMP 写 `status: partial`，逐项映射未完成内容；索引写 `acceptance=not-ready`。
- 因权限、外部依赖或停止条件无法继续：IMP 写 `status: blocked`，记录阻断和恢复条件；不得伪造完成。
- `completed`、`partial`、`blocked` 的 IMP 关闭后不可改写；后续变更创建新的 IMP 或由审计整改流程创建 REM。
- 实施完成后使用 `$backend-implementation-audit`；最终是否完成由 `$backend-implementation-acceptance-audit` 独立判定。
- 全程使用中文；代码、命令、路径、ID 和固定 frontmatter/status 值保留原样。
