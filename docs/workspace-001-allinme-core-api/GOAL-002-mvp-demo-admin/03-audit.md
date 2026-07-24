---
id: GOAL-002-mvp-demo-admin
doc: audit
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-25
version: 0.12.0
---

# 审计 · GOAL-002

## 信息就绪核对（按 scope）

| 焦点 | 状态 | 说明 |
|------|------|------|
| I-001 / I-008 | 已关 | 策略 A；2.4.1 |
| I-002～I-007 | decided | 方案冻结 |
| I-009 骨架门禁 | **verified** | GOAL-003 done；M2 已实施 |
| I-010 制品校验路径 | **open** | D-016；阻断 M4 校验宣称 |

## 审计意见台账

## A-001 · 规划合理性交叉审计（vs GOAL-003 串行）（2026-07-24）

- **source**：independent
- **auditor**：GitHub Copilot · Grok 4.5
- **类型**：design-plan
- **scope**：GOAL-002 当前规划（M0～M5、I-00N、与 GOAL-003 / Root R0.8→R1 依赖）；不审实施事实、不关门
- **verdict**：**conditional**
- **完整意见**：本节即全文（未另附 attachments）

### 范围与区间

- 工作区：`workspace-001-allinme-core-api`；`root_goal=GOAL-001-allinme-core-api`；`canonical_scope=docs/workspace-001-allinme-core-api/`；`shared_materials_catalog=none`（本 scope 未声明共享资料引用）。
- 只读依据：本目标 `00-meta` / `01-decision` / `02-execution`、附件 `mvp-domain-and-api.md` / `protocol-capability-mapping.md`；对照 [GOAL-003](../GOAL-003-modular-ioc-foundation/00-meta.md)、Root 路线图与 D-008/D-009、`cmd/server/main.go` 与 `internal/*` 现状、AGENTS §6b / principles P-001～P-005。
- **未**修改任何目标 `status` / `progress` / 方案正文 / goal-tree。

### 成果（有证据）

| 主张 | 证据 |
|------|------|
| M0 方案冻结完成，I-002～I-007 已 decided | [00-meta.md](00-meta.md) 信息表；[01-decision.md](01-decision.md) D-007～D-012 |
| 协议钉死 2.4.1 且能力映射到页面清单 | Root D-006（goal-tree 摘要）；[protocol-capability-mapping.md](attachments/protocol-capability-mapping.md) |
| 三域 API/状态机有可实施清单 | [mvp-domain-and-api.md](attachments/mvp-domain-and-api.md) |
| 业务编码串行前置 GOAL-003，门禁显式 | I-009 open；D-013；goal-tree「实施等 003」 |
| progress≈20% 与「仅方案、无业务代码」一致 | [02-execution.md](02-execution.md) |
| 与 Root 路线图 R1 / R0.8 一致 | [GOAL-001 00-meta](../GOAL-001-allinme-core-api/00-meta.md) R0.8→R1 |

### 对照成功标准 / 规划质量（本 scope）

| 维度 | 评价 |
|------|------|
| P-001 可执行性 | M0 已可冻结；M2+ 在 I-009 关闭后可直接执行。整体 MVP 体量偏满，但阶段切分清楚，**不**属「无路线图硬拆」。 |
| P-005 信息门禁 | 方案层 I-002～I-007 关闭充分；**实施门禁 I-009 仍 open（正确）**；M4 协议制品本地消费方式未登记（见 F-002）。 |
| 与 GOAL-003 关系 | sibling + 显式依赖优于错误 parent 嵌套；**I-009 关闭判据未与 003 成功标准逐项对齐**（F-001）。 |
| 范围边界 | 上传/多租户/真邮件等非目标清晰；钱包余额调整细节仍略开（F-004）。 |

### Findings

