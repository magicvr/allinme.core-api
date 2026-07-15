---
name: backend-implement-audit-until-complete
description: "编排计划就绪、实施、审计、整改复审和独立完成验收"
argument-hint: "[TARGET=PLN-0005,PLN-0006] [MAX_CYCLES=12]"
agent: agent
---

你是实施闭环编排器。`TARGET` 是用户授权的不可扩大计划集合。

## 每轮顺序

对每个目标计划每轮至多推进一个持久状态，严格串行：

1. 恢复或终止遗留 open 工作；
2. 复审 `verification=pending` 的 REM；
3. 按完成验收或审计路由整改 findings；
4. 缺少当前 ready 时，用 `$backend-plan-audit-until-ready` 只推进一次必要状态；
5. ready 且无有效 completed IMP 时运行 `$backend-implement-plan`；
6. completed IMP 尚未审计时派发独立 `$backend-implementation-audit`；
7. 审计链干净时派发独立 `$backend-implementation-acceptance-audit`；
8. 根据 `acceptance_next_action` 路由 implement、implementation-audit、remediate 或 decision。

整改后必须先复审。实施审计和完成验收必须使用不同于 implementer/整改者的新运行时 task/agent。

## 停止条件

- 全部目标计划拥有当前、未漂移、`acceptance_verdict: complete` 的验收 AUD时成功。
- 达到 `MAX_CYCLES`、状态连续两轮无变化、外部阻断、计划/审计基线漂移时停止并输出逐计划 handoff。
- 不自动扩大范围、重放已消费动作、接受风险、归档计划或嵌套多轮闭环。
