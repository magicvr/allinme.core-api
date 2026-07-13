---
status: closed
audit_id: AUD-0004
auditor: codex
audit_type: follow-up
scope: follow-up:REM-0001
subject: rem-0001-active-audits
baseline: git:631eac0781c06db235b118b1cd89c24a4f9ac82e; worktree:clean
started_at: 2026-07-14T04:56:16+08:00
completed_at: 2026-07-14T04:59:16+08:00
last_updated: 2026-07-14
related_audits: AUD-0002, AUD-0003
related_remediations: REM-0001
supersedes: none
related_plans: PLN-0005
---

# REM-0001 阶段五计划整改复审

## 目的与范围

独立复审 `REM-0001` 对 `AUD-0002`、`AUD-0003` 六个 open findings 的整改有效性。复审范围包括源审计证据、整改提交 `631eac0781c06db235b118b1cd89c24a4f9ac82e`、`PLN-0005` 与 checklist、直接事实源、索引流转和文档治理门禁；不把尚未实施的阶段五产品能力当作本次计划缺陷整改证据。

## 基线与方法

- 固定基线：`main@631eac0781c06db235b118b1cd89c24a4f9ac82e`；复审开始时工作树 clean。
- `REM-0001` 记录的实施基线为 `f6764be5a2d5ef092589c2cc822b684dd9ff725c` 上未提交 diff；当前基线已将该 diff 完整提交为 `631eac0`，因此复审以提交内容为准。
- 方法：重新从每条 source finding 建立失败条件，核对实际 commit diff、计划/checklist/架构/领域/API/验证/路线图一致性，运行整改声明的验证并增加反向检索和结构检查。

## 历史关系

- `AUD-0002` 与 `AUD-0003` 是相同历史代码 baseline 上的 legacy v1 计划审计；本记录不修改其 closed 正文。
- `REM-0001` 合并了重复的 ORDER_DELETE 幂等生命周期意见，但仍为全部六条 source finding 保留映射。

## 复核矩阵

| Source finding | Claimed remediation | Code/evidence inspected | Independent test | Verdict |
|---|---|---|---|---|
| `AUD-0002-F001` | P0 仅保留规格/fixture/spike，真实 migration、binary、runtime 和规模证据移到 M1A/后续。 | 检查 plan §3/§8/§9、checklist P0-13..P0-23、M1A-1/7/8/9 与提交 `631eac0` diff。P0-18/19/20 已移出真实 v7/runtime 证据，但 P0-22 仍要求“至少选择并实测一个单机 profile”，P0-23 又要求 P0-22 fixture run；全部 P0 阻塞完成前仍禁止 M1A。 | 反向布尔检查 `P0RequiresLiveProfile=True`；文档 validator。 | `partially-resolved`；见 `AUD-0004-F001` |
| `AUD-0002-F002` | 新增独立 5B capability、artifact、矩阵和回退链。 | 检查发布矩阵、入口矩阵、P0-19、5B-3、R4 和全仓 tag 引用；`phase5_v7_5b`/`v7-5b-feature` 均存在，5A 对 5B data 退出 78，5B artifact、四发布 tag 和禁止回退链已冻结。 | 反向检查 `Has5BTag=True`、`Has5BArtifact=True`、五 tag 集合完整；检索无残留“三个发布 tag”。 | `resolved` |
| `AUD-0002-F003` | ORDER_DELETE 收窄为 DRAFT/version/maintenance 原语并保留 create snapshot。 | 检查 plan §3.26/§4.3、checklist P0-1/M3B-6、domain/API/validation/roadmap 与现有 `idempotency_keys` 无订单 FK 基线。actor boundary、expected version、DRAFT、错误顺序、退款历史、snapshot 保留和删除后读取语义一致。 | 跨事实源布尔检查 `OrderBoundary=True`、`IdempotencyRetention=True`；文档 validator。 | `resolved` |
| `AUD-0002-F004` | 架构单事务规则收窄为每个 SQLite 原子阶段并冻结文件恢复责任。 | `docs/01-architecture.md` 已在实施前提交，明确 SQLite/文件系统不能共同事务以及准备、隔离、最终事务、purge、journal/startup/cleanup 责任；P0-1 将 architecture 加入前置事实源。 | `ArchitectureTwoStage=True`；链接/frontmatter validator。 | `resolved` |
| `AUD-0003-F001` | 删除后同 create key 重放首次 snapshot 且不创建第二订单。 | 检查 domain、plan §3.26/§4.3、checklist P0-1/M3B-6、validation；永久保留 v1/v2 snapshot、按 scope/key 重放、不查询当前订单、禁止第二创建和订单读取 not found 均已冻结。 | 同 `AUD-0002-F003` 的独立跨源检查；检查 migration 0004 无 order FK 与所选语义兼容。 | `resolved` |
| `AUD-0003-F002` | tracked work-package 表成为唯一 DAG，修正 Lock/Release 依赖。 | 表中 WP-Lock 已依赖 WP-Facts，WP-Release 已列其余七包；但 §8 仍称 WP-Facts 先于 WP-Baseline-Evidence，而表中 WP-Baseline-Evidence 输入仍是“本计划修订 revision；留存 profile”，没有 WP-Facts。 | 反向检查 `DagTableBaselineDependsFacts=False`、`DagProseSaysBaselineDependsFacts=True`；文档 validator 未检测该语义漂移。 | `partially-resolved`；见 `AUD-0004-F002` |