| ID | 级别 | 严重度 | 说明 | 证据 / 关联 |
|----|------|--------|------|-------------|
| **F-001** | **required** | med | **I-009 关闭判据模糊**：仅写「GOAL-003 成功标准 / 用户放行」，未列出与 M2 开工业务编码对应的可勾选交接清单（例如：composition root 唯一组装、port+SQLite+fake 测试绿、模块图 active、扩展 BC 指引可读）。存在「003 勾了自己的成功标准但 002 仍不知能否开工」的门禁歧义。 | [00-meta I-009](00-meta.md)；[GOAL-003 成功标准](../GOAL-003-modular-ioc-foundation/00-meta.md)；D-013 |
| **F-002** | **required** | med | **2.4.1 制品本地消费/校验路径未登记为信息项**：成功标准要求结构校验对照钉死制品，但无 I-00N 回答制品如何落仓（vendor / CI 拉取 / 仅文档 SHA）、校验命令与失败门禁。最晚阶段 **M4**；当前 M0 冻结可不重开，但 **M4 前必须关闭或 residual 接受**。 | 成功标准「结构校验」；Root D-006 SHA；workspace `shared_materials_catalog: none` |
| **F-003** | recommended | low | **M1「依赖骨架」实为等待态**，本目标内无可交付工作包；易造成 progress 叙事空转。可改为「门禁：I-009」或把 M1 并入 M2 入口条件。 | [00-meta 路线图 M1](00-meta.md) |
| **F-004** | recommended | low | 钱包 **余额调整路径**仍写「可另定 adjust」；列表 envelope 与 `internal/response`「再定」。属实施微调，建议 M3 开工前在 decision/附件钉死一句，避免静默扩 scope。 | [mvp-domain-and-api.md](attachments/mvp-domain-and-api.md) §2、§1 列表响应 |
| **F-005** | recommended | med | **MVP 交付面偏满**（JWT+RBAC+三域 CRUD/行内/批量+仪表盘+全入口 schema）。串行 003→002 正确，但 M2～M5 无风险缓冲或「可演示最小切片」优先级。建议在 M2 前用 `/govern` 确认是否坚持全量成功标准一次验收。 | [00-meta 成功标准](00-meta.md) |

### 必改项汇总

1. **F-001**：在 I-009 / D-013（或短附件）写明与 GOAL-003 成功标准的**逐项交接清单**及「用户有界并行放行」最低书面要素。  
2. **F-002**：登记协议制品本地消费与校验信息项（建议 I-010），级别 `required`，最晚 **M4**；关闭前不得宣称 schema 校验门禁已满足。

### 与既有意见的异同

- 本文件此前无 A-00N；备注仅建议审，无 self 结论可比。  
- 与 GOAL-003 **A-001**（同轮独立审）同向：整体规划合理；共同 required 主题为 **002↔003 交接契约**。

### 结论 + 建议给编排器/用户的下一步

**结论**：GOAL-002 **当前规划总体合理**——方案冻结扎实、协议映射可实施、对 GOAL-003 的串行门禁符合 P-001/P-005/Root D-009。因 F-001/F-002，verdict = **conditional**，**不可**在未响应必改项时把「规划完全就绪 / 可无歧义进入 M2·M4」当作已通过。

**建议 `/govern`**：

1. 响应本 A-001（及 GOAL-003 A-001）；关闭或记录 F-001/F-002 处置。  
2. **优先推进 GOAL-003 S2～S4**，勿在 I-009 open 时开 GOAL-002 业务编码。  
3. P-004：已有 independent、尚无 self——询问是否要对本目标 design-plan 再做 self 审计。

### 声明

本意见不修改 status/progress；响应由 `/govern` 处理。

---

## A-002 · 编排响应 A-001（2026-07-24）

- **source**：self（编排响应，**非** independent）
- **auditor**：/govern · Grok
- **类型**：response
- **scope**：响应 A-001 全部 findings；规划补强；不审代码实施
- **verdict**：**pass**（required findings 已处置；I-009/I-010 仍 open 属预期实施门禁）

### 响应范围

- 被响应：**A-001**（independent · design-plan）
- 成对：GOAL-003 A-001 / A-002

### 关闭证据表

| Finding / 项 | 处置 | 状态 | 证据路径 |
|--------------|------|------|----------|
| A-001 **F-001** | 交接清单 H1～H7 + D-014 | **closed** | [handover-to-goal-002.md](../GOAL-003-modular-ioc-foundation/attachments/handover-to-goal-002.md)；[01-decision D-014](01-decision.md)；[00-meta I-009](00-meta.md) |
| A-001 **F-002** | 登记 I-010 + D-016 候选路径 | **closed**（finding）；**I-010 信息项仍 open** 至 M4 落地 | [00-meta I-010](00-meta.md)；D-016 |
| A-001 F-003 | M1 改为「门禁：I-009」 | **closed** | [00-meta 路线图](00-meta.md) |
| A-001 F-004 | D-015 envelope + 钱包无调账 | **closed** | D-015；[mvp-domain-and-api.md](attachments/mvp-domain-and-api.md) |
| A-001 F-005 | 保持全量标准 + 切片顺序 | **closed** | D-017 |

