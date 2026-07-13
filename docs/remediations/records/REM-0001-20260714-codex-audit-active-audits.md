---
status: completed
remediation_id: REM-0001
implementer: codex
scope: audit:AUD-0002,AUD-0003
source_audits: AUD-0002, AUD-0003
source_findings: AUD-0002-F001, AUD-0002-F002, AUD-0002-F003, AUD-0002-F004, AUD-0003-F001, AUD-0003-F002
baseline: git:f6764be5a2d5ef092589c2cc822b684dd9ff725c; worktree:clean
started_at: 2026-07-14T04:45:41+08:00
completed_at: 2026-07-14T04:53:43+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# PLN-0005 活跃审计整改

## 对象与边界

本记录整改 `AUD-0002` 与 `AUD-0003` 中全部 open findings。两份审计针对同一计划和相同历史代码基线，其中 ORDER_DELETE 幂等生命周期意见重复，合并为同一实现项但保留两条 source finding 映射。本轮只修改 `PLN-0005`、配套 checklist 及为消除架构冲突必须同步的直接事实源；不修改已关闭审计，不实现阶段五产品代码，也不把 finding 标记为 resolved。

## Finding 整改矩阵

| Source finding | Root cause | Planned change | Validation | Result |
|---|---|---|---|---|
| `AUD-0002-F001` | P0 将 M1A 才能产生的 migration、feature binary、startup coordinator、完整 verify 和规模实测当作 P0 完成门禁，形成阶段循环。 | 把 P0 产物限定为接口、草案、fixture、测试骨架和可抛弃 spike；把可执行 migration、真实发布 binary、完整命令和进程/规模实测明确归入 M1A/后续门禁。 | `docs/tools/validate.ps1`；`docs/tools/validate.tests.ps1`；P0/M1A 边界检索；`git diff HEAD --check`。 | completed locally; pending follow-up audit |
| `AUD-0002-F002` | 发布矩阵只有 v6、v7 schema-only 和 5A feature，5A artifact 又必须对 5B 数据 fail-closed。 | 新增独立 `phase5_v7_5b` / `v7-5b-feature` capability、artifact、入口/数据矩阵、CI、负向 smoke、回退链和最终门禁。 | 同上；四发布产物、五 capability tag、5A 拒绝 5B 数据及 5B 最终 gate 检索。 | completed locally; pending follow-up audit |
| `AUD-0002-F003` | ORDER_DELETE 被写成完整聚合删除，但未冻结允许状态、actor/role、version/error 顺序和 create idempotency 删除后语义。 | 将其收窄为仅供受信任 admin maintenance 编排的内部 cleanup 原语；限定 DRAFT、expected version、错误优先级，并冻结 v1/v2 create snapshot 永久保留、删除后同 key 重放首次冻结响应且不得创建第二个订单。 | 同上；plan/checklist/domain/API/roadmap/validation 一致性检索。 | completed locally; pending follow-up audit |
| `AUD-0002-F004` | 架构事实源仍要求订单、退款、附件元数据跨域修改处于单一事务，与 edit/remove 两段 SQLite 事务和文件隔离协议冲突。 | 在 P0-1 前置同步 architecture，把规则收窄为每个 SQLite 原子阶段单事务，并明确文件系统不参与 SQLite 事务、准备态可见性和恢复责任。 | 同上；architecture 与 plan 的准备/最终事务边界检索。 | completed locally; pending follow-up audit |
| `AUD-0003-F001` | 与 `AUD-0002-F003` 相同，特别是订单删除后的 create idempotency snapshot 处置未定义。 | 与 `AUD-0002-F003` 共用内部 DRAFT cleanup 原语和 snapshot 保留/重放契约，并增加跨删除重试 fixture。 | 同上；ORDER_DELETE 与 idempotency snapshot 跨事实源检索。 | completed locally; pending follow-up audit |
| `AUD-0003-F002` | P0 工作包表允许 WP-Lock 独立开始且模糊 WP-Release 前置，但正文关键路径要求 WP-Facts 先于 WP-Lock 并要求 release 等待原型。 | 以 tracked 工作包表为唯一 DAG：WP-Lock 依赖 WP-Facts，WP-Release 明列全部必需前置；P0-21 和 requirements validator 校验同一 DAG。 | 同上；tracked 表、DAG 唯一事实源和 P0-21 检索。 | completed locally; pending follow-up audit |

## 实际变更

- `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md`：将 P0 限定为协议、fixture、测试规格、工具 schema 和可抛弃 spike；真实 v7 migration、release binary、startup/verify 实现、规模 wall-clock 与进程 smoke 移入 M1A/5A/5B/R。
- `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md`：新增 `phase5_v7_5b` / `v7-5b-feature`，形成四发布产物、五个互斥 capability tag、5A→5B 前进链和禁止旧 binary 读取 5B 数据的回退边界。
- `docs/plans/PLN-0005-phase-05-attachment-lifecycle.md` 与 checklist：把 tracked work-package 表设为唯一依赖 DAG，令 WP-Lock 依赖 WP-Facts、WP-Release 等待其余七包批准输出，并要求 validator 拒绝环、未知或漂移依赖。
- `docs/01-architecture.md`：把单事务规则收窄为每个 SQLite 原子阶段，冻结准备事务、文件隔离、最终事务、purge 与恢复责任。
- `docs/05-domain-model.md`、`docs/03-http-api-target.md`、`docs/04-validation.md`、`docs/06-implementation-roadmap.md`：统一 ORDER_DELETE 为无 HTTP surface 的受信任 maintenance 内部原语，仅允许 DRAFT + expected version + 无退款历史，并永久保留 create snapshot 供删除后同 key 重放且禁止第二次创建。
- 实际工作树 revision：`f6764be5a2d5ef092589c2cc822b684dd9ff725c` 上的未提交整改 diff；未修改任何 `status: closed` AUD 正文。

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，验证 41 个 Markdown 文件、frontmatter、相对链接和 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：首次因 Windows 临时 fixture `audits/README.md` 文件句柄冲突失败；确认无残留 fixture 后串行重试通过，合法治理 fixture 被接受且负向 fixture 被拒绝。
- `git diff HEAD --check`：通过；仅输出工作树未来 LF→CRLF 转换警告，无 whitespace error。
- 闭环检索确认：四发布产物/五 capability tag、独立 5B artifact、P0/M1A Evidence 边界、ORDER_DELETE DRAFT/version/idempotency 语义、architecture 两段事务和唯一 dependency DAG 均已在相关文件一致表达。

## 未完成项与剩余风险

- 本轮只修复计划与事实源可执行性，不证明阶段五实现、发布产物或部署环境已经存在。
- 未执行 Go test/vet/race：本轮没有修改 Go 代码，且审计 findings 均为计划/事实源缺陷；文档治理门禁是最小可证伪范围。
- `validate.tests.ps1` 的一次性 fixture 文件句柄冲突仍提示 Windows 上的 validator 隔离存在非本 finding 范围风险；重试通过不等于该工具风险已整改，建议后续专项或全仓审计评估。
- source findings 仍保持 open，只有新的 follow-up audit 可以确认本整改有效。

## Follow-up 交接

整改完成后由 `$backend-follow-up-audit TARGET=REM-0001` 独立复审；本记录不自行给出审计验证结论。
