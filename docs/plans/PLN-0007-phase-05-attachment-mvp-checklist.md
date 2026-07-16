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

- [ ] 确认阶段一至四测试通过，附件与页面验证矩阵仍为 `enabled: no`。
- [ ] 仅同步实现所需的领域、目标 API、路线图和验证规则；未复制 `PLN-0005` 的生产发布规格。

## 1. Schema 与本地文件

- [ ] additive migration、附件元数据和订单映射已实现并覆盖升级/回滚测试。
- [ ] 文件只写入 `DATA_DIR` 下受控目录，存储键由服务端生成且不包含客户端路径。
- [ ] PDF/PNG/JPEG 内容探测、`10 MiB` 限制、SHA-256 与危险文件名测试通过。

## 2. HTTP 附件闭环

- [ ] `POST /api/v1/attachments` 返回稳定 ID、摘要和过期时间，不返回绝对路径或公开 URL。
- [ ] `GET /api/v1/attachments/{attachmentId}` 复用订单查看授权并覆盖未认证/不可见/损坏文件。
- [ ] `DELETE /api/v1/attachments/{attachmentId}` 只删除本人未绑定附件，已绑定返回冲突。
- [ ] 新路由的 method、错误 envelope、request ID、CORS 与路由禁用回退测试通过。

## 3. 订单创建绑定与幂等

- [ ] create DTO 支持有序唯一 `attachmentIds`，缺失/`null`/空数组等价，每单最多 10 个。
- [ ] 绑定事务验证 owner、expiry、状态与单附件单订单约束；失败无部分绑定。
- [ ] 含附件请求参与幂等摘要；相同 key 重放、不同附件冲突与并发只创建一次测试通过。
- [ ] 列表 `attachmentCount`、详情/写入附件摘要与现有无附件响应兼容。
- [ ] PATCH 明确拒绝 `attachmentIds`，未实现 edit 增删/重排。

## 4. Cleanup 与可重复 Demo

- [ ] 过期未绑定附件 cleanup 可重复执行，失败残留不泄露路径且不删除已绑定文件。
- [ ] development seed/reset 覆盖确定的附件演示基线。
- [ ] 全新数据目录完成 migrate + seed + API smoke：上传 → 绑定 → 重查 → 下载 → 删除未绑定。

## 5. 文档与门禁

- [ ] 已实现契约从 target 迁入 current API，领域、场景、路线图与 CHANGELOG 同步。
- [ ] 附件验证矩阵在真实测试入口存在后改为 `enabled: yes`；页面仍为 `no`。
- [ ] `go test ./...` 与 `go vet ./...` 通过；涉及共享状态时 race 通过。
- [ ] `docs/tools/validate.ps1` 与 `docs/tools/validate.tests.ps1` 通过。
- [ ] 未新增审计工作流类型、治理拓扑 validator 或生产级 Evidence 前置。

## 6. 完成与归档

- [ ] 完成报告记录已完成项、未执行项和剩余风险，明确页面/生产部署不在范围内。
- [ ] 取得用户归档确认后，将 plan/checklist 原地改为 `status: archived`，文件不移动。

## Evidence 模板

```text
- Date:
- Revision:
- Command:
- Result:
- Paths:
- Notes:
```
