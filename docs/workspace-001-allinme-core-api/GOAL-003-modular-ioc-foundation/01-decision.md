---
id: GOAL-003-modular-ioc-foundation
doc: decision
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-24
updated: 2026-07-24
version: 0.1.0
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
   - `internal/port` — 跨界出站/入站接口（或 `internal/<bc>/port` 若接口仅单 BC 使用；跨 BC 共享的放 `internal/port`）
   - `internal/service` — 应用服务（可再按 bc 子目录）
   - `internal/repository/sqlite` — SQLite 实现
   - `internal/handler` — HTTP
   - `internal/config` — 配置（含 `SQLitePath` / 未来 `DBDriver`）
3. **依赖方向**：handler → service → port ← repository/sqlite。
4. **IoC**：构造函数注入；`internal/app` 内手写 wire-up。
5. **本目标最小垂直切片**：保留 ping/health；增加「可替换存储」证明（例如 ready 检查 DB ping 经接口，或极简 key-value/meta 表经接口读写）。

**为什么**：对齐现有目录；满足可换库与可测；避免过早微前端式过度分包。

**未选**：

- 全部接口塞进一个 `interfaces` 上帝包无界膨胀。
- 默认引入 fx/dig 等运行时容器。

## D-002 · 范围边界

**决定**：本目标不实现订单/钱包/通知完整 CRUD，不实现 JWT 全链路与 page schema 全集。GOAL-002 在骨架验收后叠加。

**未选**：本目标直接做完整个 MVP（范围失控）。
