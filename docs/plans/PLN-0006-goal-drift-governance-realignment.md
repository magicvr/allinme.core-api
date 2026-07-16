---
status: archived
plan_id: PLN-0006
owner: 后端团队
created: 2026-07-16
last_updated: 2026-07-16
applies_to: goal-drift governance realignment and demo delivery recovery
---

# 目标漂移纠偏与交付重心恢复计划

配套审计：[AUD-0010](../audits/records/AUD-0010-20260716-claude-governance-goal-drift.md)。配套 checklist：`PLN-0006-goal-drift-governance-realignment-checklist.md`。

本计划响应 `AUD-0010`：在**不推倒阶段一至四产品实现**的前提下，冻结治理自我增殖，恢复 Demo API / Admin 支撑 / 可复用骨架三项目标，并为阶段五建立可执行的最小闭环边界。

## 目标与完成边界

### 目标

1. **重新锚定项目宪章**：主目标是可运行订单运营 Demo API；次目标是支撑通用 Admin 前台真实交互；派生目标是从已验证实现提炼可复用结构。
2. **冻结治理扩张**：在附件最小闭环完成前，不新增审计工作流类型、治理拓扑门禁与无产品消费者的 validator 规则。
3. **裁剪下一阶段交付**：把附件阶段从生产级发布/证据供应链规格，收缩为可演示的 upload → bind → download（+ 删除/解绑）闭环。
4. **定义完成指标**：用端到端场景、current API 覆盖和实际复用边界衡量进度，而不是 AUD/REM 数量。

### 完成边界

本计划完成时，必须同时满足：

- README 或 overview 中有可引用的项目宪章与非目标；
- 治理最小集与“禁止扩张清单”已写入事实源；
- `PLN-0005` 的过重 P0 规格已被明确降级、切片或替代策略写清（本计划不直接实现附件代码）；
- 下一产品实施入口明确：附件 MVP 的范围/非目标/验收场景可被后续实现计划或切片后的阶段五计划直接消费；
- checklist 中已完成项均有 Evidence；未执行项有原因与风险。

**本计划不包含：** 附件/页面业务代码实现、OpenAPI 全量生成器、用户管理 API 实现、拆分独立 governance 仓库的完整迁移。

## 范围与非目标

### 范围

| 工作包 | 内容 |
|---|---|
| WP-Charter | 写入项目主/次/派生目标、非目标、完成指标与资产分层 |
| WP-Freeze | 冻结治理扩张；定义保留的最小门禁与可选治理 |
| WP-Phase5-Slice | 定义附件 MVP 与明确推迟项；给出对 `PLN-0005` 的处置策略 |
| WP-Metrics | 定义防漂移检查问题与投入预算比例 |
| WP-Handoff | 输出下一实现计划入口与 AUD-0010 关闭前置条件映射 |

### 非目标

- 不重写或删除 closed AUD/REM 历史记录；
- 不在本计划内实现 `internal/files` 或附件 HTTP；
- 不要求立刻删除全部 skill/prompt；可先降级为非必过；
- 不把本仓改造成通用 BaaS 或多租户平台；
- 不新增第 10 套审计工作流或新的 until-ready 编排；
- 不把“治理更严格”作为本计划成功标准。

## 事实源与依赖

| 事实源 | 角色 |
|---|---|
| [AUD-0010](../audits/records/AUD-0010-20260716-claude-governance-goal-drift.md) | 漂移证据与 finding |
| [README.md](../../README.md) | 对外定位入口，宪章落点之一 |
| [00-overview.md](../00-overview.md) | 项目职责与当前/目标态 |
| [06-implementation-roadmap.md](../06-implementation-roadmap.md) | 阶段顺序与完成证据 |
| [03-http-api.md](../03-http-api.md) / [03-http-api-target.md](../03-http-api-target.md) | 当前 vs 目标 API |
| [04-validation.md](../04-validation.md) | 验证门禁与能力矩阵 |
| [0001-stateful-local-demo-runtime.md](../decisions/0001-stateful-local-demo-runtime.md) | Demo 运行时边界 |
| [PLN-0005](./PLN-0005-phase-05-attachment-lifecycle.md) | 现有阶段五计划（过重，需切片） |

