---
id: GOAL-002-mvp-demo-admin
doc: decision
status: active
parent: GOAL-001-allinme-core-api
created: 2026-07-23
updated: 2026-07-24
version: 0.4.0
---

# 决策记录 · GOAL-002

## 信息需求与阶段门禁

权威表见 [00-meta.md](00-meta.md)。

- I-001 / I-008：已关闭。
- **I-002～I-007**：**`decided`**（本文件 D-007～D-012）；方案冻结完成。
- **I-009**：**open** — 阻断 **M2 起业务编码**（待 GOAL-003 或用户放行）。

## D-001 · MVP 范围与验收口径

**决定**：模块范围见 meta；验收以 page schema 覆盖 Admin 全部入口为准；实施协议为 Root 钉死的 **2.4.1**。

## D-002 · 批量动作策略（已裁决）

策略 **A**；2.4.1 以 `table.selection` + `actions.batch.request` 实现；禁止 Host Extension 冒充协议。

## D-003 · 业务域选型

**订单、钱包、通知**；细节见 D-008。

## D-004 · 非目标（本 MVP）

不实现上传；不做生产级多租户/计费/工作流；不做跨页全选、批量部分成功、行内单元格编辑、真邮件发送。

## D-005 · 暂停（已 superseded）

暂停已由 Root D-007 / 本目标 D-006 解除。

## D-006 · 采纳协议钉死 2.4.1 并恢复推进

**accepted**（2026-07-24）。实施对照 2.4.1。

## D-007 · 鉴权与会话（I-002）

**日期**：2026-07-24 · **状态**：`accepted` · **关闭** I-002

**决定**：

| 项 | 选择 |
|----|------|
| 机制 | **JWT Bearer**（`Authorization: Bearer <accessToken>`） |
| Access TTL | **1h** |
| Refresh | **MVP 不做**；过期重新登录 |
| 密码 | **bcrypt** 哈希存储 |
| 种子用户 | 三角色各一：`admin` / `operator` / `viewer`；演示密码统一 **`Demo@1234`**（仅 demo，文档标明） |
| API | `POST /v1/auth/login` → token + user；`GET /v1/auth/me` |
| 保护范围 | 除 health/ready/login 外默认需鉴权 |

**为什么**：跨仓 Renderer 友好；实现简单；满足「真实登录」非 mock。

**未选**：Session Cookie（跨站/CORS 更烦）；OAuth/OIDC（过重）；长期 refresh 体系（可后补）。

## D-008 · 三域最小模型与 API（I-003）

**日期**：2026-07-24 · **状态**：`accepted` · **关闭** I-003

**决定**：字段、状态机与端点清单见附件 [mvp-domain-and-api.md](attachments/mvp-domain-and-api.md)。摘要：

- **订单**：pending→paid|cancelled；paid→refunded；含 batch-delete 与 mark-paid/cancel。
- **钱包**：active|frozen；freeze/unfreeze 与 batch-freeze；无支付网关。
- **通知**：draft→published→archived；channel 枚举；不真发邮件。

**为什么**：覆盖 Admin CRUD + 行内/批量演示，体量可控。

**未选**：复杂库存/结算；真实邮件/短信通道。

## D-009 · RBAC（I-004）

**日期**：2026-07-24 · **状态**：`accepted` · **关闭** I-004

**决定**：

| 角色 | 能力 |
|------|------|
| `admin` | 全部菜单与写操作 |
| `operator` | 三域 CRUD + 行内/批量；无系统级扩展 |
| `viewer` | 只读 list/detail/dashboard |

- **粒度**：菜单 + 页面 + 操作（`view` / `edit` / `delete`）。
- **不做**行级数据归属。
- page schema `permissions.*` 仅用 `$context.user.roles`；**后端 handler 独立校验**。

**未选**：仅 admin 单角色；ABAC/数据行权限。

## D-010 · 协议 2.4.1 → MVP 页面映射（I-005）

**日期**：2026-07-24 · **状态**：`accepted` · **关闭** I-005

**决定**：映射表见 [protocol-capability-mapping.md](attachments/protocol-capability-mapping.md)。三域均提供 list/create/edit/detail + dashboard；登录非 schema 页。

## D-011 · 持久化 SQLite 与可换库（I-006）

**日期**：2026-07-24 · **状态**：`accepted` · **关闭** I-006

**决定**：

1. **默认驱动：SQLite**（可配置路径，如 `data/demo.db`）。
2. 业务仅依赖 **Repository / 出站端口接口**；SQLite 为实现细节。
3. 预留 `DB_DRIVER`（或等价配置）；MVP 只实现 sqlite；后续 postgres 等新增实现，不改 service。
4. 测试可用 memory/fake 实现同一接口。
5. 空库启动 seed。

**为什么**：用户要求默认 SQLite 且可换库；与 Root D-008 一致。

**未选**：纯内存默认；业务层直连 `database/sql` 无抽象。

## D-012 · page schema 生产（I-007）

**日期**：2026-07-24 · **状态**：`accepted` · **关闭** I-007

**决定**：page schema 以 **仓库内嵌 YAML/JSON**（Go `embed` 或构建期装载）按 `pageId` 下发；`GET /v1/admin/pages/{pageId}`；不进业务 DB。菜单单独 JSON/代码配置，按角色过滤。

**未选**：DB 存 schema（MVP 过重）；运行时从协议仓拉 HEAD。

## D-013 · 实施依赖 GOAL-003

**日期**：2026-07-24 · **状态**：`accepted` · **关联** I-009

**决定**：M2 起业务编码须在 GOAL-003 达到成功标准后进行，或经用户书面「有界并行」放行并记入 execution。方案冻结不依赖 GOAL-003 完成。

**为什么**：用户选方案 A（独立骨架目标）。
