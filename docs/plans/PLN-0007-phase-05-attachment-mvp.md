---
status: active
plan_id: PLN-0007
owner: 后端团队
created: 2026-07-16
last_updated: 2026-07-16
applies_to: implementation roadmap phase 5 attachment MVP
---

# 阶段五：附件 MVP 实施计划

配套路线：[Demo API 实施路线图](../06-implementation-roadmap.md)。配套 checklist：`PLN-0007-phase-05-attachment-mvp-checklist.md`。

本计划替代已归档的 [`PLN-0005`](./PLN-0005-phase-05-attachment-lifecycle.md) 作为阶段五唯一实现入口。`PLN-0005` 保留为历史规格与审计对象，不再作为开工或发布门禁。本计划响应 [`AUD-0010-F002`](../audits/records/AUD-0010-20260716-claude-governance-goal-drift.md) 与 [`PLN-0006`](./PLN-0006-goal-drift-governance-realignment.md) 的阶段五切片决定。

## 目标与完成边界

交付可重复演示、可测试的本地附件闭环：

1. `operator` 或 `admin` 上传单个允许类型文件，取得稳定附件 ID；
2. 创建订单时绑定本人未过期、未被其他订单绑定的附件；
3. 刷新并重新查询订单后仍能看到附件摘要；
4. 已认证且可查看订单的角色通过受保护 API 下载附件；
5. 未认证、不可见或非法状态请求被稳定拒绝，响应不泄露本地路径；
6. 创建者可删除本人尚未绑定的附件；
7. development reset/seed 后可恢复确定的演示基线。

完成本计划只代表附件 API 与订单创建绑定可用，不代表页面 YAML、UploadAction 映射、生产级部署或灾难恢复已完成。

## 范围与非目标

### 范围

- additive SQLite migration 与附件元数据；
- `DATA_DIR` 下受控的本地文件目录；
- `POST /api/v1/attachments` 单文件 multipart 上传；
- `GET /api/v1/attachments/{attachmentId}` 鉴权下载；
- `DELETE /api/v1/attachments/{attachmentId}` 删除本人未绑定附件；
- 订单 create 的 `attachmentIds` 绑定；订单 detail/write 响应的附件摘要；
- PDF、PNG、JPEG 服务端内容探测，单文件上限 `10 MiB`；
- 未绑定附件 24 小时过期与可重复 cleanup 命令；
- development seed/reset、真实 SQLite 和 HTTP 集成测试；
- current/target API、领域、路线图、验证矩阵与 CHANGELOG 同步。

### 非目标

以下项目不得阻塞 MVP 合并：

- 订单 edit 的附件增删、重排与组恢复；
- 订单删除 HTTP 或内部 ORDER_DELETE 清理编排；
- 对象存储、预签名 URL、公开分享、病毒扫描、缩略图、分片或断点续传；
- 页面 YAML、UploadAction 映射、L0-L4 页面校验与页面 readiness；
- capability binary、多 build tag 发布矩阵或 schema gate 产物链；
- Windows/Linux crash harness 作为实现前置；
- systemd、Task Scheduler、常驻调度或部署 profile；
- 180 天 artifact 保留、生产级 Evidence schema 或 requirements-to-test 机器矩阵；
- 主机掉电安全、完整 orphan 接管和生产级灾难恢复。

## 事实源与依赖

