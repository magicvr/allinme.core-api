---
title: GOAL-003 → GOAL-002 下游交接清单（I-009）
status: active
created: 2026-07-24
updated: 2026-07-24
parent: GOAL-003-modular-ioc-foundation
version: 0.1.0
---

# GOAL-003 → GOAL-002 交接清单

> 关闭 **GOAL-002 I-009** 的充分条件。本清单与 GOAL-003 成功标准对齐；**全部勾选**（或用户书面有界放行，见 §3）后，GOAL-002 方可进入 **M2 业务编码**。

权威引用：GOAL-003 `00-meta` 成功标准；GOAL-002 I-009 / D-013 / D-014；Root D-008。

## 1. 可勾选交接项（充分条件）

| # | 交接项 | 对应 GOAL-003 成功标准 / 证据 | 关闭 I-009 时核验 |
|---|--------|------------------------------|-------------------|
| H1 | **Composition root 唯一组装** | `cmd/server`（及可选 `internal/app`）为唯一 `New` 具体实现处；业务包不互相构造具体 repository | [ ] 代码路径 + 简短说明写入 GOAL-003 execution |
| H2 | **出站端口 + SQLite 实现** | 至少一条 port 接口 + `internal/repository/sqlite` 实现；配置含 SQLite 路径 | [ ] |
| H3 | **可替换证明（测试）** | 至少 1 个测试：service 只依赖接口，用 fake/memory **无 SQLite** 跑通 | [ ] `go test` 路径记入 execution |
| H4 | **进程可启动** | `/healthz`、`/readyz`、`/v1/ping` 仍可用 | [ ] 本地或 CI 记录 |
| H5 | **模块图 active** | [module-map-draft.md](module-map-draft.md) 与仓库目录一致，frontmatter `status: active`（或等价定稿文档） | [ ] |
| H6 | **扩展 BC 指引可读** | README 或 docs 含「如何新增业务模块 / 如何换 Repository 实现」；空 BC 目录约定见 §2 | [ ] |
| H7 | **无重型 DI 容器** | 手动构造注入；未默认引入 fx/dig 等 | [ ] 依赖清单目视 |

**判定**：H1～H7 均勾选 → GOAL-003 可关门（若其成功标准亦满）且 **GOAL-002 I-009 → verified**，证据为：本表勾选记录 + GOAL-003 `status: done`（或 execution 验收节）+ GOAL-002 更新 I-009。

## 2. 下游如何按 D-001 扩展 BC（GOAL-002 M2+）

在不破坏依赖方向的前提下：

```text
1. 在 internal/port（或 internal/<bc>/port）定义用例接口
2. 在 internal/service/<bc> 实现应用服务（只依赖 port）
3. 在 internal/repository/sqlite 增加表与实现
4. 在 internal/handler 增加 HTTP 适配（依赖 service 接口）
5. 仅在 internal/app（composition root）接线 New*
6. 禁止：handler → sqlite；service → sqlite 具体包；跨 BC 直连对方未导出内部类型
```

建议空目录占位（GOAL-003 可有界创建，**无业务逻辑**）：

- `internal/service/auth|order|wallet|notification|schemaui`（可先放 `.gitkeep`）
- 或文档中明确「首次提交某 BC 时再建模，但必须走上述步骤」

## 3. 用户有界并行放行（例外）

若 GOAL-003 未全部完成仍要启动 GOAL-002 部分编码，**书面最低要素**（写入 GOAL-002 `01-decision` 或 `02-execution`）：

| 要素 | 要求 |
|------|------|
| 范围 | 允许编码的包/用例列表（例如「仅 auth 登录，不含三域」） |
| 仍开放的 H# | 列出未勾选交接项 |
| 禁止事项 | 不得绕过 port 直连 sqlite；不得扩大到未列范围 |
| 复审触发 | 例如「GOAL-003 done 当日复审依赖方向」或固定日期 |
| 用户确认 | 明示接受 residual |

无上述书面要素 → **不得**将 I-009 标为 verified，也不得开始 M2 全量业务编码。
