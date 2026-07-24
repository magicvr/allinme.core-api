---
id: GOAL-007-notification-api
doc: decision
status: active
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.2.0
---

# 决策记录 · GOAL-007

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。

- **I-001**：required / **verified**，D-003 固定跨层契约并经 A-001 design-plan 自审 pass；N1～N3 门禁已解除。
- **I-002**：non-blocking / open，由 GOAL-002 M4 处理。

## D-001 · 继承父目标通知范围与通用约束

**日期**：2026-07-25
**状态**：`accepted`

**决定**：通知业务范围继承 GOAL-002 D-004、D-008、D-009、D-011、D-015、D-017 及附件 [mvp-domain-and-api.md](../GOAL-002-mvp-demo-admin/attachments/mvp-domain-and-api.md)：

- 状态：`draft → published → archived`。
- channel：`inbox` \| `email`（枚举）；**不真发邮件**。
- 端点轮廓：list/detail/create/update/delete/publish/batch-archive。
- viewer 只读；admin/operator 可写；SQLite 默认且 service 依赖 port。
- 列表使用 `data.list` / `data.total` envelope。

**为什么**：父目标方案冻结与三域清单已闭环，子目标不得静默扩张为真实消息通道或工作流引擎。

**未选方案**：接入 SMTP/短信网关或用户收件箱推送——超出 MVP 演示边界（D-004）。

## D-002 · 先关闭 I-001 再实施

**日期**：2026-07-25
**状态**：`accepted`

**决定**：在开始通知领域、repository 或 handler 实施前，先记录可直接测试的跨层契约（本目标 D-003），并追加 design-plan 自审。契约至少覆盖路由、请求字段、列表参数、version CAS、publish/delete 状态转换、batch-archive 原子性、RBAC 与错误语义。自审 pass 后将 I-001 标为 `verified`，方可进入 N1。

**为什么**：沿用订单 D-018 / 钱包 D-003 的有效做法，防止各层自行猜测导致不一致。

## D-003 · 通知 API 首切片跨层实施契约（I-001）

**日期**：2026-07-25
**状态**：`accepted`
**关联**：I-001；父目标 D-004 / D-008 / D-009 / D-011 / D-015 / D-017；本目标 D-001 / D-002

### 1. 路由与首切片边界