### 仍开放项（A-002 当时）

| 项 | 说明 |
|----|------|
| I-009 | 当时 open（**后已 verified**，见 A-003） |
| I-010 | 待 M4 |
| design-plan self | 未做（非本轮强制） |

### 结论

规划层 A-001 **required 已响应关闭**。

---

## A-003 · I-009 关闭记录（跟随 GOAL-003 关门）（2026-07-24）

- **source**：self（编排响应 / 门禁关闭，**非** independent）
- **auditor**：/govern · Grok
- **类型**：response / 信息门禁关闭
- **scope**：关闭 I-009；不审 M2 实施
- **verdict**：**pass**

### 关闭证据

| 项 | 证据 |
|----|------|
| H1～H7 | [handover-to-goal-002.md](../GOAL-003-modular-ioc-foundation/attachments/handover-to-goal-002.md) 全勾 |
| GOAL-003 done | GOAL-003 00-meta `status: done`；progress 100% |
| independent 实施审 | GOAL-003 **A-003** pass，无 required |
| self 关门 | GOAL-003 **A-004** pass |
| 编排响应 | GOAL-003 **A-005**；用户指令确认 |

### 仍开放

- **I-010**（M4）；M2～M5 业务实施未开始。

### 结论

**I-009 verified**；可 `/govern` 推进 **M2 鉴权**。

## A-004 · M3 订单首切片自审（2026-07-25）

- **source**：self
- **auditor**：Claude Code
- **类型**：design-plan
- **scope**：M3 订单首切片；审视既有目标文档、D-008/D-015/D-017、附件与当前 auth/RBAC/SQLite 分层代码，不审本次订单代码实施事实
- **verdict**：**conditional**

### 审视结论

- **A-002 是对 A-001 的 response，不是 self audit**；因此本条补足 M3 订单首切片的 self 审视记录。
- 既有路线图将 M3 切为订单→钱包→通知，且 D-017 明确切片顺序不缩减 GOAL-002 总成功标准；现有代码已具备 JWT Bearer、角色上下文、SQLite 与 handler→service→port←repository 的可接入边界。
- 订单 D-008/附件虽已有领域与端点轮廓，但在开始实施前仍须把请求前缀、成功/错误语义、RBAC、分页、CAS 乐观锁及本首切片边界钉死，避免在 handler/service/repository 间产生不一致。

### Findings

| ID | 级别 | 说明 | 处置要求 |
|----|------|------|----------|
| **F-006** | **required** | 订单首切片实施契约尚未完整钉死：`/v1` 路由、envelope/错误码、RBAC、`status/q/page/pageSize` 分页、version CAS、可编辑字段/状态机，以及 batch-delete 的事务限制需要明确。 | 在实施前新增决策并同步附件，作为 handler/service/repository 测试依据。 |

### 边界说明

本首切片不开放单项 `DELETE` 或 `refund` action；这只是订单交付顺序边界，**不等于**取消 D-008/GOAL-002 的历史总目标范围。它们保留给订单域后续补齐。

---

## A-005 · A-004 方案响应（2026-07-25）

- **source**：self
- **auditor**：Claude Code
- **类型**：response
- **scope**：关闭 A-004 F-006 的方案/实施入口契约；不审本次订单代码事实
- **verdict**：**pass**

| Finding | 处置 | 状态 | 证据 |
|---------|------|------|------|
| A-004 **F-006** | 固定 `/v1/orders` 路由、Bearer/RBAC、分页/搜索、状态机与 version CAS、batch-delete body/事务限制、envelope 及错误码；同步订单附件。 | **closed** | [01-decision.md D-018](01-decision.md)；[mvp-domain-and-api.md](attachments/mvp-domain-and-api.md) 订单节 |

