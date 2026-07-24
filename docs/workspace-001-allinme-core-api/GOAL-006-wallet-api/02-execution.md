---
id: GOAL-006-wallet-api
doc: execution
status: done
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.6.0
---

# 执行记录 · GOAL-006

## 时间线

### 2026-07-25 · 目标立项

- 从 GOAL-002 M3 渐进拆分出钱包 API 独立工作包。
- 登记成功标准、范围边界与 W0～W4 路线图。
- 登记 I-001 required 实施契约门禁；当前状态 `open`，尚未进入钱包代码实施。
- 继承父目标已确认的钱包轮廓与 IoC/RBAC/SQLite/envelope 约束（D-001）。

### 2026-07-25 · W0 钱包实施契约冻结与自审

- 记录 D-003，固定七个 `/v1/wallets` 端点及首切片范围。
- 固定钱包模型、创建默认值、PUT 仅 ownerName、balance/accountNo/currency 不可变、active/frozen 状态与 version CAS。
- 固定 status/q/page/pageSize 列表契约、稳定排序、LIKE 字面匹配与分页溢出拒绝。
- 固定 freeze/unfreeze 状态方向；batch-freeze `{ids}` 1～100、单事务预检与 all-or-nothing。
- 固定 Bearer/RBAC、成功 envelope、400/404/409/500 稳定错误 code、internal 不泄露及 1 MiB JSON 边界。
- 固定 SQLite 时间戳、事务 seed 与 W1～W4 最小测试覆盖。
- 完成 A-001 design-plan 自审，verdict `pass`，无 required/recommended finding；I-001 标记 `verified`，W0 完成，W1～W3 门禁解除。
- 尚未创建或修改钱包产品代码；progress 保持 0%。

### 2026-07-25 · W1 领域、port、service 与接口级测试完成

**实现事实**：

| 路径 | 说明 |
|------|------|
| `internal/domain/wallet.go` | Wallet 聚合、active/frozen 状态及已知状态校验。 |
| `internal/port/wallet.go` | WalletRepository port、列表筛选及 wallet-specific not-found/accountNo/version/state/input 稳定错误。 |
| `internal/service/wallet/service.go` | 可注入时钟/ID；list/get/create/update/freeze/unfreeze/batch-freeze 用例；输入、分页溢出、币种、CAS、状态与批量 IDs 校验。 |
| `internal/service/wallet/service_test.go` | fake repository 接口级测试，覆盖默认 CNY/自定义币种、负余额/字段校验、accountNo 冲突、active/frozen owner 更新与不可变字段、version CAS、freeze/unfreeze 状态机、batch 原子回滚/去重/上限、list/get 输入。 |

**D-003 对齐**：

- 创建固定 active/version=1，余额默认 0，币种 trim 后大写并校验三位 A-Z。
- UpdateInput 仅暴露 version + ownerName；返回结果保持 accountNo/balance/currency/status/createdAt 不变，active/frozen 均可更新。
- freeze/unfreeze 使用期望状态 + version 的 repository CAS 入口；成功 version+1。
- batch-freeze 在 service 层完成 1～100、trim、非空和去重，向 repository 传递规范化副本；原调用 ids 不被修改。
- service/domain 未依赖 SQLite 或 HTTP 具体实现。

**验证事实**：已运行 `gofmt`；`go test -count=1 ./internal/service/wallet` **pass**；`go test -count=1 ./...` **pass**；`go vet ./...` **pass**；`git diff --check` 与 `git diff --cached --check` **pass**。

**边界**：W2 SQLite schema/repository/seed、W3 HTTP/RBAC 尚未实施。progress 调整为 **20%**，仅计入已完成的 W1 产品切片。

### 2026-07-25 · W2 SQLite Repository 与事务种子完成

**实现事实**：

| 路径 | 说明 |
|------|------|
| `internal/repository/sqlite/db.go` | 新增 `wallets` schema；accountNo unique，balance/version/status 具备数据库约束。 |
| `internal/repository/sqlite/wallet.go` | WalletRepository SQLite adapter：Create/Get/List、owner-only CAS、状态 CAS、事务 batch-freeze、Count、固定宽度 UTC 纳秒时间戳。 |
| `internal/repository/sqlite/seed.go` | 新增 `SeedWallets`；空表检查与 active/frozen 两条种子在单一 transaction 中完成，失败全回滚、非空不重复。 |
| `internal/repository/sqlite/wallet_test.go` | repository 与 seed 测试覆盖回滚重试、幂等、状态/q/分页、LIKE `%`/`_` 字面转义、时间排序、accountNo unique、owner CAS、状态 CAS/错误分类、batch 原子回滚。 |

**D-003 对齐与复核**：

