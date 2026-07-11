---
status: active
owner: 后端团队
last_updated: 2026-07-11
applies_to: allinme.core-api HTTP API
---

# HTTP API

## 1. 当前端点

### `GET /healthz`

用于进程存活检查。

成功响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

实现位于 `internal/httpapi/handler.go`，行为由 `internal/httpapi/handler_test.go` 验证。当前端点只证明 HTTP 进程可响应，不证明数据库、外部依赖或 Schema-UI 业务能力就绪。

## 2. Schema-UI 业务端点

当前尚未实现页面配置、数据源、表单提交、行级 Action 或上传 HTTP 端点。未来端点必须先引用 [`schema-ui-docs`](../../schema-ui-docs/README.md) 中对应的数据源/Action/上传契约，再在本文件补充业务特有内容。

每个新增端点至少记录：

- method、path、认证和权限；
- path/query/header/body 结构与大小限制；
- 成功状态码和响应结构；
- 稳定错误码、错误体与重试/幂等语义；
- Schema-UI `datasource` 或 `action` 映射；
- handler、业务和协议测试证据。

## 3. 跨端契约纪律

- 本文件不得创造 Schema-UI 私有字段。
- 前端需要的分页、responseMapping、Action outcome 和上传行为以协议仓为核心契约。
- API 的业务字段和权限可以由本仓定义，但必须能映射到协议已有结构。
- 若无法无损映射，先在 Schema-UI 仓提出 ADR/协议变更，再实现两端。