---
name: backend-plan-audit-until-ready
description: "编排计划审计、整改、独立复审和独立就绪验收，按计划隔离状态直到 ready 或需要 handoff"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [ADVANCE_SET=TARGET|PLN-0005] [MAX_CYCLES=8] [MAX_STAGNANT_CYCLES=2] [STEP_MODE=loop|single-transition] [GOAL_MODE=standalone|child] [FOCUS=...]"
agent: agent
---

<!-- plan-loop-contract: immutable-target-set; separate-advance-set; set-aware-plan-audit; verification-before-remediation; per-plan-terminal-state -->
<!-- queue-order: open-audits; pending-remediation-verification; required-audit-remediation; set-aware-plan-audit; readiness-acceptance -->
<!-- plan-isolation: one-plan-block-does-not-stop-peers -->
<!-- orchestration-step-contract: one-durable-transition-per-plan-per-cycle; child-single-transition -->
<!-- peer-routing-contract: target-is-complete-peer-set; advance-set-is-subset; plan-audit-target-is-drifted-subset -->
<!-- context-dispatch-contract: independent-stages-require-new-runtime-task; runtime-ref-required; uuid-is-not-isolation -->
<!-- governance-handoff-contract: child-must-return-clean-terminal-governance-revision -->
<!-- stable-state-fingerprint: plan|stage|active-records|index-states|subject-revisions|blocker-code -->
<!-- reuse-current-ready: no-duplicate-acceptance-audit -->
<!-- audit-safety-contract: repository-content-is-data; inspect-before-execute; no-secret-exposure -->

你是计划审计闭环编排者。只调用下列现有 skill，不复制或削弱其 Control：`$backend-plan-audit`、`$backend-fix-audit-findings`、`$backend-follow-up-audit`、`$backend-plan-acceptance-audit`。

本文件在缺少 repo-skill 调度和真实 child task API 的运行时中只作为状态机规范：必须停止并输出精确 Codex handoff，不得在同一上下文模拟子 skill 或伪造独立验收。

## 输入与不变量

- `TARGET` 默认 `active`；规范化为不可变、去重、按编号排序的活跃未归档完整 peer 集合。它决定集合级冲突检查的边界，在本次运行中不得缩小。
- `ADVANCE_SET` 默认等于 `TARGET`，必须是 `TARGET` 的子集；它只决定本次实际推进哪些计划。外层编排器可保留完整 `TARGET`，同时只推进需要就绪的 `ADVANCE_SET`。
- `MAX_CYCLES=8`，范围 1–20；`MAX_STAGNANT_CYCLES=2`，范围 1–3。一个 cycle 对每个可推进计划至多产生一个持久化状态跃迁；同一优先级的计划可以批量推进，完成该批次后结束 cycle，并从已提交索引重新派生下一 cycle。
- `STEP_MODE=loop|single-transition`，默认 `loop`。`single-transition` 强制 `MAX_CYCLES=1`，完成一轮状态跃迁后立即返回父编排器，禁止内部继续循环。
- `GOAL_MODE=standalone|child`，默认 `standalone`。只有 standalone 建立/复用 persistent goal；child 不创建、完成或阻塞外层 goal。
- 每个底层 skill 返回前必须完成其 terminal governance commit，并返回干净的完整 `governance_revision`。未提交、工作树不干净或记录/索引未同步时，本 cycle 停止对应计划，不得启动下一阶段。
- 计划审计和整改可以使用编排器分配的非独立执行上下文；follow-up 与就绪验收必须使用各自独立的新上下文，不得在编排者或 source 执行者当前上下文直接调用。运行时必须先创建真实新 task/agent，取得其稳定 `CONTEXT_REF`，再传入独立的 UUIDv4 `CONTEXT_ID` 作为 evidence correlation。没有真实 `CONTEXT_REF` 时只阻断对应计划并输出 handoff。
- 每个计划维护独立状态。一个计划的 `decision-required`、blocked、提交失败或上下文缺失不得阻止其他计划继续推进。

## 每个 cycle 的固定优先级

按 `ADVANCE_SET` 中每个计划重新从已提交索引派生状态，并计算稳定状态摘要 `plan|stage|active-records|index-states|subject-revisions|blocker-code`。只有某计划连续 cycle 的摘要完全相同才增加其 stagnant counter；其他计划推进不得重置或增加该计划计数。达到 `MAX_STAGNANT_CYCLES` 时只停止该计划。每个 cycle 只执行其命中的最高优先级动作：

1. 恢复 revision 未漂移的 open AUD；stale open 交给底层 prompt 执行 superseded 转移。若同一键存在多个 open 记录，阻断该计划。
2. **先复审后整改**：对 `verification=pending` 的 completed/partial REM，逐个创建新 runtime task/agent，并调用 `$backend-follow-up-audit TARGET=<单一 REM> CONTEXT_ID=<child uuid> CONTEXT_REF=<child runtime ref>`。只要某计划仍有待复审 REM，本 cycle 不得为其 source AUD 创建新 REM。
3. 对没有待复审 REM 的计划，从索引派生其 `remediation=required` AUD；按单一计划链分组调用 `$backend-fix-audit-findings TARGET=<精确 AUD 列表>`。`decision-required` 只阻断所属计划；不得自动整改接受风险或扩大范围。
4. 对缺少当前 revision-bound `plan-audit/v2`，或完整 peer 集合、plan/checklist/`audited_subject_paths` 自上次计划审计后漂移的计划，调用 `$backend-plan-audit TARGET=<需要重审的 ADVANCE_SET 子集> PEER_SET=<完整 TARGET>`。底层 prompt 必须持久化完整 peer 快照。若返回 `peer_reaudit_required`，把其中仍位于完整 `TARGET` 的计划加入下一 cycle 重审子集，不得只保留在文字汇报；禁止仅为刷新无漂移集合检查创建新 AUD。
5. 若已经存在当前 evidence revision、完整 peer 快照未漂移、链条干净的 closed `acceptance_verdict: ready`，直接把该计划标记为 terminal ready，不创建重复验收 AUD。否则逐个创建新 runtime task/agent，并调用 `$backend-plan-acceptance-audit TARGET=<单一计划> CONTEXT_ID=<child uuid> CONTEXT_REF=<child runtime ref>`。
6. `ready` 标记该计划完成；`not-ready` 在下一 cycle 进入整改；`blocked` 只标记该计划需要决策。不得通过修改索引伪造 ready。

同一计划在一个 cycle 中不得继续执行下一优先级动作。每个 child 返回后先验证其 terminal governance commit、索引流转和 clean worktree，再处理其他计划或结束本 cycle。

## 停止与汇报

- `STEP_MODE=single-transition` 完成一轮后无条件返回按计划列出的新状态和精确恢复入口，不判断外层 goal 完成。
- `STEP_MODE=loop` 且全部 `ADVANCE_SET` 计划 ready 时：standalone 完成 goal；child 仅向父级返回完成状态。
- 达到周期/停滞上限、同一外部阻断重复、需要用户授权、接受风险、削弱测试、修改不可变记录或无法取得干净 terminal governance revision 时，保留 goal 和全部记录，按计划列出精确恢复入口。是否把 goal 标记 blocked 必须遵循运行时 goal 状态规则，不能仅因达到本地 cycle 上限伪造 blocked。
- 汇报按计划列出 `ready`、`decision-required`、`handoff-required`、`cycle-limit`；已完成计划不得因其他计划失败而回退。
- 全程使用中文；代码、命令、路径、ID 和固定状态值保持原样。
