---
status: closed
audit_id: AUD-0003
auditor: GitHub Copilot
audit_type: targeted
scope: plan:PLN-0005
subject: phase-05-attachment-lifecycle
baseline: git:912ce322bcde15fffcef913b81c7d50088e11853; worktree:clean
started_at: 2026-07-14T03:37:56+08:00
completed_at: 2026-07-14T03:51:00+08:00
last_updated: 2026-07-14
related_audits: AUD-0001, AUD-0002
supersedes: none
related_plans: PLN-0005
---

# 阶段五附件生命周期计划审计

## 目的与范围

审计 `PLN-0005` 及其 checklist 的正确性、完整性、可执行性，以及与路线图、事实源和当前实现的兼容性。本记录不代表全仓质量审计。

## 基线与方法

- 基线：`main@912ce322bcde15fffcef913b81c7d50088e11853`，审计开始时工作树干净。
- 对象：`docs/plans/PLN-0005-phase-05-attachment-lifecycle.md` 及同号 checklist。
- 方法：核对计划生命周期、事实源、当前代码、测试、迁移、CI、Evidence、工作分解及完成门禁；只读取与计划假设和依赖直接相关的源码范围。
- 身份与生命周期：plan/checklist 文件名、主题、`plan_id: PLN-0005`、`status: active` 和计划索引一致；两份文件互相链接，当前无已勾选 checklist 项。
- 事实基线：当前为 Go 1.26、schema v6；`internal/files`、附件 route 和 phase 5 capability 尚未实现，计划通过 P0 No-Go 前置明确区分当前事实与未来契约。

## 历史关系

- `AUD-0001` 已解决计划与审计治理结构问题，并将本计划登记为当前唯一活跃计划；本轮复核其治理结论在当前 baseline 上是否仍成立，同时审查此前未覆盖的计划内容质量。
- `AUD-0001-F001`、`AUD-0001-F002`、`AUD-0001-F003` 在当前 baseline 上仍保持 resolved：计划与审计目录、编号、记录生命周期和 validator 规则仍有效。
- `AUD-0002` 在本轮开始后以相同 baseline 并发创建，关闭本记录时仍为 open 且 Findings 为“审计进行中”；本轮不覆盖该记录，也没有可比较的相反结论。后续关闭 `AUD-0002` 时应引用本记录并解释重复或差异。

## Findings

### AUD-0003-F001 - ORDER_DELETE 未冻结订单创建幂等历史的处置语义

- 受影响计划：`PLN-0005`。
- Severity: high。
- Evidence: 计划第 3 节第 26 项、第 4.1/4.3 节和 checklist `P0-1`、`P0-2`、`M3B-6` 只要求在存在 `refunds` 或 `refund_idempotency_keys` 时拒绝 ORDER_DELETE，并声明“只有无退款历史的订单才允许进入清理原语”；最终事务要求删除订单聚合行。当前 `internal/store/migrations/0004_idempotency.sql` 的 `idempotency_keys.order_id` 没有外键，但持久保存 create snapshot，并由 `internal/store/orders.go` 按 scope/key 直接重放；计划同时要求 v1/v2 snapshot 历史永不改写且重放不查询当前订单。计划没有说明订单删除成功后相关 `idempotency_keys` 是保留、删除、墓碑化还是使删除前置拒绝，也没有定义随后相同 key 的重放结果。
- Impact: 实现者可能保留能重放已删除订单的 create snapshot，也可能删除幂等历史并改变相同 key 的稳定语义；两种实现都满足现有 ORDER_DELETE 文字的一部分，却产生相反的外部契约和数据保留结果。该决定还影响 migration、最终事务、恢复 tombstone、Evidence 和隐私/保留策略，不能留给实现偶然决定。
- Recommendation: 在 P0-1 事实源同步前冻结 order create idempotency 历史策略。明确 ORDER_DELETE 对 v1/v2 `idempotency_keys` 的查询、保留/删除/墓碑规则，相同 scope/key 在删除后的重放或冲突结果，以及这些行是否像退款历史一样阻断删除；将该规则加入最终单事务、crash/recovery fixture、snapshot 兼容测试和恢复手册。
- Owner: 阶段五协议 owner / storage reviewer。
- Disposition: open。

