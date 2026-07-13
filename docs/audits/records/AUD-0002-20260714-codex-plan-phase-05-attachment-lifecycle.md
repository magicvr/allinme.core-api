---
status: closed
audit_id: AUD-0002
auditor: codex
audit_type: targeted
scope: plan:PLN-0005
subject: phase-05-attachment-lifecycle
baseline: git:912ce322bcde15fffcef913b81c7d50088e11853; worktree:clean
started_at: 2026-07-14T03:37:54+08:00
completed_at: 2026-07-14T03:47:11+08:00
last_updated: 2026-07-14
related_audits: AUD-0001, AUD-0003
supersedes: none
related_plans: PLN-0005
---

# PLN-0005 阶段五附件生命周期计划审计

## 目的与范围

审计 `PLN-0005` 计划及其 checklist 的正确性、完整性、可执行性，以及与路线图、事实源和当前实现基线的兼容性。本记录只覆盖选中计划，不代表全仓质量审计。

## 基线与方法

- 固定基线：`main@912ce322bcde15fffcef913b81c7d50088e11853`；审计开始时工作树 clean。
- 对象：`docs/plans/PLN-0005-phase-05-attachment-lifecycle.md` 及同号 checklist；两者 frontmatter、文件名、主题、状态和 `docs/plans/README.md` 索引一致。
- 模式：`audit-only`，未修改计划、checklist 或产品实现。
- 方法：读取路线图、当前/目标 HTTP API、领域模型、架构、验证矩阵、Schema-UI 接入文档、相关 ADR、场景、阶段四前置计划、当前 migration/订单幂等/applock/app/admin/HTTP/CI 实现，并核对计划十项必审内容。
- 审计期间出现并发用户改动 `AUD-0003`；该文件不属于固定 baseline，未被本审计修改。

## 历史关系

- `AUD-0001` 的文档治理 finding 在当前 baseline 上仍为 resolved：计划与审计编号、目录、frontmatter、配对和 closed 记录规则均由 validator 强制。
- `AUD-0003` 在本审计开始后以相同代码 baseline 并发创建，关闭本记录时仍为 open 且没有 finding 结论；本记录不覆盖或取代它，后续应在其关闭时比较结论。

## Findings

### AUD-0002-F001 - P0 完成门禁依赖 M1A 才会实现的可执行能力

- Affected plan: `PLN-0005`
- Severity: high
- Category: 计划缺陷 / 工作分解
- Evidence: plan `§3` 要求全部 P0 阻塞工作包完成前保持 No-Go，并在 `§9.1` 要求 P0 完成后才进入 M1A；但 checklist `P0-18` 已要求可运行的 `verify-attachments --full`、规模 fixture 和 wall-clock 结果，`P0-19` 已要求 capability 包、v7/5A 数据与真实 binary smoke，`P0-20` 已要求统一 startup coordinator 和真实 `cmd/api` 进程证据。相同能力又分别在 `M1A-1/M1A-7/M1A-8` 中作为实现交付。当前代码仅有 v6 migration、单一 applock 和现有 app/admin 生命周期，不能在禁止 M1A 开工的同时产出这些 v7/5A 运行证据。
- Impact: P0 无法按自身定义完成，M1A 也无法合法开始；执行者只能违反阶段门禁，或把未实现的设计/模拟证据误记为真实 binary Evidence。
- Recommendation: 把 P0 限定为接口、状态机、DDL 草案、fixture 规格和最小可抛弃原型；将 v7 migration、feature binary、startup coordinator、verify 命令、规模实测和真实进程 smoke 移入 M1A/后续门禁。若必须先做 executable spike，应把它定义为独立前置实现工作包，并明确其代码可进入基线及后续复用条件。
- Owner: 后端团队 / 阶段五协议 owner
- Disposition: open

### AUD-0002-F002 - 发布矩阵没有能够运行 5B 数据和功能的 capability

- Affected plan: `PLN-0005`
- Severity: high
- Category: 计划缺陷 / 发布与回退
- Evidence: plan `§3.1`、`§4.1` 和 checklist `P0-19` 只定义 `phase5_v6_gate`、`phase5_v7_schema_only`、`phase5_v7_5a` 三个发布 tag；`v7-5a-feature` 明确在出现 5B group/kind 时 fail-closed。plan `§8-§9.3` 与 checklist `M1B/M3B/M4/5B-2/5B-4` 又要求实现、启动、smoke 和最终部署 5B 的 group、`REMOVING`、`ORDER_DELETE`、`ORPHAN_FINAL` 数据，但 `5B-3` 仍只验证“三个发布 tag”。
- Impact: 完成 M1B/M3B 后没有任何被发布矩阵允许启动的 API/admin artifact；若继续使用 `phase5_v7_5a`，会违反其 fail-closed capability，若用 `phase5_test` 或无 tag 发布，则违反 provenance 和部署门禁。
- Recommendation: 增加独立 `phase5_v7_5b`/`v7-5b-feature` capability 与 api/admin artifact，更新互斥 tag、入口/数据阶段矩阵、CI、manifest、负向 smoke、回退链和 5B Evidence；或重构 capability 模型，使最终 feature artifact 明确声明并验证其支持的 5A/5B 数据集合，但不能继续把 5A-only artifact当作 5B 发布物。
- Owner: 后端团队 / release owner
- Disposition: open