## Findings

### AUD-0004-F001 - P0 仍以真实部署 profile 阻塞 M1A 开工

- Maps to: `AUD-0002-F001`
- Severity: high
- Evidence: `PLN-0005` 继续规定全部 P0 阻塞工作包完成前保持 No-Go，5A-I 要求 P0 完成后进入 M1A/后续；checklist `P0-22` 仍要求“至少选择并实测一个单机 profile”，包括监督、cleanup 调度、退出码和恢复证据，`P0-23` 又要求 P0-22 产生 fixture run。真实 cleanup、watchdog、阶段五恢复和 5A binary 分别到 M1A/5A-D 才实现，当前 P0 仍无法在禁止 M1A 开工的同时实测完整 profile。
- Impact: 虽然 migration、capability、startup coordinator 和规模实测已正确后移，执行者仍必须在 P0 伪造阶段五部署 profile evidence，或违反 No-Go 进入 M1A；原循环门禁的根因未完全消除。
- Recommendation: P0-22 只冻结目标 profile、监督器/timer 配置模板、命令/退出码/告警 acceptance 和环境 owner；把真实 profile 实测、cleanup 调度、watchdog/recovery/ENOSPC 证据完全移入 5A-D/5B-4。P0-23 对 P0-22 只接受 `contract-fixture`，不得要求真实 run。
- Owner: 阶段五协议 owner / release owner
- Disposition: partially-resolved

### AUD-0004-F002 - 唯一 dependency DAG 对 Baseline-Evidence 仍自相矛盾

- Maps to: `AUD-0003-F002`
- Severity: medium
- Evidence: plan tracked 表的 `WP-Baseline-Evidence` 输入是“本计划修订 revision；留存 profile”，没有 `WP-Facts`；同一 plan §8 明确说“WP-Facts 先于依赖它的 Schema-Recovery、HTTP-Order、Lock 与 Baseline-Evidence”。P0-21 声称表是唯一 DAG 并要求拒绝正文漂移，但当前 validator 通过了这一矛盾。
- Impact: Baseline-Evidence owner 仍有两种合法执行顺序，requirements validator 无法从表导出与正文一致的图；elapsed、输入 revision 和 P0 No-Go 判定仍可能分叉。
- Recommendation: 选择一个顺序并只写在表中。若 Baseline-Evidence 必须等待事实源修订，将其输入改为 `WP-Facts；本计划修订 revision；留存 profile`；否则删除 §8 对 Baseline-Evidence 的前置声明。为 P0-21 validator 增加该边的正反 fixture。
- Owner: 阶段五协议 owner / Evidence tooling owner
- Disposition: partially-resolved

## 验证结果

- `git show 631eac0`：逐文件检查 REM 实际提交，与 REM 声明范围一致；closed source AUD 正文在 `f6764be..631eac0` 间无 diff。
- 独立结构检查：5B tag/artifact、ORDER_DELETE actor/state/version/snapshot、architecture 两阶段契约均返回 true；P0 live-profile 依赖返回 true，DAG table/prose 对 Baseline-Evidence 分别返回 false/true。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，验证 42 个 Markdown 文件的 frontmatter、相对链接和 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过，合法治理 fixture 被接受，缺失矩阵/索引/链接、孤立计划和不完整 closed audit 被拒绝。
- `git diff HEAD --check`：通过；只有工作树未来 LF→CRLF 转换警告，无 whitespace error。

## 未执行项与剩余风险

- 未执行阶段五 binary、migration、文件系统、crash 或部署测试：相应实现仍不存在，本次只验证计划缺陷整改，不能把未来 Evidence 当作已通过。
- 未执行 Go test/vet/race：整改提交只修改文档治理与计划契约；源 finding 的可证伪条件是计划可执行性和事实源一致性。
- `AUD-0004-F001` 与 `AUD-0004-F002` 需要新的 REM；其余四条 source finding 已在当前 committed baseline 上独立确认 resolved。
- validator 当前不解析 P0 dependency DAG 的语义边，也不识别 P0/M1A 部署证据循环；文档门禁通过不推翻两项新 finding。

## 关闭结论

`REM-0001` 部分通过：六条 source findings 中四条 resolved，两条 partially-resolved。独立 5B capability、ORDER_DELETE/idempotency 生命周期与 architecture 事务边界已消除原根因；P0/M1A 循环仍残留真实 profile 门禁，DAG 仍对 Baseline-Evidence 自相矛盾。因此本 follow-up audit 关闭但进入新的整改队列，`REM-0001` 标记 partial verification，source audits 的活动队列转移到 `AUD-0004`。
