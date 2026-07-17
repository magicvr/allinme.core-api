---
status: active
plan_id: PLN-0007
owner: 后端团队
created: 2026-07-16
last_updated: 2026-07-16
applies_to: implementation roadmap phase 5 attachment MVP
---

# 阶段五：附件 MVP Checklist

配套计划：[`PLN-0007-phase-05-attachment-mvp.md`](./PLN-0007-phase-05-attachment-mvp.md)。历史规格 [`PLN-0005`](./PLN-0005-phase-05-attachment-lifecycle.md) 已归档，不再作为本 checklist 的门禁。

每个完成项记录日期、revision、命令、结果和 Evidence 路径；未执行项写明原因与风险。

## 0. 基线与事实源

- [x] 确认阶段一至四 targeted tests 通过，附件实现前页面验证矩阵保持 `enabled: no`。
- [x] 仅同步实现所需的领域、目标 API、路线图和验证规则；未复制 `PLN-0005` 的生产发布规格。

## 1. Schema 与本地文件

- [x] additive migration、附件元数据和订单映射已实现并覆盖升级/回滚测试。
- [x] 文件只写入 `DATA_DIR` 下受控目录，存储键由服务端生成且不包含客户端路径。
- [x] PDF/PNG/JPEG 内容探测、`10 MiB` 限制、SHA-256 与危险文件名测试通过。

## 2. HTTP 附件闭环

- [x] `POST /api/v1/attachments` 返回稳定 ID、摘要和过期时间，不返回绝对路径或公开 URL。
- [x] `GET /api/v1/attachments/{attachmentId}` 复用订单查看授权并覆盖未认证/不可见/损坏文件。
- [x] `DELETE /api/v1/attachments/{attachmentId}` 只删除本人未绑定附件，已绑定返回冲突。
- [x] 新路由的 method、错误 envelope、request ID、CORS 与路由禁用回退测试通过。

## 3. 订单创建绑定与幂等

- [x] create DTO 支持有序唯一 `attachmentIds`，缺失/`null`/空数组等价，每单最多 10 个。
- [x] 绑定事务验证 owner、expiry、状态与单附件单订单约束；失败无部分绑定。
- [x] 含附件请求参与幂等摘要；相同 key 重放、不同附件冲突与并发只创建一次测试通过。
- [x] 列表 `attachmentCount`、详情/写入附件摘要与现有无附件响应兼容。
- [x] PATCH 明确拒绝 `attachmentIds`，未实现 edit 增删/重排。

## 4. Cleanup 与可重复 Demo

- [x] 过期未绑定附件 cleanup 可重复执行，失败残留不泄露路径且不删除已绑定文件。
- [x] development seed/reset 覆盖确定的附件演示基线。
- [x] 全新数据目录完成 migrate + seed + API smoke；真实 socket 覆盖上传两个附件、创建绑定、幂等重放、list/detail、viewer 下载、删除未绑定、API 重启下载、cleanup 与 development reset。

## 5. 文档与门禁

- [x] 已实现契约从 target 迁入 current API，领域、场景、路线图与 CHANGELOG 同步。
- [x] 附件验证矩阵在真实测试入口存在后改为 `enabled: yes`；页面仍为 `no`。
- [ ] `go test ./...` 与 `go vet ./...` 通过；`go vet ./...` 和全部附件 targeted packages 已通过，全仓仅 `internal/protocol` 既有 fixture conformance 偏差失败；race 因 Windows ThreadSanitizer shadow-memory 分配失败未执行完成。
- [x] `docs/tools/validate.ps1` 与 `docs/tools/validate.tests.ps1` 通过（以当前 PowerShell execution policy 运行，未使用 Bypass）。
- [x] 未新增审计工作流类型、治理拓扑 validator 或生产级 Evidence 前置。

## 6. 完成与归档

- [ ] 完成报告记录已完成项、未执行项和剩余风险，明确页面/生产部署不在范围内。
- [ ] 取得用户归档确认后，将 plan/checklist 原地改为 `status: archived`，文件不移动。

## Evidence 模板

```text
- Date: 2026-07-17
- Revision: working tree (no commit)
- Command: `go test ./internal/files ./internal/order ./internal/store ./internal/httpapi ./internal/app ./internal/admin -count=1`
- Result: attachment targeted packages, `go vet ./...`, runtime socket/CLI smoke and both documentation validators passed; `go test ./...` remains blocked only by pre-existing `internal/protocol` fixture conformance mismatches; race could not start because ThreadSanitizer could not allocate shadow memory on Windows
- Paths: `internal/files`, `internal/order`, `internal/store`, `internal/httpapi`, `internal/app`, `internal/admin`
- Notes: 页面、订单附件编辑、ORDER_DELETE、crash harness、调度、build tag/capability、Evidence 供应链未实现且不属于 PLN-0007
```