**结论**：A-004 的 required finding 已由 D-018 与附件关闭；本 verdict 仅说明方案与实施入口已就绪，**不声明订单代码或测试已完成**。单项 DELETE/refund 仍保留为订单域后续范围，GOAL-002 总成功标准未缩减。

---

## A-006 · 渐进子目标路线图规划自审（2026-07-25）

- **source**：self
- **auditor**：Claude Code
- **类型**：design-plan / stage
- **scope**：GOAL-002 经 D-019/D-020 调整后的父子目标结构、完成切片补录边界、未来阶段渐进立项、信息门禁与进度口径；不审钱包实施代码
- **verdict**：**pass**
- **完整意见**：本节即全文

### 范围与区间

- 工作区：`workspace-001-allinme-core-api`；Root `GOAL-001-allinme-core-api`；canonical scope 与目标路径一致；`shared_materials_catalog: none`，本 scope 不使用共享资料。
- 对照：P-001～P-005、父目标 M0～M5、D-017～D-020、A-001～A-005、GOAL-004/005/006 五件套及更新后的 `goal-tree.md`。
- 本次仅审路线图与治理结构；GOAL-006 代码尚未实施，也未被写成已完成。

### 成果（有证据）

| 主张 | 证据 |
|------|------|
| GOAL-002 保留总范围与最终验收责任 | [00-meta.md](00-meta.md) 成功标准/路线图；D-019 |
| 已完成切片有独立边界和关门证据 | [GOAL-004](../GOAL-004-auth-rbac-menu/00-meta.md)、[GOAL-005](../GOAL-005-order-api-first-slice/00-meta.md) 及各自 A-001 |
| 订单未虚标全量完成 | GOAL-005 明确排除 DELETE/refund；父目标 M3d 保留后续 |
| 当前工作包独立且未虚构进度 | [GOAL-006](../GOAL-006-wallet-api/00-meta.md) active / 0%；execution 仅记录立项 |
| 未来目标未过早批量创建 | 父目标 M3c/M3d/M4/M5 仅保留路线图说明 |
| 树与状态一致 | [goal-tree.md](../goal-tree.md) 含 GOAL-004～006 的 parent/status/progress |

### 对照规划质量与信息门禁

| 维度 | 结论 |
|------|------|
| P-001 路线图与拆分 | **满足**：先有既有路线图，再按有独立交付/证据的阶段拆分；未来阶段渐进立项。 |
| 历史真实性 | **满足**：补录日与历史实施日分开；父目标历史未迁移或删除。 |
| 完成边界 | **满足**：GOAL-004/005 各自 close-out；GOAL-005 不冒充订单全量完成。 |
| progress 口径 | **满足**：父目标保持 50%，结构调整未计为产品进展；GOAL-006 为 0%。 |
| P-005 当前门禁 | **满足**：GOAL-006 I-001 明确 required/open，并阻断 W1～W3；GOAL-002 I-010 继续只阻断 M4 校验宣称。 |
| 审计意见响应 | **满足**：A-001 required 已由 A-002/A-003 关闭；A-004 required 已由 A-005 关闭；本次补足新版路线图同 scope 的 self 审视。 |

### Findings

| ID | 级别 | 说明 | 状态 |
|----|------|------|------|
| F-007 | recommended | 父目标长期保留历史实现细节会有一定重复；后续更新应以子目标为当前执行真相，父目标只追加摘要与链接，避免双写漂移。 | open（非阻断） |

### 必改项汇总

- **无 required / 必改 findings。**
- 当前开放 required 信息门禁不是本次路线图缺陷：GOAL-006 I-001 阻断钱包实施；GOAL-002 I-010 阻断 M4 校验宣称。

### 结论 + 建议下一步

渐进拆分符合 P-001，父子边界、历史事实、状态与信息门禁一致；新版路线图规划自审 **pass**。下一步应聚焦 **GOAL-006 I-001 钱包实施契约冻结与 design-plan 审视**，在其关闭前不进入钱包代码实施。

---

## A-007 · M3a 订单首切片实施事实独立审计（2026-07-25）

- **source**：independent
- **auditor**：Claude Code · Opus 4.8
- **类型**：execution-facts
- **scope**：GOAL-002 M3a 订单 API 首切片实施事实；核对父目标 D-018、02-execution、代码与测试，并交叉核对补录子目标 GOAL-005；不审钱包/通知、订单 DELETE/refund、Page Schema 或 GOAL-002 整体关门
- **verdict**：**conditional**
- **完整意见**：本节即全文

