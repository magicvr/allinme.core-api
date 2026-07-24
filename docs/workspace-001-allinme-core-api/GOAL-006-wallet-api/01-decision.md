---
id: GOAL-006-wallet-api
doc: decision
status: active
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.1.0
---

# 决策记录 · GOAL-006

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。

- **I-001**：required / open，阻断 W1～W3；实施前须固定钱包首切片契约。
- **I-002**：non-blocking / open，由 GOAL-002 M4 处理。

## D-001 · 继承父目标钱包范围与通用约束

**日期**：2026-07-25  
**状态**：`accepted`

**决定**：钱包业务范围继承 GOAL-002 D-008、D-009、D-011、D-015、D-017：active/frozen；freeze/unfreeze/batch-freeze；viewer 只读；SQLite 默认且 service 依赖 port；列表使用 `data.list`/`data.total` envelope；创建可设初始余额，更新不得修改余额；不提供调账或支付网关。

**为什么**：这些范围已由父目标规划与审计闭环，不应在子目标静默扩张。

**未选方案**：增加充值/提现/任意余额调整——超出 MVP，且会引入账务一致性与审计复杂度。

## D-002 · 先关闭 I-001 再实施

**日期**：2026-07-25  
**状态**：`accepted`

**决定**：在开始钱包领域、repository 或 handler 实施前，先记录可直接测试的跨层契约，并追加 design-plan 自审。契约至少覆盖路由、请求字段、列表参数、version CAS、freeze/unfreeze 状态转换、batch-freeze 原子性、RBAC 与错误语义。

**为什么**：沿用订单首切片 A-004/A-005 的有效做法，防止各层自行猜测导致不一致。

**后续**：用 `/govern` 推进 GOAL-006 I-001 契约冻结。
