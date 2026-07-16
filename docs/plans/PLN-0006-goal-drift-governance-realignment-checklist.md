---
status: active
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
  - Revision: `bbff5f9b727918d85ba1999e524d596fc2d4608a`
  - Evidence: `AUD-0010` baseline；`git rev-parse HEAD`；近 50 提交统计 product=5 / governance=34 / other=11
  - Scope: 宪章、治理冻结、阶段五切片、防漂移指标；不含附件业务实现

## 1. WP-Charter — 项目宪章

- [ ] 在 `README.md` 和/或 `docs/00-overview.md` 写入主目标、次目标、派生目标。
- [ ] 明确非目标：非通用 BaaS、非生产分布式订单系统、非 AI 治理框架（默认）。
- [ ] 写明目标优先级：Demo 闭环 > Admin 场景 > 模板抽象 > 流程优化。
- [ ] 写明资产分层：runtime 可复用；order 为示例域；protocol 为生态可选；governance 默认不随模板复制。

## 2. WP-Freeze — 治理冻结与最小门禁

- [ ] 在 `docs/04-validation.md` 或 `docs/README.md` 记录治理最小集。
- [ ] 写明附件 MVP 完成前的治理扩张冻结清单。
- [ ] 确认不新增 skill/prompt/工作流类型作为本计划交付物。
- [ ] 确认 closed AUD/REM 未被改写。
- [ ] 决定 audit-workflow 拓扑校验的去留：保留 / 降级 / 移出必过（记录决定与理由）。

## 3. WP-Phase5-Slice — 阶段五最小闭环

- [ ] 成文附件 MVP 用户场景：登录 → 打开订单 → 上传 → 绑定 → 刷新仍在 → 鉴权下载 → 无权限拒绝 → 删除/解绑。
- [ ] 成文 MVP API/数据边界：单文件、类型/大小限制、元数据、绑定、下载、删除/解绑、reset/seed。
- [ ] 成文明确推迟项：crash harness、调度 profile、Evidence 供应链、capability binary 矩阵等。
- [ ] 对 `PLN-0005` 做出处置决定并记录：归档并替代 / 顶部警告并缩 P0 / 新建 `PLN-0007`。
- [ ] 确认切片后的 P0 不要求先建设生产级发布工程。

## 4. WP-Metrics — 防漂移机制

- [ ] 写入完成指标：端到端 Admin 场景数、current API 覆盖、可复用边界。
- [ ] 写入投入预算：约 70/20/10，治理连续超 20% 需说明产品阻塞。
- [ ] 写入里程碑轻量漂移五问（最近提交产品比、路线图是否有代码进展、无消费者抽象、流程是否自服务、新增可演示场景）。
- [ ] 写入“第二次使用后再抽象”规则。

## 5. WP-Handoff — 交接与审计映射

- [ ] 建立 AUD-0010-F001…F004 到本计划工作包/后续计划的映射表。
- [ ] 明确下一实现入口（建议附件 MVP 实现计划 ID 或启动条件）。
- [ ] 明确 AUD-0010 保持 open 直到关闭条件满足；本 checklist 完成 ≠ 附件已实现。

## 6. 文档与验证门禁

- [x] `docs/audits/README.md` 已索引 `AUD-0010` 恰好一次。
  - Date: 2026-07-16
  - Paths: `docs/audits/README.md`
  - Evidence: 索引顶部新增 `AUD-0010` 一条；`validate.ps1` 通过
- [x] `docs/plans/README.md` 已索引 `PLN-0006` plan + checklist。
  - Date: 2026-07-16
  - Paths: `docs/plans/README.md`
  - Evidence: 活跃计划增加 `PLN-0006` 双链接；与 `PLN-0005` 并列
- [x] `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1` 通过。
  - Date: 2026-07-16
  - Command: `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`
  - Result: exit 0；Validated 61 Markdown files
- [x] 若未改 workflow：记录 `validate-audit-workflows.ps1` 未要求；若改了 workflow：已运行并通过。
  - Date: 2026-07-16
  - Notes: 本轮未修改 `.github/prompts` 或 `.agents/skills`，未要求运行 `validate-audit-workflows.ps1`
- [ ] 需要时运行 `docs/tools/validate.tests.ps1` 并记录结果。
- [x] 未将附件/页面能力的验证矩阵行伪标为 `enabled: yes`。
  - Date: 2026-07-16
  - Notes: 本轮未修改 `docs/04-validation.md` 能力矩阵

## 7. 全量确认与归档前置

- [ ] 全量验证、未执行项与剩余风险已记录。
- [ ] 完成报告已提交（已完成项 / 未执行项 / 剩余风险 / 对 AUD-0010 的影响）。
- [ ] 取得用户归档确认后，才将 plan 与 checklist 同步改为 `status: archived`（文件不移动）。

## Evidence 模板（勾选时复制）

```text
- Date:
- Revision:
- Command:
- Result:
- Paths:
- Notes:
```