### 范围与区间

- 工作区：`workspace-001-allinme-core-api`；`root_goal=GOAL-001-allinme-core-api`；`canonical_scope=docs/workspace-001-allinme-core-api/`，目标及 GOAL-005 均在该范围内。
- `shared_materials_catalog: none`；本 scope 未使用或声明共享资料引用。
- 权威实施契约：本目标 [01-decision.md](01-decision.md) **D-018**；原始实施记录：[02-execution.md](02-execution.md) 2026-07-25 M3；补录边界：[GOAL-005](../GOAL-005-order-api-first-slice/00-meta.md)。
- 代码区间：`internal/domain/order.go`、`internal/port/order.go`、`internal/service/order/`、`internal/repository/sqlite/`、`internal/handler/order*`、`internal/app/app.go` 与 `internal/response/`。
- 本审计仅追加意见，未修改目标 `status` / `progress`、决策正文或 `goal-tree.md`。

### 成果（有证据）

| 主张 | 证据与复核结论 |
|------|----------------|
| 首切片七个 `/v1/orders` 端点已接线 | `internal/handler/handler.go` 注册 list/detail/create/update/mark-paid/cancel/batch-delete；无裸 `/orders` 兼容路由。 |
| Bearer 与角色边界已落实 | 同文件对全部订单路由套 `RequireAuth`；写路由另套 `RequireRoles("admin", "operator")`；`internal/handler/order_test.go` 覆盖未认证 401、viewer 写入 403 与三角色读写路径。 |
| 领域、port、service 与唯一 composition root 边界存在 | `internal/domain/order.go`、`internal/port/order.go`、`internal/service/order/service.go`；`internal/app/app.go` 注入 SQLite repository，service 未直连 `database/sql`。 |
| D-018 的分页、搜索、默认值、CAS 与 pending-only 状态机已实现 | `service.go` 校验 page/pageSize/status、默认创建 pending/CNY/version=1、PUT 与动作校验 version/state；SQLite `WHERE version=? AND status=pending` 执行 CAS。 |
| batch-delete 为事务 all-or-nothing | service 拒绝空/重复/超过 100；`internal/repository/sqlite/order.go` 先在单事务内核对全部状态，再删除并提交；测试覆盖混合状态回滚。 |
| SQLite seed 与复核修正事实可核对 | `seed.go` 在单事务内做空表检查和四状态插入；repository 测试覆盖中途失败回滚、重试、固定宽度时间戳与同秒排序。 |
| 成功 envelope 与主要错误映射存在 | `internal/response/envelope.go` 输出 `{code:0,message:"ok",data:...}`；`internal/handler/order.go` 映射 400/404/409/500 且不回传底层错误文本。 |
| 当前验证命令可重复通过 | 本轮执行 `go test -count=1 ./...` **pass**、`go vet ./...` **pass**、`git diff --check` **pass**。 |

### 对照成功标准（M3a / GOAL-005 首切片）

| 标准 | 结论 |
|------|------|
| 领域状态、Repository port、service 用例 | **达成** |
| SQLite schema、查询、CAS、事务 batch-delete、幂等种子 | **达成** |
| list/detail/create/update/mark-paid/cancel/batch-delete | **达成** |
| Bearer 与 admin/operator/viewer 读写边界 | **达成** |
| 分页、搜索、CAS、状态机与成功 envelope 的跨层证据 | **基本达成** |
| D-018 稳定错误码的跨层测试证据 | **证据不足**（F-008） |
| 本轮 test/vet/diff-check | **达成** |
| 单项 DELETE/refund 与 Page Schema | 明确范围外；父目标 M3d/M4 仍开放，未被本结论放行 |

### Findings

