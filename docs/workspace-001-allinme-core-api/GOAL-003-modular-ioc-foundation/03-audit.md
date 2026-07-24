---
id: GOAL-003-modular-ioc-foundation
doc: audit
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-24
updated: 2026-07-24
version: 0.3.0
---

# 审计 · GOAL-003

## 信息就绪核对

| 项 | 状态 | 说明 |
|----|------|------|
| I-001 包布局 | decided | D-001 |
| I-002 wire | decided | 不引入 |
| I-003 SQLite 驱动库 | open · non-blocking | **S3 当日**关闭 |
| I-004 MetaStore 切片 | decided | D-004 |

## 审计意见台账

## A-001 · 规划合理性交叉审计（R0.8 / vs GOAL-002 前置）（2026-07-24）

- **source**：independent
- **auditor**：GitHub Copilot · Grok 4.5
- **类型**：design-plan
- **scope**：GOAL-003 当前规划（S1～S4、成功标准、D-001/D-002、对 GOAL-002 I-009 的供给）；不审代码实施完成度、不关门
- **verdict**：**conditional**
- **完整意见**：本节即全文（未另附 attachments）

### 范围与区间

- 工作区：`workspace-001-allinme-core-api`；Root / canonical 与 GOAL-002 A-001 相同；`shared_materials_catalog=none`。
- 只读依据：本目标五件套、[module-map-draft.md](attachments/module-map-draft.md)、Root D-008/D-009、GOAL-002 I-009/D-013、`cmd/server/main.go`（handler 直连、无 `internal/app` / port）、`internal/` 目录现状、principles P-001～P-005。
- **未**修改 status/progress/方案/goal-tree。

### 成果（有证据）

| 主张 | 证据 |
|------|------|
| 独立 R0.8 骨架目标，不吞并三域业务 | [00-meta 概述](00-meta.md)；D-002；Root D-009 |
| 包布局与依赖方向已决策 | D-001；module-map-draft |
| 成功标准可验收且对齐 P-M1～P-M8 | [00-meta 成功标准](00-meta.md)；Root D-008 |
| 不引入重型 DI；wire 明确非 MVP | I-002 decided；P-M7 |
| progress 0% 诚实（仅立项+布局决策） | [02-execution.md](02-execution.md)；仓库尚无 port/app 代码 |
| 作为 GOAL-002 实施前置写在树与下游 | goal-tree；00-meta 下游；GOAL-002 I-009 |

### 对照成功标准 / 规划质量（本 scope）

| 维度 | 评价 |
|------|------|
| P-001 | 目标可直接执行；S1～S4 粒度合适，**无需**再拆大量子目标。 |
| 范围裁剪 | 「垂直切片证明可换存储 + composition root」正确；避免与 GOAL-002 混目标。 |
| 与现状差距 | `main` 仍 `handler.Register` 直连，与目标态一致为「待 S2/S3 改造」，规划方向对。 |
| 对下游供给 | 成功标准偏「骨架证明」；**对 GOAL-002 M2 的可操作交接说明不足**（F-001）。 |

### Findings

| ID | 级别 | 严重度 | 说明 | 证据 / 关联 |
|----|------|--------|------|-------------|
| **F-001** | **required** | med | **对 GOAL-002 I-009 的供给契约未写清**：本目标成功标准自洽，但未声明「验收通过 = 关闭 GOAL-002 I-009 的充分条件」及下游如何按 D-001 扩展 auth/order 等 BC（仅 module-map 一句扩展点）。建议在 00-meta/01-decision 增加「下游交接」小节，与 GOAL-002 I-009 清单对齐。 | 成功标准；[module-map-draft](attachments/module-map-draft.md)；GOAL-002 I-009 |
| **F-002** | recommended | low | **S1 状态滞后**：I-001/D-001 已 decided，module-map-draft 已存在，但路线图 S1 仍「进行中」。建议 `/govern` 将 S1 标完成或写明唯一未完成物（例如 map 升为 active）。 | [00-meta 路线图](00-meta.md)；D-001 |
| **F-003** | recommended | low | **I-003（SQLite 驱动库）open · non-blocking** 可接受；规划应承诺 **S3 接线当日**写入 execution 关闭，避免拖到 S4 验收争论。 | [00-meta I-003](00-meta.md) |
| **F-004** | recommended | low | 成功标准「至少一条出站端口」示例（HealthStore / ExampleRepository）二选一未钉死；不影响合理性，S2 开工前选定可减少空转。 | 成功标准第 3 条 |
| **F-005** | recommended | med | **风险（非否决）**：过薄骨架可能导致 GOAL-002 立刻重做分包。若用户更在意一次成型，可**有界**把「空 BC 目录约定 + README 新增模块步骤」写入本目标成功标准（仍禁止做完整 CRUD）。属产品取舍，需用户/编排器决定，非审计改范围。 | D-002；Root P-M4 |

### 必改项汇总

1. **F-001**：补充与 GOAL-002 I-009 对齐的**下游交接说明**（充分条件列表 + BC 扩展步骤指针）；与 GOAL-002 A-001 F-001 成对关闭。

### 与既有意见的异同

- 此前无 A-00N。  
- 与 GOAL-002 **A-001** 同向：R0.8→R1 串行合理；共同必改主题为 **交接契约**。本侧不要求扩大为完整 MVP。

### 结论 + 建议给编排器/用户的下一步

**结论**：GOAL-003 **规划合理且必要**——独立可验收骨架、手动 IoC、范围不吞业务，符合 Root D-008/D-009 与 P-001。因对 GOAL-002 门禁供给表述不足，verdict = **conditional**。

**建议 `/govern`**：

1. 响应本 A-001 + GOAL-002 A-001；先闭环 F-001（交接清单），再推进 **S2 目录与端口落地**。  
2. 可选：收尾 S1（F-002）、S2 前选定垂直切片端口形态（F-004）。  
3. P-004：询问是否需要对本目标 design-plan 做 self 审计后再推进 S2。  
4. **不要**在本目标未达成功标准（或用户书面有界放行）时启动 GOAL-002 M2 业务编码。

### 声明

本意见不修改 status/progress；响应由 `/govern` 处理。

---

## A-002 · 编排响应 A-001（2026-07-24）

- **source**：self（编排响应，**非** independent）
- **auditor**：/govern · Grok
- **类型**：response
- **scope**：响应 A-001 findings；规划补强；不审代码
- **verdict**：**pass**（required 已关闭；代码实施未开始属预期）

### 关闭证据表

| Finding | 处置 | 状态 | 证据 |
|---------|------|------|------|
| **F-001** | 交接清单 + D-003；GOAL-002 D-014 成对 | **closed** | [handover-to-goal-002.md](attachments/handover-to-goal-002.md)；[01-decision D-003](01-decision.md)；GOAL-002 D-014 |
| F-002 | S1 → 完成 | **closed** | [00-meta 路线图](00-meta.md) |
| F-003 | I-003 最晚 **S3 当日** | **closed**（承诺） | [00-meta I-003](00-meta.md) |
| F-004 | 钉死 **MetaStore** | **closed** | D-004；I-004 decided |
| F-005 | 有界空 BC + README 步骤入成功标准 | **closed** | D-005；00-meta 成功标准 |

### 仍开放项

| 项 | 说明 |
|----|------|
| 代码 S2～S4 | 未开始 |
| I-003 驱动库 | S3 当日 verified |
| design-plan self | 未做（P-004 待用户） |

### 结论

可进入 **S2 目录与 MetaStore 端口落地**；完成 H1～H7 前不得放行 GOAL-002 M2。

## 备注

- 2026-07-24：A-001 independent；A-002 govern 响应。
