---
status: archived
plan_id: PLN-0006
owner: 后端团队
created: 2026-07-16
last_updated: 2026-07-16
applies_to: goal-drift governance realignment and demo delivery recovery
---

# 目标漂移纠偏与交付重心恢复 Checklist

配套计划：`PLN-0006-goal-drift-governance-realignment.md`。配套审计：[AUD-0010](../audits/records/AUD-0010-20260716-claude-governance-goal-drift.md)。

每个完成项必须紧随实际日期、revision、命令、结果和 Evidence 位置；未执行项必须写明原因和风险。

## 0. 基线与范围

- [x] 基线、范围与依赖已确认。
  - Date: 2026-07-16
  - Revision: worktree based on `bbff5f9b727918d85ba1999e524d596fc2d4608a`
  - Evidence: `AUD-0010` baseline；近 50 提交统计 product=5 / governance=34 / other=11
  - Scope: 宪章、治理冻结、阶段五切片、防漂移指标；不含附件业务实现

## 1. WP-Charter — 项目宪章

- [x] 在 `README.md` 和 `docs/00-overview.md` 写入主目标、次目标、派生目标。
- [x] 明确非目标：非通用 BaaS、非生产分布式订单系统、非 AI 治理框架（默认）。
- [x] 写明目标优先级：Demo 闭环 > Admin 场景 > 模板抽象 > 流程优化。
- [x] 写明资产分层：runtime 可复用；order 为示例域；protocol 为生态可选；governance 默认不随模板复制。
  - Date: 2026-07-16
  - Paths: `README.md`、`docs/00-overview.md`、`docs/README.md`
  - Evidence: `00-overview.md#2-项目宪章与防漂移规则` 是唯一正文；README 只保留摘要与链接；docs README 将其登记为单一事实源

## 2. WP-Freeze — 治理冻结与最小门禁

- [x] 在 `docs/04-validation.md` 记录治理最小集。
- [x] 写明附件 MVP 完成前的治理扩张冻结清单。
- [x] 确认不新增 skill/prompt/工作流类型作为本计划交付物。
- [x] 确认 closed AUD/REM 未被改写。
- [x] 决定 audit-workflow 拓扑校验的去留并落实。
  - Date: 2026-07-16
  - Decision: `validate-audit-workflows.ps1` 移出默认产品 CI，保留为修改 prompt/skill 时的可选维护检查
  - Paths: `.github/workflows/ci.yml`、`docs/04-validation.md`、`docs/tools/README.md`、`docs/audits/README.md`、`docs/plans/README.md`
  - Evidence: `validate.ps1` 仍保护 frontmatter、链接、编号、索引和终态历史；未新增或删除任何 skill/prompt；closed AUD/REM 文件无 diff

## 3. WP-Phase5-Slice — 阶段五最小闭环

- [x] 成文附件 MVP 用户场景：登录 → 打开订单 → 上传 → 绑定 → 刷新仍在 → 鉴权下载 → 无权限拒绝 → 删除未绑定。
- [x] 成文 MVP API/数据边界：单文件、类型/大小限制、元数据、创建绑定、下载、未绑定删除、cleanup、reset/seed。
- [x] 成文明确推迟项：crash harness、调度 profile、Evidence 供应链、capability binary 矩阵等。
- [x] 对 `PLN-0005` 做出处置决定：原地归档并由 `PLN-0007` 替代。
- [x] 确认切片后的 P0 不要求先建设生产级发布工程。
  - Date: 2026-07-16
  - Paths: `docs/plans/PLN-0007-phase-05-attachment-mvp.md`、配套 checklist、`PLN-0005` plan/checklist、`docs/plans/README.md`、`docs/06-implementation-roadmap.md`、`docs/03-http-api-target.md`、`docs/05-domain-model.md`、`docs/04-validation.md`
  - Evidence: `PLN-0005` 保留原路径和历史正文，状态为 archived；所有未勾选项保持；`PLN-0007` 是唯一阶段五实现入口
  - Notes: 本项完成不表示附件代码已实现，附件验证矩阵仍为 `enabled: no`

## 4. WP-Metrics — 防漂移机制

- [x] 写入完成指标：端到端 Admin 场景数、current API 覆盖、可复用边界。
- [x] 写入投入预算：约 70/20/10，治理连续超 20% 需说明产品阻塞。
- [x] 写入里程碑轻量漂移五问。
- [x] 写入“第二个真实项目出现相同需求时再抽象”规则。
  - Date: 2026-07-16
  - Paths: `docs/00-overview.md#2-项目宪章与防漂移规则`
  - Evidence: 完成指标、预算、五问和抽象时机只在 overview 维护，README/validation 仅链接

## 5. WP-Handoff — 交接与审计映射

- [x] 建立 AUD-0010-F001…F004 到本计划工作包/后续计划的映射表。
- [x] 明确下一实现入口为 `PLN-0007`。
- [x] 明确 AUD-0010 保持 open 直到关闭条件满足；本 checklist 完成 ≠ 附件已实现。
  - Date: 2026-07-16
  - Paths: `PLN-0006-goal-drift-governance-realignment.md#交接与审计映射`、`AUD-0010`
  - Evidence: F001/F003 映射治理冻结；F002 映射 PLN-0007；F004 映射项目宪章；AUD-0010 `related_plans` 已包含 PLN-0007

## 6. 文档与验证门禁

- [x] `docs/audits/README.md` 已索引 `AUD-0010` 恰好一次。
- [x] `docs/plans/README.md` 已索引 `PLN-0006`/`PLN-0007`，并将 `PLN-0005` 移入稳定路径归档索引。
- [x] `docs/tools/validate.ps1` 通过。
  - Command: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`
  - Result: 通过；Validated 63 Markdown files
- [x] `docs/tools/validate.tests.ps1` 通过。
  - Command: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`
  - Result: 通过；合法结构、断链、plan 配对、audit 必需字段与唯一索引 fixture 符合预期
- [x] `docs/tools/validate-audit-workflows.ps1` 作为可选维护检查仍可通过。
  - Command: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-audit-workflows.ps1`
  - Result: 通过；脚本保留但不再由默认产品 CI 调用
- [x] 未将附件/页面能力的验证矩阵行伪标为 `enabled: yes`。
  - Date: 2026-07-16
  - Notes: 本轮修改了 `docs/04-validation.md` 的附件最低证据口径，但 `enabled` 保持 `no`

## 7. 全量确认与归档前置

- [x] 全量验证、未执行项与剩余风险已记录。
- [x] 完成报告已提交（本 checklist、AUD-0010 验证结果与最终交付摘要）。
- [x] 用户于 2026-07-16 明确要求完成上述治理工作，plan 与 checklist 已同步原地归档，文件未移动。

归档前剩余风险：

- 附件 MVP 尚未实现；`AUD-0010-F001/F002` 的产品交付关闭证据必须由 `PLN-0007` 产生。
- 当前只是将治理工作流拓扑校验移出默认 CI；脚本与 9 套入口仍保留，未来若继续扩张应拆出独立项目，而不是重新升级为默认门禁。

## Evidence 模板

```text
- Date:
- Revision:
- Command:
- Result:
- Paths:
- Notes:
```