| ID | 级别 | 严重度 | 说明 | 证据 / 关联 |
|----|------|--------|------|-------------|
| **F-008** | **required** | med | **“错误 envelope 有跨层测试”及 D-018 稳定错误码的完成主张证据不足**：HTTP 集成测试对 404、stale version、invalid state、batch invalid state 仅断言 HTTP 状态，没有断言响应 `code` 分别为 `order_not_found`、`version_conflict`、`invalid_state`；未覆盖 `order_no_conflict` 的 HTTP code，也未通过可注入失败路径覆盖未知内部错误统一映射为 `internal` 且不泄露底层错误。当前实现静态阅读符合映射，但测试不能防止稳定错误码或泄露约束回归，因此 GOAL-002 02-execution“错误 envelope 有跨层测试”和 GOAL-005 对应成功标准不宜无条件视为已证实。 | D-018 §8；[02-execution.md](02-execution.md) 第 74 行主张；[GOAL-005 00-meta](../GOAL-005-order-api-first-slice/00-meta.md) 成功标准；`internal/handler/order_test.go` 仅状态断言；`internal/handler/order.go` `orderError` |
| **F-009** | recommended | low | `versionConflict` 将 CAS 更新 0 行统一折叠为 `version_conflict`；在 service 读取成功后、实际 UPDATE 前记录被并发删除时，HTTP 会返回 409 而非 D-018 的 404。该竞态窗口很窄，不否定首切片主体完成，但建议后续明确 repository 的“not found vs stale”判别语义并补测试。 | D-018 §8；`internal/repository/sqlite/order.go` 的 `Update` / `ChangeStatus` / `versionConflict` |

### 必改项汇总

1. **F-008**：补充 HTTP 跨层断言，至少验证 `bad_request`、`order_not_found`、`order_no_conflict`、`version_conflict`、`invalid_state` 的响应 `code`；为未知 repository/service 错误建立可注入 handler 测试，断言 500 `internal` 且响应不包含底层错误文本。完成后更新实施事实，或将现有“错误 envelope 有跨层测试”主张收窄为当前实际覆盖范围。

### 与既有意见的异同

- 与 **A-004/A-005** 同向：D-018 的实施入口契约完整且 required F-006 已在编码前关闭；本意见不重开方案 finding。
- 与 **GOAL-005 A-001 self close-out** 的主体结论一致：实现、RBAC、SQLite、CAS、事务与验证命令均有事实证据；差异是该自审把“错误 envelope”整体判定为达成，而本次独立审计发现稳定错误 `code` 的跨层断言缺口，故 verdict 从 pass 收紧为 **conditional**。
- **A-006** 审的是渐进路线图，不覆盖本次 execution-facts；两者无冲突。

### 结论 + 建议给编排器/用户的下一步

订单首切片的代码实现、SQLite 事务与种子、Bearer/RBAC、CAS 状态机以及本轮 test/vet/diff-check 均可重复核对，主体实施事实成立；但 D-018 把错误码定义为稳定契约，而现有跨层测试未验证这些 `code`，与文档“错误 envelope 有跨层测试”的完成主张存在重要证据缺口。因此本 scope verdict = **conditional**，在 F-008 关闭前不宜把 M3a 实施事实视为无条件通过，也不宜基于本意见推进 GOAL-002 整体关门。

建议由 `/govern`：

1. 汇总本 A-007 与 GOAL-005 A-001 的差异，确认以补测试关闭 F-008（推荐），或收窄成功主张并重新评估 GOAL-005 关门状态；
2. 修正后记录可核对命令与测试路径，必要时请求 `/audit GOAL-002 A-007 F-008 关闭复审`；
3. F-009 作为非阻断项排入后续订单健壮性工作。

### 声明

本意见不修改 `status` / `progress`；响应由 `/govern` 处理。

---

## A-008 · 编排响应 A-007 F-008（2026-07-25）

- **source**：self（编排响应，非 independent）
- **auditor**：/govern · Claude Code
- **类型**：response / finding-closure
- **scope**：响应 A-007 F-008；核对稳定错误码与 500 internal 不泄露测试；评估 GOAL-005 既有 done 状态；不复审 F-009，不审订单后续范围或 GOAL-002 整体关门
- **verdict**：**pass**
- **完整意见**：本节即全文

### 响应范围与用户裁决

- 被响应：**A-007 F-008**（independent，required / med）。
- 既有差异：GOAL-005 A-001 self close-out 为 pass；A-007 对相同首切片实施事实收紧为 conditional，指出错误 `code` 测试证据不足。
- 用户本轮书面指令选择按 A-007 建议**补齐测试并留痕评估 done 状态**，未选择忽略 independent finding 或接受残余风险；该修正路径消解了结论差异。
- 工作区、Root Goal 与 canonical scope 仍匹配；本 scope 不使用共享资料。I-010 仅阻断 M4，与本次 M3a finding 关闭无关。

