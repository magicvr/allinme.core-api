---
status: open
audit_id: AUD-0010
auditor: Claude Code
audit_type: governance
scope: repository:allinme.core-api
subject: goal-drift
baseline: git:bbff5f9b727918d85ba1999e524d596fc2d4608a; worktree:clean
started_at: 2026-07-16T21:19:45+08:00
completed_at: pending
last_updated: 2026-07-16
related_audits: AUD-0001
related_remediations: none
related_plans: PLN-0005, PLN-0006
---

# 项目目标漂移专项审计

## 目的与范围

检查 `allinme.core-api` 当前实现、文档、提交重心与治理体系，是否仍对齐以下原始目标：

1. 作为可运行、可演示的订单运营 Demo API；
2. 为通用 Admin 前台提供真实后台交互场景；
3. 为后续 API 项目提供结构参考与可复用逻辑。

本次审计**不**替代附件阶段计划审计，也**不**直接修改产品代码。审计结论只记录证据、影响与建议处置；纠偏实施由 `PLN-0006` 承载。

范围：

- 产品定位文档：`README.md`、`docs/00-overview.md`、`docs/06-implementation-roadmap.md`、`docs/decisions/0001-stateful-local-demo-runtime.md`；
- 运行时产品面：`cmd/`、`internal/`、当前 HTTP 路由与阶段五/六缺口；
- 治理资产：`docs/plans/`、`docs/audits/`、`docs/remediations/`、`docs/implementations/`、`docs/tools/`、`.agents/skills/`、`.github/prompts/`；
- 近 50 次提交与 2026-07-14 之后的提交重心。

## 基线与方法

- 基线：`main@bbff5f9b727918d85ba1999e524d596fc2d4608a`，审计开始时工作树干净。
- 方法：
  1. 只读重建宣称目标与路线图完成态；
  2. 盘点 `internal/` 实际模块与 HTTP 路由面；
  3. 对比产品文档与治理文档体量；
  4. 统计近 50 次提交前缀与 2026-07-14 之后提交主题；
  5. 抽样阅读 `PLN-0005`、`AUD-0001`–`AUD-0009`、验证器与工作流入口。

可复现的提交统计命令：

```sh
git rev-list --count HEAD
git log -50 --pretty=%s
git log --since=2026-07-14T00:00:00+08:00 --pretty=%h %ad %s --date=short
```

近 50 次提交粗分结果（按 subject 前缀）：

| 类别 | 数量 | 判定口径 |
|---|---:|---|
| 产品相关 | 5 | `feat|fix(auth|order|store|http|files|page|pages)` |
| 治理相关 | 34 | `docs(audit|audits|remediation|remediations)`、`feat|refactor|fix(governance)`、`fix(ci)`、`ci:` |
| 其他 | 11 | merge、普通 docs、未归入上两类 |

2026-07-14 之后提交主题几乎全部为 `docs(audits)`、`docs(audit)`、`docs(remediation)`、`feat(governance)`、`refactor(governance)` 与 CI 文档自测，未见附件或页面产品实现提交。

## 历史关系

- `AUD-0001` 已解决“计划与审计混目录、归档可覆盖、命名未强制”等问题，属于合理文档工程化。
- `AUD-0002`–`AUD-0009` 与 `REM-0001`–`REM-0006` 主要围绕 `PLN-0005` 计划契约、校验器与工作流闭环；多条 finding 已 `retired-by-scope`。
- 本审计不否定阶段一至四的产品交付，也不否定轻量文档纪律。结论聚焦：**产品目标文案仍在，近期交付注意力已明显转向治理元系统**。

## Findings

### AUD-0010-F001 - 近期提交重心已从产品交付转向治理元系统

- Severity: high
- Evidence: 全仓约 125 次提交；近 50 次中治理相关约 34、产品相关约 5。2026-07-14 之后主线为审计/整改/治理/CI 文档自测（如 `4f28d58 feat(governance)`、`f87920b refactor(governance)`、`bbff5f9 Merge ... simplify-audit-governance`），同期无 `internal/files`、`internal/pages` 或附件 HTTP 交付。`docs/implementations/records/` 为空，而 `docs/audits/records/` 已有 AUD-0001 至 AUD-0009。
- Impact: 仓库注意力与变更历史已更像“AI 审计操作系统宿主”，而非“继续交付订单运营 demo / Admin 后台支撑 / 可复用 API 骨架”。后续贡献者与自动化容易把治理完善误认为主交付。
- Recommendation: 冻结新增 AUD 类型、skill/prompt 工作流和治理拓扑门禁扩张；将主线切回产品里程碑，并以 `PLN-0006` 记录纠偏范围、预算与停止条件。
- Owner: 后端团队
- Disposition: open

### AUD-0010-F002 - 阶段五计划复杂度显著超过 Demo 边界，且产品代码仍未启动

