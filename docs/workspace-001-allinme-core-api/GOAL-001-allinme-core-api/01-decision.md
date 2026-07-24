---
id: GOAL-001-allinme-core-api
doc: decision
status: active
parent: null
created: 2026-07-23
updated: 2026-07-24
version: 0.3.0
---

# 决策记录 · GOAL-001

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。

- **I-001** 策略已 `decided`（A 协议演进）。
- **I-006** 新协议制品 **`verified`**（2.4.1）；Root / R1 协议门禁已关闭。
- I-002 / I-003 已恢复主动收集（仍为 required，阻断各自实施/方案冻结门禁）。

## D-001 · 总目的与交付边界

**决定**：

- 本项目是**可复用核心 API 基座** + **Schema-UI 后端实现方**（page 生产、业务 API、鉴权等）。
- **仅后端**交付；Admin 前端 / Renderer 在其他仓库。
- 复用策略：**模板/骨架复用为主**，可复用能力**逐步**抽到 `pkg/`（轻量 C），不追求一开始就是完整框架库。

**为什么**：

- 协议仓库章程明确：不提供业务后端；本仓补齐「后端生产方」角色。
- 前端与 Renderer 分仓可避免协议/渲染/业务后端三者耦合，便于多项目复用同一后端基座。
- 过早抽大而全库会拖慢 MVP；先在骨架内跑通再沉淀 `pkg/` 更稳。

**未选方案**：

- **本仓同时做 Renderer/Admin 壳**：扩大范围，偏离「核心 API」焦点。
- **仅做成不可 fork 的一次性 demo**：无法支撑「后续项目复用」。
- **一开始做成完整 Go framework 库**：设计成本高，信息不足时易空转。

## D-002 · Schema-UI 协议版本钉死（已 superseded）

**状态**：`superseded`（2026-07-24 · 由 **D-006** 取代）

**原决定（立项时）**：固定消费协议制品 **`2.0.0`**（`meta.protocolVersion: "2.0"`）。

**修订状态（2026-07-23 · D-005）**：`2.0.0` 曾作对照基线；实施待新协议。

**终态**：实施钉死见 **D-006（2.4.1）**。原则不变：只消费不可变制品；不在本仓扩展协议语义。

## D-003 · 第一阶段以 MVP Demo Admin 推进

**决定**：

路线图 R1 对应 `GOAL-002-mvp-demo-admin`：demo 完整 Admin（真实登录/权限；schema 菜单路由；订单/钱包/通知；仪表盘；行内+批量）。验收：page schema 覆盖 Admin 全部入口。批量走协议演进（见 D-005 / D-006）。

**为什么**：用户确认的最小可验证方向；P-001 先路线图再分子目标。

**未选方案**：

- **先只做 ping/health 无业务域**：无法验证 Admin 协议闭环。
- **同时铺开生产级多租户/计费等**：超出 demo 范围。

## D-004 · 工作区与代码布局（默认确认）

**决定**：

- 治理真相源：**显式工作区** `docs/workspace-001-allinme-core-api/`（`workspace.md` + `goal-tree.md` + 平铺 `GOAL-*`）。
- 代码布局：维持现有 **仓库根 Go 布局**（`cmd/`、`internal/`、`pkg/`）。

**为什么**：对齐 `docs/architecture/workspace-protocol.md` 与 directory-layout；代码布局与当前 Go 骨架一致。

**修订（2026-07-24）**：初版误写「仓库根隐式单工作区」；已迁至上述 canonical 范围（见 02-execution 当日记录）。

## D-005 · 协议优先演进；本仓暂停（已 superseded 暂停段）

**状态**：策略 A 与「在协议仓演进」仍有效；**本仓 `blocked` 暂停**由 **D-007** 解除（2026-07-24）。

**原决定（2026-07-23 · 用户确认）**：

1. 批量动作交付策略采用 **A：推动 `schema-ui-docs` 协议演进并发布新版本**。
2. 当时 `GOAL-001` 与 `GOAL-002` 均 `blocked`。
3. 协议工作在 **`schema-ui-docs` 仓库**进行；本仓不伪造协议字段/语义。
4. 恢复条件：新协议可固定引用 + 能力覆盖 → 关闭 I-006/I-008 → 钉死版本 → 用户确认恢复。