### AUD-0002-F003 - ORDER_DELETE 缺少完整业务和订单创建幂等生命周期决策

- Affected plan: `PLN-0005`
- Severity: high
- Category: 计划缺陷 / 决策完整性与数据
- Evidence: plan `§3.26`、`§4.3` 和 checklist `M3B-6` 把 `ORDER_DELETE` 定义为会实际删除订单聚合行的 service/repository 用例，但只冻结“无退款及退款幂等历史”条件，没有冻结允许删除的订单状态、调用者/授权边界和 version 冲突顺序。当前 `internal/store/migrations/0004_idempotency.sql` 的订单创建 `idempotency_keys.order_id` 没有订单外键；plan `§6.1` 又要求 v1/v2 重放不查询当前订单，而 ORDER_DELETE 没有说明保留该记录后是否允许重放已删除订单，或删除记录后如何避免同 key 创建第二个订单。
- Impact: 实现可以删除任意无退款历史的订单，包括非 DRAFT/已履约订单；删除后同一 create key 可能返回一个已不存在订单的冻结 `201`，或因清理幂等记录而破坏“同 key 不重复创建”的契约。不同实现者会作出不兼容选择。
- Recommendation: 在 P0-1 事实源同步前决定该能力到底是“附件清理原语”还是“完整订单删除用例”。若是完整删除用例，必须冻结允许状态、actor/role、version、错误优先级，以及订单 create idempotency snapshot 的保留、重放和 tombstone 语义，并给出跨删除前后重试 fixture；若只是未来删除流程的内部文件步骤，则不得在本阶段直接删除订单聚合行。
- Owner: 后端团队 / domain+order owner
- Disposition: open

### AUD-0002-F004 - P0 事实源同步遗漏与两段事务直接冲突的架构文档

- Affected plan: `PLN-0005`
- Severity: medium
- Category: 计划缺陷 / 事实源一致性
- Evidence: `docs/01-architecture.md:58` 当前要求“跨订单、退款、附件元数据的修改在单一事务中完成”。plan `§6.2` 明确采用“准备事务 → 文件隔离 → 最终订单事务”，并声明两段事务之间不存在持续 SQLite writer fence。checklist `P0-1` 的开工前事实源列表只包含领域模型、目标 API、路线图和验证矩阵；架构文档直到 `R2` 实现完成后才同步。
- Impact: M1A/M3B 实现阶段同时存在相反的权威约束，架构审查无法判断两段事务是批准的例外还是计划偏离；若到 R2 才处理，代码可能先于架构决策落地。
- Recommendation: 将 `docs/01-architecture.md` 加入 P0-1/WP-Facts，在任何相关实现前把该规则收窄为“单次 SQLite 原子阶段”或记录新的 ADR，明确 SQLite 与文件系统不能共同事务、准备态可见性、最终事务原子边界及恢复责任。
- Owner: 后端团队 / architecture owner
- Disposition: open

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，验证 37 个 Markdown 文件及 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过。
- `go test ./... -count=1`：通过，wall 22.6s。
- `go vet ./...`：通过。
- `go test -race ./... -count=1`：通过，wall 112.4s。
- `go doc os.Root`：确认 Go 1.26 当前标准库提供 `os.Root`，且其文档明确说明会跟随根内 symlink、不能阻止文件系统边界或设备文件；plan 要求额外平台适配层与原型是必要且与基线相容的。
- 首次将两个文档验证脚本并行执行时，`validate.ps1` 读到了 `validate.tests.ps1` 的临时 fixture 并失败；按规范命令串行重跑后均通过。该现象属于工具并行隔离风险，不作为 `PLN-0005` finding；若需评估其全仓影响，建议执行 `$backend-full-audit` / `backend-full-audit`。

## 未执行项与剩余风险

- 未执行远端 GitHub Actions、artifact retention/profile 校验、真实 v6/v7/5A/5B binary smoke、Windows reparse/rename/share-mode、Linux 监督器/调度、ENOSPC、真实 crash harness 或成套恢复；这些能力在当前 baseline 尚未实现，不能写成已满足。
- 当前全仓 test/vet/race 通过只证明 v6 基线健康，不证明计划中的 v7 schema、文件恢复、capability artifact 或部署门禁可行。
- checklist 当前全部未勾选，没有发现缺少 Evidence 的虚假完成项；但四个 open finding 修复前，P0/M1A 和最终 5B 路径不具备可执行闭环。
- 单计划审计未检查所有仓库模块的系统性质量。文档工具并发隔离问题及任何其他全仓风险应由 `$backend-full-audit` 单独评估。

## 关闭结论

本轮发现 4 个新问题：high 3、medium 1。没有跨计划冲突，因为当前只有 `PLN-0005` 为 active；但计划内部存在 P0/M1A 循环门禁和缺失 5B 发布 capability，均阻断按文档顺序交付。四个 finding 均保持 open，后续修改 plan/checklist/事实源后应创建新的 follow-up audit 复核。本审计完成并关闭不表示这些 finding 已整改，也不表示计划完成。
