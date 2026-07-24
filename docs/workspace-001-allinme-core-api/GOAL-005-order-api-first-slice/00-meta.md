---
id: GOAL-005-order-api-first-slice
title: 订单 API 首切片
status: done
parent: GOAL-002-mvp-demo-admin
created: 2026-07-25
updated: 2026-07-25
version: 0.1.0
progress: 100%
---

# GOAL-005 · 订单 API 首切片

## 概述

依据 GOAL-002 已发生的 M3 订单实施事实补录本子目标：交付订单领域、SQLite repository、服务用例、Bearer/RBAC HTTP API、种子数据及跨层测试的第一个完整切片。

> 本目标是“订单 API 首切片”，不是订单域全量完成。单项 `DELETE` 与 `refund` action 仍属于 GOAL-002 后续范围。

## 成功标准

- [x] 订单领域状态、Repository port 与 service 用例落地
- [x] SQLite schema、查询、CAS、事务 batch-delete 与幂等种子落地
- [x] list/detail/create/update/mark-paid/cancel/batch-delete API 按 D-018 实现
- [x] Bearer 与 admin/operator/viewer 读写边界落实
- [x] 分页、搜索、version CAS、状态机与错误 envelope 有跨层测试
- [x] `go test -count=1 ./...`、`go vet ./...`、`git diff --check` 在实施完成时通过

## 范围外 / 后续

- 单项 `DELETE /v1/orders/{id}`
- `refund` action 与 paid→refunded HTTP 路径
- 对应 Page Schema（属于 GOAL-002 后续阶段）

## 信息就绪与未知项

本首切片的实施契约已由父目标 D-018 与 A-004/A-005 闭环。补录时无新增 required 信息项；后续 DELETE/refund 的具体契约须在其实施目标开始前审视。

## 父目标

- [GOAL-002-mvp-demo-admin](../GOAL-002-mvp-demo-admin/00-meta.md)

## 备注

- 父目标路线图阶段：M3 的订单首切片。
- 实施与验证事实发生于 2026-07-25；同日补录目标结构。