依赖：

- 阶段一至四已实现且保持可用；
- 不依赖新的治理校验器；
- 附件实现可在本计划归档后由新计划或切片后的阶段五计划启动。

## 冻结决策

以下决策在本计划 active 期间冻结；变更必须先改本计划与对应事实源。

1. **目标优先级**：Demo 闭环 > Admin 可联调场景 > 模板抽象 > 流程优化。
2. **仓库身份**：本仓是 Demo/Reference API，不是 Agent Governance Framework。若继续建设治理操作系统，必须拆到独立仓库或明确标为可选子树。
3. **治理最小集（保留）**：
   - `go test ./...`、`go vet ./...`、需要时 race；
   - current/target 状态词与单一事实源；
   - ADR + CHANGELOG；
   - 验证矩阵 enabled 表；
   - 基础 frontmatter / 相对链接检查；
   - 短 plan + checklist（可选，按功能需要）。
4. **治理扩张冻结清单（附件 MVP 完成前禁止）**：
   - 新增 AUD 工作流类型 / skill / prompt 配对；
   - 新增仅服务治理拓扑的 validator 规则；
   - 新增 production-grade Evidence schema 作为 demo P0；
   - 对校验器/编排器再开审计闭环，除非 CI 已红且无更轻修复。
5. **阶段五 MVP 范围**：
   - 单文件上传（类型与大小限制）；
   - 附件元数据持久化；
   - 绑定到订单；
   - 鉴权下载；
   - 删除或解绑的最小语义；
   - reset/seed 可恢复；
   - HTTP 集成测试 + 一条 Admin 可演示场景。
6. **阶段五明确推迟**：
   - capability binary 矩阵与多 build tag 发布链；
   - Windows/Linux crash harness 作为 P0；
   - Task Scheduler/systemd 部署 profile 作为 P0；
   - 180 天 artifact 保留与完整 Evidence 供应链；
   - requirements-to-test machine matrix 作为实现前置；
   - 跨平台掉电安全与完整 orphan 接管编排。
7. **对 PLN-0005 的处置（已选定）**：
   - `PLN-0005` 与 checklist 原地 `archived`，保留全部历史规格、未勾选项和 closed 审计链接；
   - 轻量 [`PLN-0007`](./PLN-0007-phase-05-attachment-mvp.md) 是唯一阶段五实现入口；
   - 不从 `PLN-0005` 复制 capability binary、crash harness、调度 profile 或 Evidence 供应链作为 MVP 前置。
8. **投入预算（正常阶段）**：约 70% 产品/场景，20% 测试与契约，10% 文档与治理。连续迭代治理 >20% 必须书面说明解除了哪个产品阻塞。
9. **抽象规则**：先完成业务实现；只有第二个真实项目出现相同需求时才稳定抽取，禁止为想象中的后续项目预建框架。
10. **本计划成功不看**：新增 AUD 数量、新增 validator 行数、闭环轮次。

## 工作包与负责人

| 工作包 | 负责人 | 输入 | 出口 |
|---|---|---|---|
| WP-Charter | 后端团队 | AUD-0010 F004；README/overview | 宪章与非目标合入事实源 |
| WP-Freeze | 后端团队 | AUD-0010 F001/F003；audits/tools/validation | 最小门禁与冻结清单合入事实源；CI/文档说明一致 |
| WP-Phase5-Slice | 后端团队 | AUD-0010 F002；roadmap；PLN-0005 | MVP 场景/API/非目标成文；PLN-0005 处置决定记录 |
| WP-Metrics | 后端团队 | AUD-0010 关闭条件 | 防漂移检查 5 问 + 预算规则写入 overview 或 validation |
| WP-Handoff | 后端团队 | 上述出口 | 下一实现入口与 AUD-0010 finding 映射表 |