- list 使用 `created_at DESC, id DESC`，offset 计算前拒绝 int 溢出。
- q 参数对 `\\`、`%`、`_` 做 SQLite LIKE escape；测试证明 `%`/`_` 作为字面字符，不会退化为全量通配。
- owner update 只更新 `owner_name`、version、updated_at，active/frozen 均可执行；余额、accountNo、currency、status、createdAt 保持不变。
- CAS 0 行会进一步区分 `wallet_not_found`、`version_conflict` 与 `invalid_state`，避免统一折叠。
- batch-freeze 在同一 transaction 中先核对全部目标存在且 active，再统一更新；任一 missing/frozen 均无部分变更。
- 时间戳固定为 UTC 九位纳秒文本，测试覆盖同秒 0ns/100ms 排序与存储格式。

**验证事实**：`go test -count=1 ./internal/repository/sqlite` **pass**；`go test -count=1 ./...` **pass**；`go vet ./...` **pass**；`git diff --check` 与 `git diff --cached --check` **pass**。

**边界**：`SeedWallets` 尚未接入 composition root；钱包 HTTP/RBAC、请求边界与跨层集成测试仍属于 W3。progress 调整为 **50%**。

### 2026-07-25 · W3 composition root、HTTP/RBAC 与跨层测试完成

**实现事实**：

| 路径 | 说明 |
|------|------|
| `internal/app/app.go` | 唯一 composition root 构造 SQLite WalletRepository，启动时运行 SeedWallets，注入 wallet service，并暴露于 App/handler deps。 |
| `internal/handler/handler.go` | 注册七个 `/v1/wallets` 路由；全部 Bearer，GET 三角色可读，写路由仅 admin/operator。 |
| `internal/handler/wallet.go` | list/detail/create/update/freeze/unfreeze/batch-freeze HTTP 适配；1 MiB body、未知字段/尾随 JSON 拒绝、分页解析、成功 envelope 与六类稳定错误映射。 |
| `internal/handler/wallet_test.go` | 完整 app→JWT→handler→service→SQLite 集成：启动 seed、RBAC、创建/筛选/详情、owner-only 更新、frozen owner 更新、CAS、状态动作、batch 回滚/成功及错误 code。 |
| `internal/handler/wallet_internal_test.go` | 注入含敏感 SQLite 路径的未知 service 错误，验证 HTTP 500 / `internal` 且响应不泄露底层文本。 |

**D-003 对齐与验证范围**：

- 成功响应：list `data.list/data.total`；单项 `data=wallet`；batch `data.frozen`。
- 稳定错误跨层断言：`bad_request`、`wallet_not_found`、`account_no_conflict`、`version_conflict`、`invalid_state`、`internal`。
- PUT 请求类型只暴露 version/ownerName；携带 balance 等不可编辑字段因 `DisallowUnknownFields` 返回 400。
- HTTP 测试确认 owner 更新不改变 balance/currency/status；active/frozen 两状态均可更新 owner。
- batch-freeze 对 mixed frozen、missing 均回滚，对成功目标全部冻结；重复 IDs 返回 bad_request。
- 请求边界覆盖未知字段、尾随第二个 JSON、非法 status、极大分页和超过 1 MiB body。

**验证事实**：`go test -count=1 ./internal/handler` **pass**；`go test -count=1 ./...` **pass**；`go vet ./...` **pass**；`git diff --check` 与 `git diff --cached --check` **pass**。

**边界**：全部产品成功标准已有当前证据，但尚未执行 W4 execution-facts/close-out 自审，因此目标保持 `active`，progress 调整为 **90%**，不提前标记 done。

### 2026-07-25 · W4 最终验证、实施事实自审与关门

- 重新执行 wallet service/SQLite/handler targeted tests、全量 `go test -count=1 ./...`、`go vet ./...`、staged/unstaged `git diff --check`，全部 **pass**。
- 额外 targeted `go test -race` 因本机 Windows `runtime/cgo` 的 `cgo.exe: exit status 2` 未完成；不属于成功标准，作为环境验证限制写入 A-002，未伪记为 pass。
- 完成 A-002 execution-facts / close-out 自审：对照 D-003、全部成功标准、I-001/I-002、代码和测试，verdict **pass**，无 required/recommended finding。
- I-001 verified；I-002 non-blocking/open 且属于父目标 M4，不阻断本目标关门。
- GOAL-006 更新为 `done` / 100%；父目标摘要与 `goal-tree.md` 同步。

## 待办

- 本目标内无待办。
- 父目标后续：通知 API、订单 DELETE/refund、M4 Page Schema/协议制品校验及 MVP 集成验收。

## 进度评估

**100%**：W0～W4、全部成功标准、required 信息门禁、强制验证与 close-out 自审均已闭环；目标已关门。
