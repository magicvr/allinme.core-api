---
id: GOAL-001-allinme-core-api
doc: decision
status: active
parent: null
created: 2026-07-23
updated: 2026-07-24
version: 0.4.0
---

# 决策记录 · GOAL-001

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。

- **I-001** 策略已 `decided`（A 协议演进）。
- **I-006** 新协议制品 **`verified`**（2.4.1）。
- **I-002 / I-003** **`decided`**（权威在 GOAL-002 D-007 / D-008）。
- **I-007** 模块化/IoC 原则 **`decided`**（本文件 D-008；落地 GOAL-003）。

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

## D-008 · 模块化与 IoC 基础设计原则

**日期**：2026-07-24  
**状态**：`accepted`  
**关联**：关闭 Root I-007；约束全部子目标实施；落地见 GOAL-003

**决定**：从设计与编码开始，本仓采用 **IoC（构造注入）+ 接口边界** 的模块化结构，目标为高内聚、低耦合，**替换模块实现时不修改调用方业务代码**。

### 模块划分原则（P-M1～P-M8）

| # | 原则 | 要求 |
|---|------|------|
| **P-M1** | Composition Root | 仅 `cmd/server`（及可选极薄 `internal/app` / `internal/di`）负责组装与 `New`；业务包不互相构造具体实现类型 |
| **P-M2** | 依赖倒置 | 应用/领域层依赖 **接口**；基础设施（SQLite、HTTP 细节、时钟等）实现接口 |
| **P-M3** | 接口隔离 | 接口按用例切分（如读写分离），避免上帝接口 |
| **P-M4** | 高内聚分包 | 按业务能力/限界上下文分包（auth、order、wallet、notification、schemaui 等），而非无限膨胀的单一 `service` 大包 |
| **P-M5** | 稳定依赖方向 | `handler → service → port ← repository`；禁止基础设施依赖入站适配器；禁止跨域 service 直连对方内部未导出细节 |
| **P-M6** | 可替换实现 | 持久化、密码哈希、ID/时钟等可测边界一律接口；测试可用 fake / memory double |
| **P-M7** | IoC 方式（MVP） | **手动构造注入**；接线膨胀后再评估 `google/wire` 等代码生成，**默认不上**重型运行时 IoC 容器 |
| **P-M8** | 协议边界不变 | 模块化不改变 Schema-UI 权威；只消费钉死 2.4.1，不在本仓发明协议语义 |

### 推荐包轮廓（GOAL-003 细化并可微调）

```text
cmd/server                 composition root
internal/config
internal/domain/<bc>       领域模型（可薄）
internal/port 或 */port    入站/出站接口
internal/service/<bc>      应用服务（只依赖接口）
internal/repository/sqlite 出站适配器（默认可换）
internal/handler           HTTP 入站
internal/auth
internal/schemaui          page schema 生产
pkg/*                      仅真正跨项目可复用内核
```

**为什么**：

- 用户明确要求从设计起 IoC、接口协作、可换实现。
- 与「可复用核心 API 基座 / 模板复用」一致；SQLite 默认可换库也依赖同一倒置边界。
- 手动注入在小中型 Go 服务中清晰、少魔法，利于模板复制。

**未选方案**：

- **无接口、包内直接依赖 sqlite 驱动**：换库必改业务层。
- **全局 service locator / 包级 var 单例乱取**：隐藏依赖，难测难换。
- **默认上重量级 DI 容器**：与 MVP 与模板简洁性不符。

## D-009 · 设立 GOAL-003 作为 R0.8 骨架前置

**日期**：2026-07-24  
**状态**：`accepted`  
**关联**：D-008；GOAL-002 实施顺序

**决定**：

1. 新建子目标 **`GOAL-003-modular-ioc-foundation`**（parent = Root），路线图阶段 **R0.8**。
2. GOAL-003 交付可验收的模块目录、端口接口约定、composition root、SQLite 适配器骨架与文档化模块图；**不**在本目标内完成三域完整业务。
3. GOAL-002 **大规模业务与 page schema 实施**以 GOAL-003 达到其成功标准（或用户书面放行的有界并行）为前提；方案冻结本身不阻塞，**编码顺序**优先骨架。

**为什么**：模块化单独可验收，避免与三域业务进度混在同一目标里难审计。

**未选方案**：仅写原则不落骨架目标；或把骨架完全塞进 GOAL-002 无独立门禁。
