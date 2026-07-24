---
id: GOAL-003-modular-ioc-foundation
doc: decision
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-24
updated: 2026-07-24
version: 0.2.0
---

# 决策记录 · GOAL-003

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。原则全文见 Root D-008。

## D-001 · 包布局与端口约定

**日期**：2026-07-24  
**状态**：`accepted`  
**关闭** I-001

**决定**：

1. **Composition root**：`cmd/server` 调用可选 `internal/app`（`NewApp(cfg) (http.Handler, cleanup, error)`）完成组装；禁止在 handler/service 包内构造具体 repository。
2. **按能力分包**（可随 GOAL-002 扩展）：
   - `internal/port` — 跨界出站/入站接口
   - `internal/service` — 应用服务（可再按 bc 子目录）
   - `internal/repository/sqlite` — SQLite 实现
   - `internal/handler` — HTTP
   - `internal/config` — 配置（含 `SQLitePath` / 未来 `DBDriver`）
3. **依赖方向**：handler → service → port ← repository/sqlite。
4. **IoC**：构造函数注入；`internal/app` 内手写 wire-up。
5. **垂直切片**：见 **D-004 MetaStore**。

## D-002 · 范围边界

**决定**：本目标不实现订单/钱包/通知完整 CRUD，不实现 JWT 全链路与 page schema 全集。GOAL-002 在骨架验收后叠加。

## D-003 · 下游交接 = GOAL-002 I-009 充分条件（响应 A-001 F-001）

**日期**：2026-07-24  
**状态**：`accepted`

**决定**：

1. 本目标**验收通过的充分条件之一**是：可填写 [handover-to-goal-002.md](attachments/handover-to-goal-002.md) 的 **H1～H7**。
2. 声明：**H1～H7 全部勾选且有证据** ⇒ 构成关闭 GOAL-002 **I-009 → verified** 的充分条件（GOAL-002 侧同步勾选与 meta 更新）。
3. BC 扩展步骤以交接清单 §2 与 D-001 为准；module-map 扩展点与之对齐。

**为什么**：与 GOAL-002 A-001 F-001 / 本目标 A-001 F-001 成对关闭「供给契约」缺口。

## D-004 · 垂直切片端口 = MetaStore（响应 A-001 F-004）

**日期**：2026-07-24  
**状态**：`accepted`  
**关闭** I-004

**决定**：骨架垂直切片采用 **`MetaStore` 接口**（例如 `Get(ctx, key) (string, error)` / `Set(...)` / `Ping(ctx) error`），用于 ready 探测与「可换存储」证明；**不**采用模糊的 ExampleRepository 双名并存。

SQLite 实现读写简单 `meta` 表；fake 实现内存 map。

## D-005 · 有界空 BC 目录约定（响应 A-001 F-005 recommended）

**日期**：2026-07-24  
**状态**：`accepted`

**决定**：本目标成功标准**包含**（有界）：在文档或仓库中约定 GOAL-002 将使用的 BC 扩展路径，**允许**创建空目录占位（`.gitkeep`），**禁止**在本目标实现完整 CRUD/JWT/page schema。降低「002 立刻重做分包」风险，且不吞并 MVP 业务。
