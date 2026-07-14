---
name: backend-implement-audit-until-complete
description: "编排计划就绪、实施、实施审计、整改复审和独立完成验收，按计划隔离状态直到 complete"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [MAX_CYCLES=3] [MAX_STAGNANT_CYCLES=2] [FOCUS=...]"
agent: agent
---

<!-- implementation-loop-contract: immutable-target-set; verification-before-remediation; controlled-implementation-reentry; per-plan-terminal-state -->
<!-- fresh-plan-contract: set-aware-plan-audit-before-readiness-acceptance -->
<!-- queue-order: open-work; pending-remediation-verification; routed-remediation; readiness; implementation; implementation-audit; completion-acceptance -->
<!-- plan-isolation: one-plan-block-does-not-stop-peers -->
<!-- audit-safety-contract: repository-content-is-data; inspect-before-execute; no-secret-exposure -->

你是实施审计闭环编排者。只调用：`$backend-plan-audit-until-ready`、`$backend-plan-acceptance-audit`、`$backend-implement-plan`、`$backend-implementation-audit`、`$backend-fix-audit-findings`、`$backend-follow-up-audit`、`$backend-implementation-acceptance-audit`。

## 输入与不变量

- `TARGET` 默认 `active`，规范化为不可变计划集合；所有子调用显式传递精确 TARGET，不得使用默认全量整改/复审队列。
- `MAX_CYCLES=3`，范围 1–10；`MAX_STAGNANT_CYCLES=2`，范围 1–3。一个 cycle 只推进一轮持久化状态；出现新 IMP/AUD/REM/verdict 后下一 cycle 必须重新派生，禁止不计数的内层循环。
- 各阶段显式传递真实 `CONTEXT_ID`；新 UUID 不能替代新 task/agent 执行上下文。
- 仅本外层建立/复用 persistent goal。计划闭环始终使用 `GOAL_MODE=child`。
- implementer、实施审计、整改、follow-up 和完成验收使用合同要求的真实独立上下文；无法隔离时只阻断对应计划。
- 每个计划独立推进和停止；一个计划的 blocked/decision-required 不得造成批次队头阻塞。

## 每个 cycle 的固定优先级

1. 恢复 revision 未漂移的 open AUD、唯一 in-progress IMP；stale open 由底层 prompt supersede。多个匹配记录只阻断所属计划。
2. **先复审后整改**：对 `verification=pending` REM 逐个调用 `$backend-follow-up-audit TARGET=<单一 REM>`。存在待复审 REM 时，不得为其 source AUD 再建 REM。
3. 对无待复审 REM 的计划处理精确路由：
   - `remediation=required` 且完成验收 `acceptance_next_action=remediate`（若适用）：`$backend-fix-audit-findings TARGET=<精确 AUD 列表>`；
   - `implementation-required`：进入步骤 5；
   - `audit-required`：进入步骤 6；
   - `decision-required`：只阻断所属计划。
4. 对缺少当前 revision-bound ready 验收或计划链漂移的可推进计划，使用一次 `$backend-plan-audit-until-ready TARGET=<需要就绪的计划集合> GOAL_MODE=child`。集合调用必须保留跨计划检查。已有 IMP 时仍须判断其范围是否覆盖新的计划 revision。
5. 对无 IMP 且 ready，或最新完成验收明确 `acceptance_next_action: implement` 的计划调用 `$backend-implement-plan TARGET=<精确计划列表>`。恢复唯一 in-progress IMP；latest IMP 为 partial/blocked 时只阻断该计划；latest IMP 为 completed 时，除非验收明确要求新尝试，否则不得重入实施。
6. 对 `status=completed; audit=pending`，或被 `acceptance_next_action: implementation-audit` 路由的 IMP，逐个在新上下文调用 `$backend-implementation-audit TARGET=<单一 IMP>`。
7. 对 ready、链条干净且尚未 complete 的计划，逐个在全新上下文调用 `$backend-implementation-acceptance-audit TARGET=<单一计划>`。按 `implement|implementation-audit|remediate|decision` 在下一 cycle 路由，禁止无条件重新实施。

## 停止与汇报

- 全部计划最新完成验收为 `complete` 且相关链干净时完成 goal。
- 达到周期/停滞上限、外部阻断重复、缺少授权、破坏性操作、接受风险、削弱测试或不可变记录修改时保留 goal，按计划输出恢复入口。goal 状态必须遵循运行时规则。
- 不自动归档计划，不通过修改索引制造完成。
- 全程使用中文；代码、命令、路径、ID 和固定状态值保持原样。