全部路由使用 `/v1/notifications`，不提供裸 `/notifications` 兼容路径：

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/v1/notifications` | 列表、筛选、分页 |
| GET | `/v1/notifications/{id}` | 详情 |
| POST | `/v1/notifications` | 创建 draft 通知 |
| PUT | `/v1/notifications/{id}` | 仅 draft 可更新内容字段，携带 version |
| DELETE | `/v1/notifications/{id}` | 仅 draft 可删，携带 version（query 或 body，见 §3） |
| POST | `/v1/notifications/{id}/publish` | draft → published，携带 version |
| POST | `/v1/notifications/batch-archive` | 批量归档 published 通知，事务 all-or-nothing |

本首切片不提供 unpublish、restore、单项 archive action、真实邮件/短信发送或用户级收件箱已读状态。

### 2. 领域字段与创建契约

通知响应模型：

| 字段 | 规则 |
|------|------|
| `id` | 服务端生成的字符串主键 |
| `title` | 必填、trim 后非空；draft 可由 PUT 修改 |
| `body` | 必填字符串（允许 trim 后为空串）；draft 可由 PUT 修改 |
| `channel` | `inbox` \| `email`；创建默认 `inbox`；draft 可由 PUT 修改；**不**触发外发 |
| `status` | `draft` \| `published` \| `archived`；创建固定为 `draft` |
| `version` | 创建固定为 `1`；成功写入后 `+1` |
| `publishedAt` | 可空；仅在 publish 成功时写入 UTC RFC3339；创建与 draft 更新保持 `null` |
| `createdAt` / `updatedAt` | UTC RFC3339；成功写入刷新 `updatedAt` |

`POST /v1/notifications` 请求字段为 `title`、`body`、可选 `channel`。客户端不得指定 `id`、`status`、`version`、`publishedAt` 或时间字段；未知 JSON 字段按 `bad_request` 拒绝。

`channel` 非法值（非 `inbox`/`email`）→ `bad_request`。

### 3. 更新、删除与 publish CAS

- `PUT /v1/notifications/{id}` 请求含 `version`、`title`、`body`、可选 `channel`；`version >= 1`。
- PUT **仅允许** `status=draft`；published/archived 上 PUT → `invalid_state`。
- PUT 不接受或修改 `status`、`publishedAt`；出现这些字段按未知字段拒绝。
- `POST .../publish` body 固定 `{ "version": <int64> }`；仅允许 `draft → published`；成功时写入 `publishedAt=now`、version+1。
- `DELETE /v1/notifications/{id}`：请求携带 version——**优先** JSON body `{ "version": <int64> }`（与写动作一致）；若无 body，则接受 query `?version=`。仅允许删除 `draft`；published/archived 删除 → `invalid_state`。成功 HTTP 200，`data={"deleted": true}`（或等价布尔成功体，实现时固定一种并在测试锁定）。
- PUT / publish / DELETE 均以 `id + version` 做 repository CAS；成功写入后 version 加 1（DELETE 成功则行消失，无需 +1 语义对外暴露）。
- 目标不存在 → `notification_not_found`；陈旧 version → `version_conflict`；状态方向不合法 → `invalid_state`。

### 4. 列表契约

`GET /v1/notifications` 支持：

| 参数 | 规则 |
|------|------|
| `status` | 可空；非空时仅 `draft` / `published` / `archived` |
| `channel` | 可空；非空时仅 `inbox` / `email` |
| `q` | 对 `title` 做字面包含匹配；SQLite LIKE 必须转义 `%` / `_` |
| `page` | 默认 1，必须 `>= 1` |
| `pageSize` | 默认 20，范围 1～100 |

分页 offset 计算必须拒绝 int 溢出。列表稳定排序为 `createdAt DESC, id DESC`。

### 5. batch-archive 原子性

- body 固定为 `{ "ids": ["..."] }`，与订单/钱包批量模式一致；不另加 versions map。
- `ids` 必须 1～100 个；trim 后非空且不得重复。
- repository 在单一 SQLite transaction 中先核对全部目标，再统一更新。
- 所有目标必须存在且当前为 `published`；任一不存在、为 draft 或已 archived，整批不变更。
- 成功时全部变为 `archived`、各自 version 加 1、刷新 `updatedAt`，`publishedAt` 保持不变，返回 `data={"archived": n}`。
- 不支持部分成功，符合父目标 D-004。

### 6. 鉴权与 RBAC

- 全部通知路由必须 Bearer。
- `viewer` 可访问 list/detail，不可写。
- `admin`、`operator` 可 create/update/delete/publish/batch-archive。
- 后端 handler 独立执行角色校验，不依赖未来 Page Schema permissions。

### 7. 响应与错误语义

所有成功响应使用 HTTP 200 + 既有 envelope：

- list：`data={"list": [...], "total": <int>}`
- detail/create/update/publish：`data=<notification>`
- delete：`data={"deleted": true}`
- batch-archive：`data={"archived": <int>}`

稳定错误：

| HTTP | code | 场景 |
|------|------|------|
| 400 | `bad_request` | JSON、字段、分页、channel、ids 等输入不合法 |
| 404 | `notification_not_found` | 单项或批量目标不存在 |
| 409 | `version_conflict` | PUT/publish/DELETE CAS 版本陈旧 |
| 409 | `invalid_state` | 非 draft 更新/删除、非 draft publish、batch 状态不合法 |
| 500 | `internal` | 未知内部错误；不得泄露底层错误文本 |

JSON body 上限 1 MiB，拒绝未知字段和尾随第二个 JSON 值。

### 8. SQLite、种子与测试入口

- `notifications` schema 由 SQLite adapter 拥有；service 仅依赖 NotificationRepository port。时间戳以固定宽度 UTC 纳秒文本保存，确保 `TEXT` 排序与领域时间顺序一致。
- 空表 seed 在单一 transaction 内完成，至少包含 draft、published、archived 各一条；失败须全回滚且可重试；非空表不重复插入。
- N1～N4 测试至少覆盖：默认值/输入校验、list/filter（status/channel/q）/pagination/溢出、LIKE 字面转义、PUT 仅 draft、publish 写 publishedAt、DELETE 仅 draft、CAS、batch-archive 回滚、seed 幂等与失败回滚、Bearer/RBAC、成功 envelope、全部稳定错误 code 与 internal 不泄露。

**为什么**：把父目标通知轮廓钉成与订单/钱包同构的可测试跨层契约，同时固定「不真发邮件」「线性状态机」「batch 仅 published→archived」等演示边界。

**未选方案**：

- 真邮件发送或 outbox 投递：D-004 明确不做。
- unpublish / restore / 从 archived 再编辑：扩大状态机，MVP 无必要。
- batch-archive 携带 version map：与既有 `{ids}` 批量约定不一致。
- 允许删除 published：与「先归档再清理」演示路径冲突；首切片删除仅 draft。
- 部分成功批量：父目标明确不做。

**后续**：A-001 design-plan 自审 pass 后 I-001 已 verified；N0 完成，N1 已实施（见 execution）。