### 关闭证据表

| Finding / 项 | 状态 | 修正与证据 |
|--------------|------|------------|
| A-007 **F-008**：400/404/409 稳定错误 `code` 缺少跨层断言 | **closed** | `internal/handler/order_test.go` 新增 `bad_request`、`order_not_found`、`order_no_conflict`、`version_conflict`、`invalid_state` 的 HTTP 响应 code 断言。 |
| A-007 **F-008**：未知内部错误未验证统一 `internal` 且不泄露 | **closed** | `internal/handler/order.go` 以本地 `orderService` 接口支持 handler 依赖注入；新增 `internal/handler/order_internal_test.go`，注入含敏感路径的失败错误并断言 HTTP 500、`code=internal`、响应不含底层错误文本。 |
| 修正后验证 | **pass** | `gofmt`；`go test -count=1 ./...`；`go vet ./...`；`git diff --check` 均于 2026-07-25 通过；事实写入 [02-execution.md](02-execution.md)。 |
| GOAL-005 状态留痕 | **done 保持** | [GOAL-005 02-execution](../GOAL-005-order-api-first-slice/02-execution.md) 记录关门后 finding 与修正；[GOAL-005 A-002](../GOAL-005-order-api-first-slice/03-audit.md) 记录关闭复核。无 status/progress 变化。 |

### GOAL-005 done 状态评估

1. A-007 确认首切片端点、SQLite、事务、RBAC、CAS 与主体实现事实成立；F-008 针对的是 D-018 错误契约的**测试证据缺口**，并未发现产品范围或实现主路径缺失。
2. P-003 下 required finding 在开放期间不得作为放行依据；本轮先完成修正和验证，再形成关闭记录，未用 GOAL-005 的既有 `done` 绕过该门禁推进 GOAL-002 关门。
3. 修正未改变 GOAL-005 成功边界或产品进度；在 F-008 有证据关闭后，GOAL-005 可继续保持 `done` / 100%。因 status/progress/parent 均未变化，`goal-tree.md` 无需更新。
4. 若修正未通过验证，正确处置应为恢复开放 F-008 并重新评估 GOAL-005 状态；本轮未触发该分支。

### 仍开放项

| 项 | 级别 | 状态 | 影响 |
|----|------|------|------|
| A-007 **F-009** | recommended / low | open | 非阻断；后续订单健壮性工作明确并发删除时 404 vs 409 语义。 |
| **I-010** | required | open | 仅阻断 M4 协议制品校验宣称；不影响本次 M3a F-008 关闭。 |

### 结论

A-007 F-008 的两部分证据缺口均已由可重复测试关闭；A-007 的 execution-facts 门禁可由 conditional 转为响应后的 **pass**。GOAL-005 保持 `done` / 100%，并已在父子目标 execution/audit 中留下“关门后 finding → 修正 → 验证 → 关闭”的完整轨迹。本响应不放行 GOAL-002 整体关门，也不关闭 F-009 或 I-010。

---

## A-009 · A-007 F-008 关闭独立复审（2026-07-25）

- **source**：independent
- **auditor**：Claude Code · Opus 4.8
- **类型**：finding-closure
- **scope**：仅复审 A-007 F-008 的关闭证据与 A-008 响应真实性；核对稳定错误 code、500 internal 不泄露、验证命令及 GOAL-005 done 留痕；不复审 F-009，不审订单后续范围、钱包、M4 或 GOAL-002 整体关门
- **verdict**：**pass**
- **完整意见**：本节即全文

### 范围与区间

- 工作区：`workspace-001-allinme-core-api`；Root `GOAL-001-allinme-core-api`；canonical scope 与目标路径匹配。
- `shared_materials_catalog: none`；本 scope 未使用共享资料引用。
- 被复审意见：A-007 F-008（required / med）；响应记录：A-008；子目标留痕：GOAL-005 A-002 与其 02-execution。
- I-010 仅阻断 M4 协议制品校验；与本次 M3a 错误契约测试关闭无关。F-009 明确排除在本复审 scope 外。
- 本意见未修改任何目标 `status` / `progress`、决策正文或 `goal-tree.md`。