| 事实源 | 角色 |
|---|---|
| [项目宪章](../00-overview.md#2-项目宪章与防漂移规则) | 目标优先级、非目标与防漂移规则 |
| [路线图阶段五](../06-implementation-roadmap.md#6-阶段五附件) | 阶段顺序与完整 demo 边界 |
| [目标 HTTP API](../03-http-api-target.md) | 待实现附件 endpoint 草案 |
| [领域模型](../05-domain-model.md) | 附件所有权、绑定、访问和过期语义 |
| [验证规则](../04-validation.md) | 最低测试与能力矩阵 |
| [ADR-0001](../decisions/0001-stateful-local-demo-runtime.md) | SQLite + 本地文件的 Demo 运行时边界 |

依赖阶段一至四已有 auth、order、migration、seed/reset、统一错误、CORS 和集成测试基础。不得以建立新治理工作流作为本计划前置。

## 冻结决策

1. 附件 ID 为服务端生成的稳定不透明 ID；客户端文件名不参与存储路径。
2. 首版只允许 PDF、PNG、JPEG，单文件最大 `10 MiB`；服务端探测内容，不仅信任扩展名或声明 MIME。
3. 文件内容保存在 `DATA_DIR` 下受控目录，SQLite 保存元数据、所有者、状态、过期时间与订单映射；响应不返回绝对路径或公开 URL。
4. 上传先产生本人拥有、24 小时有效的未绑定附件；订单 create 在事务内验证并绑定。
5. 订单 create 的 `attachmentIds` 缺失、`null` 与空数组等价；ID 必须唯一并保持输入顺序；每个订单最多 10 个附件。
6. 含附件的订单创建必须将规范化 `attachmentIds` 纳入幂等摘要；历史无附件请求的重放行为不得被破坏。实现只需稳定、可测试的版本分派，不建设发布产物矩阵。
7. 下载仅允许已绑定附件，并复用订单查看授权；不存在、不可见或非法附件不得泄露本地文件事实。
8. 删除只允许 `operator`/`admin` 删除本人未绑定附件；已绑定附件返回状态冲突。
9. cleanup 只处理稳定过期的未绑定附件和本次操作可识别的失败残留；首版不引入常驻调度和跨进程接管协议。
10. 订单 PATCH 不接收 `attachmentIds`；edit 增删与解绑另开后续计划。
11. 列表可只返回 `attachmentCount`；详情和写入响应返回 `id/fileName/contentType/sizeBytes/sha256/createdAt` 摘要。
12. 若实现需要扩大到 crash、调度、发布或 Evidence 供应链，立即停止并新建独立后续计划，不扩写本计划 P0。

## 工作包与负责人

| 工作包 | 负责人 | 出口 |
|---|---|---|
| WP0 事实源同步 | 后端团队 | 领域、目标 API、路线与验证矩阵仅做实现所需的短 diff |
| WP1 schema + files | 后端团队 | additive migration、repository、本地文件适配和内容校验 |
| WP2 HTTP 附件 | 后端团队 | upload/download/delete、JWT、错误、CORS 与负向测试 |
| WP3 订单创建绑定 | 后端团队 | `attachmentIds`、幂等兼容、订单摘要与事务测试 |
| WP4 cleanup + seed/reset | 后端团队 | 过期清理、失败残留清理、演示数据可重复 |
| WP5 收敛与验收 | 后端团队 | current API、验证矩阵 `enabled: yes`、场景、CHANGELOG 与全量门禁 |

工作包按 WP0 → WP1 → WP2/WP3 → WP4 → WP5 推进。每个工作包必须产生产品代码或直接验证产品行为；不得新增治理工作流类型。

## 风险、回退与停止条件

| 风险 | 缓解与回退 |
|---|---|
| 文件与数据库不能单事务提交 | 使用可识别临时文件和有限补偿；失败后不暴露成功元数据，cleanup 可重复 |
| 文件名或路径穿越 | 只使用服务端存储键；原始文件名仅作经过清理的下载展示元数据 |
| 幂等兼容破坏历史订单创建 | 保持无附件路径兼容；对含附件请求使用明确版本与固定测试 fixture |
| 范围重新膨胀 | 一旦 crash/调度/发布/Evidence 供应链成为前置，停止并拆新计划 |
| 阶段一至四回归 | 关闭附件路由或回退新增代码；additive migration 保持旧二进制可识别的失败边界 |

## 验收与 Evidence

必须具备以下可证伪证据：

1. 真实 HTTP：登录 → 上传 → 创建并绑定 → 重新查询 → 鉴权下载 → 删除未绑定附件；
2. 负向输入：超限、伪造类型、危险文件名、非法 ID、他人附件、过期附件、重复绑定和越权下载；
3. 数据一致性：上传/绑定失败不产生可见半成功；相同幂等 key 不创建第二个订单；
4. 可重复 Demo：全新目录 migrate + seed 后可运行场景，reset 后恢复确定基线；
5. `go test ./...`、`go vet ./...`，涉及并发状态时 `go test -race ./...`；
6. `docs/tools/validate.ps1` 与 `docs/tools/validate.tests.ps1`；
7. 附件行只有在真实入口存在并通过时才改为 `enabled: yes`；
8. 明确记录页面、生产部署、crash harness 与 Evidence 供应链未执行且不属于本计划。

配套 checklist：`PLN-0007-phase-05-attachment-mvp-checklist.md`。