- Severity: high
- Evidence: `docs/06-implementation-roadmap.md` L68–77 将阶段五定义为上传、绑定、鉴权下载的附件生命周期。`PLN-0005` 仍 `active`，但从 L52 起冻结 40+ 项，覆盖 build tag/capability binary、跨平台 crash harness、ENOSPC、Task Scheduler/systemd、Evidence schema、180 天 artifact 保留、requirements-to-test matrix 等。当前 `internal/` 无 `files`/`pages` 模块；附件与页面 API 仍为 target（`docs/03-http-api.md` 未实现声明与 `docs/03-http-api-target.md`）。
- Impact: 下一阶段在实现前即被生产级发布/恢复/证据供应链规格阻塞，最短可演示闭环（upload → bind → download）无法进入代码。Demo ADR（`docs/decisions/0001-stateful-local-demo-runtime.md`）选择零外部依赖本地运行时的意图被计划复杂度抵消。
- Recommendation: 不覆盖改写 closed 审计历史。新建 `PLN-0006` 定义纠偏与阶段五裁剪原则；将 `PLN-0005` 标记为需切片或 supersede 的过重计划，先交付最小附件闭环，其余生产级恢复与证据供应链降为后续可选。
- Owner: 后端团队
- Disposition: open

### AUD-0010-F003 - 治理文档与工作流体量已超过产品说明，形成自我指涉维护税

- Severity: medium
- Evidence: 产品说明（`docs/00`–`06`、ADR、scenarios、CHANGELOG）约 1k 行量级；治理资产（plans/audits/remediations/implementations/tools、`.agents/skills`、`.github/prompts`）约 4k+ 行量级。`docs/audits/README.md` L39–49 定义 9 套 prompt/skill 工作流；`docs/tools/validate-audit-workflows.ps1` 强制工作流拓扑一致。多条 AUD/REM 的对象是契约解析器、active audits 与 WP-Facts 输出，而非业务 endpoint；`retired-by-scope` 说明部分 finding 只针对后来删除的自然语言门禁。
- Impact: 单功能交付前固定成本被抬高；新人或后续项目若照搬模板会先复制流程工厂。治理规则开始审计/整改自身，收益对 Demo/Admin/复用三项目标递减。
- Recommendation: 保留轻量纪律（current/target、ADR、验证矩阵、基础 frontmatter/链接检查、go test/vet/race）。将 9 工作流闭环、终态账本强制与 audit-workflow 拓扑校验降为可选或外置；禁止无产品阻塞时新增治理规则。
- Owner: 后端团队
- Disposition: open

### AUD-0010-F004 - 对 Admin 前台与后续项目复用目标的完成定义缺失，导致“工程完成”被误读为“目标完成”

- Severity: medium
- Evidence: `README.md` L7–9 与 `docs/00-overview.md` L14–34 正确定位为 Schema-UI 后端消费者与订单运营 demo 宿主；阶段一至四已实现 auth/order/refunds/dashboard，可支撑登录、列表、状态机、审批与看板。但仓库缺少将“通用 Admin 前台支撑”与“后续 API 复用”写成可验收完成定义的显式宪章：无 OpenAPI 产物、无用户管理 HTTP、无页面下发、附件未实现，完整验收仍要求附件+页面+前端联调（`docs/06-implementation-roadmap.md` L91–99）。可复用运行时骨架存在，但治理默认随仓复制，模板边界未切分。
- Impact: 贡献者可能继续优化流程与垂直深度，却不补齐 Admin 联调与模板可迁移性；也可能误判当前仓已是通用 admin 后端或完整脚手架。
- Recommendation: 在 README/overview 增加项目宪章：主目标/次目标/派生目标/非目标，以及完成指标（端到端 Admin 场景数、current API 覆盖、第二个项目实际复用项）。明确 runtime 可复用、order 为示例域、governance 默认不复制。
- Owner: 后端团队
- Disposition: open

## 验证结果

- 文档宣称目标：与 Demo API + Schema-UI 宿主一致；未宣称本仓是治理平台或通用 BaaS。
- 产品完成态：阶段一至四实现与路线图一致；附件与页面未实现；`internal/files`、`internal/pages` 不存在。
- 提交重心：近 50 次治理相关约 34、产品相关约 5；2026-07-14 后无附件/页面产品提交。
- 治理链：AUD-0001 合理；AUD-0002–0009 主要服务计划/校验器闭环；implementation records 空。
- 本次未运行 `go test` / `go vet` / race：审计不修改 Go 源码，产品回归不在本记录验证范围。
- 本次未运行 `docs/tools/validate.ps1`：将在写入本记录与 `PLN-0006` 后执行，作为文档创建后的结构门禁，不作为本审计对象正确性证明。

## 未执行项与剩余风险

- 未对全部历史 commit diff 做逐文件人工复核；提交统计依赖 subject 前缀，可能低估少量“docs 中含产品澄清”的混合提交。
- 未测量精确 LOC（使用量级对比）；若后续需要精确账本，应在 clean checkout 上按路径重算。
- 未启动 API 与 Admin 前台做运行时联调；“可支撑 Admin 的部分场景”来自路由与领域实现审查，不是端到端 UI 证据。
- 剩余风险：若不冻结治理扩张并裁剪阶段五，仓库可能在零附件代码情况下继续增殖 AUD/REM/validator，进一步拉开与原始目标的距离。

## 关闭结论

本审计保持 `open`。四个 finding 均成立且未整改。纠偏实施计划为 `PLN-0006`；整改与复审不得改写本记录，完成后应通过新的 follow-up AUD 或关闭前更新 disposition（若本记录仍 open 且尚未终态）。关闭条件：

1. F001：主线恢复产品里程碑，治理扩张已冻结并有证据；
2. F002：阶段五已裁剪为最小闭环，或被同目标更轻计划明确替代，且附件 MVP 进入实现；
3. F003：治理最小集已写入事实源，非最小治理从必过路径移除或外置；
4. F004：项目宪章与完成指标已写入 README/overview，并被后续交付引用。