### 关闭证据复核

| F-008 要求 | 证据 | 复核结果 |
|------------|------|----------|
| HTTP 跨层断言 `bad_request` | `internal/handler/order_test.go`：未知 JSON 字段与极大分页均断言 HTTP 400 + `code=bad_request` | **满足** |
| HTTP 跨层断言 `order_not_found` | 同文件：不存在订单详情断言 HTTP 404 + `code=order_not_found` | **满足** |
| HTTP 跨层断言 `order_no_conflict` | 同文件：重复 `ORD-HTTP` 创建断言 HTTP 409 + `code=order_no_conflict` | **满足** |
| HTTP 跨层断言 `version_conflict` | 同文件：stale version PUT 断言 HTTP 409 + `code=version_conflict` | **满足** |
| HTTP 跨层断言 `invalid_state` | 同文件：paid 后 cancel 与混合状态 batch-delete 均断言 HTTP 409 + `code=invalid_state` | **满足** |
| 未知内部错误可注入 | `internal/handler/order.go` 的本地 `orderService` 接口允许 handler 测试注入失败 service；生产 `*order.Service` 仍满足接口 | **满足** |
| 500 `internal` 且不泄露底层错误 | `internal/handler/order_internal_test.go` 注入含 `C:\\secret\\orders.db` 的 SQLite 错误，断言 HTTP 500、`code=internal`，且响应不含完整错误或敏感路径 | **满足** |
| 修正事实与状态留痕 | GOAL-002 02-execution、A-008；GOAL-005 02-execution、A-002 对关门后 finding、修正、验证及 done 保留依据互相指回 | **满足** |

### 可重复验证

本轮独立复审重新执行：

- `go test -count=1 ./internal/handler`：**pass**
- `go test -count=1 ./...`：**pass**
- `go vet ./...`：**pass**
- `git diff --check`：**pass**

### Findings

- **无新增 required 或 recommended finding。**
- A-007 **F-008**：**closed（独立复审确认）**。
- A-007 **F-009**：仍为 recommended / low / open；不在本次复审范围，不影响 F-008 关闭结论。

### 与既有意见的异同

- 与 **A-008 self response** 同向：其关闭表中的代码路径、测试覆盖和命令结果均可重复核对。
- 与 **GOAL-005 A-002 self 关闭复核** 同向：F-008 是测试证据缺口，修正后不改变首切片成功边界；保留 `done` / 100% 有可追踪依据。
- A-007 原 conditional verdict 的唯一 required F-008 已关闭；历史 verdict 保留不改，由本 A-009 记录关闭后的独立结论。

### 必改项汇总

- **无。**

### 结论 + 建议给编排器/用户的下一步

A-007 F-008 的全部关闭要求均有代码、跨层测试、内部错误不泄露测试和可重复命令证据，A-008 的关闭声明真实充分。**F-008 独立关闭复审 verdict = pass**；该 finding 不再阻断 M3a 实施事实或 GOAL-005 首切片关门状态。

本结论不关闭 F-009、I-010，不放行 GOAL-002 整体关门。建议由 `/govern` 汇总 A-009，并继续按当前路线图处理 GOAL-006 钱包契约门禁，或另行安排非阻断 F-009。

### 声明

本意见不修改 `status` / `progress`；响应由 `/govern` 处理。

---

## 备注

- 2026-07-24：A-001 independent；A-002 govern 响应；A-003 I-009 关闭。
- 2026-07-25：A-004 self design-plan；已由 A-005 以 D-018 与附件关闭其 required finding。
- 2026-07-25：A-006 self design-plan 审视渐进子目标路线图，pass，无 required；GOAL-006 I-001 仍作为钱包实施门禁。
- 2026-07-25：A-007 independent execution-facts 审计订单首切片，conditional；提出 F-008 required 与 F-009 recommended。
- 2026-07-25：A-008 `/govern` 响应以稳定错误码和 internal 不泄露测试关闭 F-008；GOAL-005 保持 done，F-009 仍开放。
- 2026-07-25：A-009 independent finding-closure 复审确认 F-008 closed / pass，无新增 finding。