每个工作包完成后在 checklist 记录：日期、revision、改动路径、验证命令与结果。

## 交接与审计映射

| AUD-0010 finding | 本计划处置 | 后续关闭证据 |
|---|---|---|
| F001 近期重心治理化 | WP-Freeze 将工作流拓扑校验移出默认产品 CI，并冻结新增治理入口 | `PLN-0007` 产生附件产品实现提交，主线恢复产品里程碑 |
| F002 阶段五过重 | `PLN-0005` 原地归档；`PLN-0007` 定义附件 MVP 范围与停止条件 | `PLN-0007` 进入实现并交付 upload → bind → download 场景 |
| F003 治理维护税 | `04-validation.md` 定义最小门禁；九套工作流改为按风险选用 | 非最小治理不再阻塞默认产品 PR，且附件 MVP 前不再扩张 |
| F004 Admin/复用完成定义缺失 | `00-overview.md` 成为项目宪章、防漂移和资产分层事实源 | 后续路线与交付引用宪章指标 |

下一产品实现入口为 [`PLN-0007`](./PLN-0007-phase-05-attachment-mvp.md)。`PLN-0006` checklist 完成只表示治理纠偏已经落地，不表示附件已经实现；`AUD-0010` 保持 open，直到上述产品证据满足其关闭条件。

## 风险、回退与停止条件

| 风险 | 缓解 | 回退 |
|---|---|---|
| 纠偏计划本身变成新的重治理工程 | 本计划禁止新增工作流；文档变更保持短小 | 停止扩展，只保留宪章 + 冻结清单 |
| 直接大改 PLN-0005 破坏历史审计链接 | 优先新建轻量实现计划并归档旧计划 | 仅增加警告横幅，不改历史审计 |
| 删除治理文件导致 CI 红 | 先降级必过项，再归档；每步跑 validate | 恢复被删路径或临时 keep 脚本 |
| 宪章与现有 demo 定位冲突 | 宪章必须兼容 README 现有 Schema-UI/demo 表述 | 只追加“目标优先级/非目标”，不改已实现能力描述 |
| 无 Admin 前台仓时无法证明场景 | 先用 API smoke + 场景文档作为中间证据 | 在 checklist 标记联调未执行及风险 |

停止条件：

- 出现新增审计工作流或 validator“为了完成纠偏计划” → 停止，回到 WP-Freeze；
- 附件 MVP 范围重新膨胀到 crash/调度/Evidence 供应链 → 停止实现准备，重做切片；
- 阶段一至四回归失败 → 优先修产品回归，本计划文档工作让路。

## 验收与 Evidence

### 验收标准

1. **宪章可见**：任意维护者从 README 或 overview 能读到主/次/派生目标与非目标。
2. **冻结可执行**：文档写明附件 MVP 完成前禁止哪些治理扩张。
3. **阶段五可开工**：存在不超过最短路径的 MVP 范围说明（场景、API、非目标、测试最低集）。
4. **历史可追溯**：AUD-0010 仍可访问；closed 记录未被改写。
5. **结构门禁通过**：`docs/tools/validate.ps1` 通过；若改动了 workflow 文件才需要 `validate-audit-workflows.ps1`。
6. **未假完成产品**：不将本计划勾选为“附件已实现”。

### Evidence 最低要求

- 文档 diff 路径列表；
- 关键命令与退出码；
- 对 `PLN-0005` 处置的明确语句（归档 / 警告 / 替代计划 ID）；
- AUD-0010 各 finding 的建议处置状态（仍 open 可接受，只要映射到后续动作）。

### 建议后续（本计划外）

1. 按 [`PLN-0007`](./PLN-0007-phase-05-attachment-mvp.md) 实现附件 MVP；
2. OpenAPI 最小导出；
3. 页面下发；
4. Admin 前台端到端联调；
5. 第二个真实项目出现相同需求后再抽取 runtime kit。

配套 checklist：`PLN-0006-goal-drift-governance-realignment-checklist.md`。
