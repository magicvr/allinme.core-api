---
id: GOAL-006-wallet-api
doc: decision
status: active
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.2.0
---

# 决策记录 · GOAL-006

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。

- **I-001**：required / verified，D-003 固定跨层契约并经 A-001 design-plan 自审 pass；W1～W3 门禁已解除。
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

## D-003 · 钱包 API 首切片跨层实施契约（I-001）

**日期**：2026-07-25
**状态**：`accepted`
**关联**：I-001；父目标 D-008 / D-009 / D-011 / D-015 / D-017；本目标 D-001 / D-002

### 1. 路由与首切片边界

全部路由使用 `/v1/wallets`，不提供裸 `/wallets` 兼容路径：

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/v1/wallets` | 列表、筛选、分页 |
| GET | `/v1/wallets/{id}` | 详情 |
| POST | `/v1/wallets` | 创建 active 钱包 |
| PUT | `/v1/wallets/{id}` | 仅更新 ownerName，携带 version |
| POST | `/v1/wallets/{id}/freeze` | active → frozen，携带 version |
| POST | `/v1/wallets/{id}/unfreeze` | frozen → active，携带 version |
| POST | `/v1/wallets/batch-freeze` | 批量冻结 active 钱包，事务 all-or-nothing |

本首切片不提供 DELETE、batch-unfreeze、调账、充值、提现或支付网关。

### 2. 领域字段与创建契约

钱包响应模型：

| 字段 | 规则 |
|------|------|
| `id` | 服务端生成的字符串主键 |
| `accountNo` | 必填、trim 后非空、全局唯一；创建后不可修改 |
| `ownerName` | 必填、trim 后非空；唯一可由 PUT 修改的业务字段 |
| `balanceCents` | int64，创建时可填，默认 `0`，必须 `>= 0`；创建后不可修改 |
| `currency` | trim 后转大写，须为三位 A-Z 字母，默认 `CNY`；创建后不可修改 |
| `status` | `active` \| `frozen`；创建固定为 `active` |
| `version` | 创建固定为 `1`；成功写入后 `+1` |
| `createdAt` / `updatedAt` | UTC RFC3339 时间；成功写入刷新 `updatedAt` |

`POST /v1/wallets` 请求字段为 `accountNo`、`ownerName`、可选 `balanceCents`、可选 `currency`。客户端不得指定 `id`、`status`、`version` 或时间字段；未知 JSON 字段按 `bad_request` 拒绝。

### 3. 更新与状态 CAS

- `PUT /v1/wallets/{id}` 请求仅含 `version`、`ownerName`；`version >= 1`，active/frozen 两种状态均可更新 ownerName。
- PUT 不接受或修改 `accountNo`、`balanceCents`、`currency`、`status`；出现这些字段按未知字段拒绝。
- freeze/unfreeze body 均为 `{ "version": <int64> }`。
- freeze 仅允许 `active → frozen`；unfreeze 仅允许 `frozen → active`。
- PUT、freeze、unfreeze 均以 `id + version` 做 repository CAS；成功后 version 加 1。
- 目标不存在返回 `wallet_not_found`；陈旧 version 返回 `version_conflict`；状态方向不合法返回 `invalid_state`。

### 4. 列表契约

`GET /v1/wallets` 支持：

| 参数 | 规则 |
|------|------|
| `status` | 可空；非空时仅 `active` / `frozen` |
| `q` | 对 `accountNo` 或 `ownerName` 做字面包含匹配；SQLite LIKE 必须转义 `%` / `_` |
| `page` | 默认 1，必须 `>= 1` |
| `pageSize` | 默认 20，范围 1～100 |

分页 offset 计算必须拒绝 int 溢出。列表稳定排序为 `createdAt DESC, id DESC`。

### 5. batch-freeze 原子性

- body 固定为 `{ "ids": ["..."] }`，继承父目标附件；不另加 versions map。
- `ids` 必须 1～100 个；trim 后非空且不得重复。
- repository 在单一 SQLite transaction 中先核对全部目标，再统一更新。
- 所有目标必须存在且当前为 `active`；任一不存在或已 frozen，整批不变更。
- 成功时全部变为 `frozen`、各自 version 加 1、刷新 `updatedAt`，返回 `data={"frozen": n}`。
- 不支持部分成功，符合父目标 D-004。

### 6. 鉴权与 RBAC

- 全部钱包路由必须 Bearer。
- `viewer` 可访问 list/detail，不可写。
- `admin`、`operator` 可 create/update/freeze/unfreeze/batch-freeze。
- 后端 handler 独立执行角色校验，不依赖未来 Page Schema permissions。

### 7. 响应与错误语义

所有成功响应使用 HTTP 200 + 既有 envelope：

- list：`data={"list": [...], "total": <int>}`
- detail/create/update/freeze/unfreeze：`data=<wallet>`
- batch-freeze：`data={"frozen": <int>}`

稳定错误：

| HTTP | code | 场景 |
|------|------|------|
| 400 | `bad_request` | JSON、字段、分页、金额、币种、ids 等输入不合法 |
| 404 | `wallet_not_found` | 单项或批量目标不存在 |
| 409 | `account_no_conflict` | accountNo 重复 |
| 409 | `version_conflict` | PUT/freeze/unfreeze CAS 版本陈旧 |
| 409 | `invalid_state` | freeze/unfreeze 方向或 batch-freeze 状态不合法 |
| 500 | `internal` | 未知内部错误；不得泄露底层错误文本 |

JSON body 上限 1 MiB，拒绝未知字段和尾随第二个 JSON 值。

### 8. SQLite、种子与测试入口

- `wallets` schema 由 SQLite adapter 拥有；service 仅依赖 WalletRepository port。时间戳以固定宽度 UTC 纳秒文本保存，确保 `TEXT` 排序与领域时间顺序一致。
- 空表 seed 在单一 transaction 内完成，至少包含一条 active、一条 frozen；失败须全回滚且可重试；非空表不重复插入。
- W1～W4 测试至少覆盖：默认值/输入校验、list/filter/pagination/溢出、accountNo unique、PUT 不改余额、CAS、freeze/unfreeze 状态机、batch-freeze 回滚、seed 幂等与失败回滚、Bearer/RBAC、成功 envelope、全部稳定错误 code 与 internal 不泄露。

**为什么**：沿用订单首切片已验证的端口、CAS、事务、RBAC、envelope 与错误契约模式，同时把钱包特有的余额不可变、双向冻结状态与批量冻结边界固定为可直接测试的实现入口。

**未选方案**：

- accountNo/currency/balance 的通用 PUT：会破坏账户标识与余额语义，超出“仅元数据更新”。
- batch-freeze 携带版本 map：与父目标已固定 `{ids}` body 不一致，MVP 复杂度收益不足。
- 幂等 freeze/unfreeze（目标状态相同时仍 200）：会掩盖客户端状态陈旧；本契约选择 `invalid_state`。
- 部分成功批量：父目标明确不做。

**后续**：完成 design-plan 自审；通过后将 I-001 标为 `verified`，W0 完成，方可进入 W1。