**为什么**：互操作批量必须进协议。

## D-006 · 钉死 Schema-UI 协议制品 2.4.1

**日期**：2026-07-24  
**状态**：`accepted`  
**关联**：关闭 I-006；取代 D-002 的实施钉死

**决定**：

固定消费不可变协议制品：

| 字段 | 值 |
|------|-----|
| 制品名 | `schema-ui-protocol` |
| artifactVersion | **2.4.1** |
| meta.protocolVersion | **`"2.4"`** |
| Git tag | **`v2.4.1`** |
| Release | https://github.com/magicvr/schema-ui-docs/releases/tag/v2.4.1 |
| 制品文件 | `schema-ui-protocol-2.4.1.tar.gz` |
| **artifact SHA-256** | `c027fa6c5b4bcb379a2fc90f6447f0e8df0729df5657fcb5d6a382d9ee3fbb18` |
| contentDigest | `sha256:d6852ee6ff12a19b00b4acf0b51e221457fe14691a53ff5ed828876877766efe` |

**能力覆盖结论（相对 MVP / 原 2.0.0 缺口）**：

| 能力 | 2.4.1 | capability / 锚点 |
|------|-------|-------------------|
| 行内动作 | 支持 | `actions.row.request` |
| 批量（当前页多选 + batch request） | 支持（自 2.2） | `table.selection`、`actions.batch.request` · ADR-0022 |
| 页面工具栏 / 新建入口 | 支持（自 2.1） | `actions.page.trigger` · ADR-0020 |
| 列表→编辑/详情导航 | 支持（自 2.1） | `actions.row.navigate` · ADR-0021 |
| 编辑加载回填 | 支持（自 2.1） | `form.record.load` · ADR-0021 |
| 只读详情 | 支持（自 2.4） | `record.view.load` / `recordView` · ADR-0024 |
| 容器权限继承 | 支持（可选，自 2.3） | `permissions.inheritance` · ADR-0023 |

**明确不纳入本仓 MVP 的协议后续项（残余 / 非目标，非协议缺失伪装）**：跨页全选、批量部分成功、行内单元格编辑、导入导出向导、异步任务、树表；**上传**按 GOAL-002 非目标不做。升级协议须**单独决策**，禁止静默跟进 HEAD。

**为什么**：

- 满足 D-005 恢复条件：批量已在核心协议；CRUD 生命周期主路径已有 capability。
- 用户 2026-07-24 确认钉死 **2.4.1**（非仅 2.2 最低批量版本），以覆盖详情/权限等后续可用能力。
- PATCH 2.4.1 相对 2.4.0 固化发布身份与 informative 卫生；机器契约与 2.4 线一致。

**未选方案**：

- **继续钉 2.0.0**：与策略 A / 批量需求冲突。
- **只钉 2.2.0**：批量够用，但丢失 2.3/2.4 权限继承与 recordView。
- **钉 HEAD / 无 SHA**：不可复现。

## D-007 · 解除 blocked 并恢复 R1 推进

**日期**：2026-07-24  
**状态**：`accepted`  
**关联**：I-006 verified；GOAL-002 I-008 verified；用户确认

**决定**：

1. `GOAL-001` 与 `GOAL-002`：`status` **`blocked` → `active`**。
2. 路线图：R0.5 **完成**；R0 边界钉死完成；R1 **进行中**。
3. 恢复 GOAL-002 对 I-002～I-005 的主动信息收集与方案推进；**仍不得**在 I-002～I-005 等 required 门禁未关闭时假装方案冻结完毕或大规模实施写路径。
4. 实施与 page schema 生产必须以 **D-006 钉死的 2.4.1 制品**为准。

**为什么**：D-005 恢复条件已齐；用户指令「OK 钉死 2.4.1 并解除 blocked」。

**未选方案**：只记录证据仍保持 blocked（会继续空转）。