### AUD-0003-F002 - P0 工作包依赖表与固定关键路径互相矛盾

- 受影响计划：`PLN-0005`。
- Severity: medium。
- Evidence: 第 3 节 P0 tracked 表中 `WP-Lock` 的输入是“现有 applock revision”，未依赖 `WP-Facts`；`WP-Files` 和 `WP-Runtime` 依赖 `WP-Lock`；`WP-Release` 只写“其他原型输出”，没有列出确切前置。第 8 节却把 P0 关键路径固定为 `WP-Facts -> WP-Lock -> WP-Files/WP-Runtime -> WP-Release`，并称其他包按输入并行。checklist `P0-21` 要求记录并执行这些阻塞项，但没有指定以表格还是第 8 节为准。
- Impact: owner 无法判断 WP-Lock 是否可与 WP-Facts 并行，也无法机械验证 WP-Release 需要等待哪些工作包；这会改变 P0 elapsed 估算、并行边界、输入 revision 和 No-Go 判定，并使 requirements matrix 对依赖顺序产生两个合法解释。
- Recommendation: 选择一个依赖 DAG 作为单一事实源。若 WP-Facts 确实阻塞锁协议，更新表格 `WP-Lock` 输入；若不阻塞，修改固定关键路径。把 `WP-Release` 的全部前置工作包和允许提前开始的子工作写清，并让 `P0-21`/requirements validator 检查同一 DAG。
- Owner: 阶段五协议 owner / release owner。
- Disposition: open。

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过；验证 37 个 Markdown 文件的 frontmatter、相对链接、计划/审计治理规则和 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；合法治理 fixture 被接受，缺失链接、孤立 plan 和不完整 closed audit 被拒绝。
- `go test ./... -count=1`：通过。
- `go vet ./...`：通过。
- `go test -race ./... -count=1`：通过。
- `go test -tags phase5_unknown ./internal/applock -count=1`：通过，确认 Go 会接受额外未知 tag；本轮未将其列为 finding，因为计划可通过“未选择任一已知 capability 时编译失败”的包级约束实现所需安全目标，但 P0-19 的测试名称应避免把该机制误述为枚举任意 tag。
- `go doc os.Root` / `go doc os.OpenRoot`：通过，当前 Go 1.26 toolchain 提供计划依赖的 API，且文档与计划“不等价于 no-follow/普通文件/文件系统边界保证”的威胁模型一致。
- `git diff --check`：通过。关闭时 HEAD 仍为 `912ce322bcde15fffcef913b81c7d50088e11853`。

## 未执行项与剩余风险

- 未运行 phase 5 目标 package、发布 tag、migration、文件系统、真实进程 crash、ENOSPC、Windows/Linux 双平台和部署 profile 验证：对应代码、tag、fixture、artifact 与支持环境尚未交付，不能把计划中的未来要求写成已满足。
- 未访问远端 GitHub Actions retention policy、artifact URL 或远端 CI：P0-23 已正确把 180 天策略验证设为 No-Go 前置；当前无法证明组织策略或未来 Evidence 可用性。
- 当前实现没有附件功能，本轮通过的是阶段四基线，不证明计划中的安全、恢复或性能目标已实现。
- `AUD-0002` 是同一对象、同一 baseline 的并发开放审计。其后续结论可能重复或取代本轮证据，关闭时必须显式关联，避免两个记录形成无解释的相反意见。
- 两项 finding 都属于计划缺陷；本轮未发现需要在该记录中扩大为全仓系统性问题的证据。如需仓库级保证，应另行执行 `$backend-full-audit` / `backend-full-audit`。

## 关闭结论

本轮发现 1 项 high、1 项 medium 计划缺陷；没有跨多个活跃计划的冲突，因为当前只有 `PLN-0005` 一份活跃计划。当前代码基线 test/vet/race 与文档验证均通过，但 `PLN-0005` 在 ORDER_DELETE 幂等历史和 P0 依赖 DAG 冻结前不应进入对应实现或把 P0 排期视为可执行。两项 finding 均保持 open disposition；本记录按 `audit-only` 关闭，未修改 plan/checklist 或产品实现。后续整改与复核应创建新的 follow-up audit，不改写本 closed 记录。