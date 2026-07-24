---
id: GOAL-005-order-api-first-slice
doc: decision
status: done
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.1.0
---

# 决策记录 · GOAL-005

## D-001 · 采用父目标 D-018 作为首切片实施契约

**日期**：2026-07-25  
**状态**：`accepted`

**决定**：本目标直接采用 GOAL-002 D-018 固定的 `/v1/orders` 路由、Bearer/RBAC、分页与搜索、version CAS、状态机、batch-delete 事务边界、envelope 及错误码。

**为什么**：D-018 是实施前形成并由 GOAL-002 A-005 关闭 required finding 的权威契约；本目标不复制第二套协议语义。

**未选方案**：在补录目标内重新设计订单 API——会使已经完成的代码与历史审计失去稳定依据。

## D-002 · 首切片关门不代表订单域全量完成

**日期**：2026-07-25  
**状态**：`accepted`

**决定**：本目标成功边界不包含单项 DELETE 与 refund HTTP action；二者继续保留在 GOAL-002 总范围，并在后续独立工作包中补齐。

**为什么**：符合父目标 D-017 的渐进切片节奏，同时避免把部分交付错误表述为全量完成。

## D-003 · 以补录结构引用历史事实

**日期**：2026-07-25  
**状态**：`accepted`

**决定**：保留 GOAL-002 的决策、执行与审计历史作为原始记录；本目标建立独立边界并链接证据，不迁移或删除父目标内容。
