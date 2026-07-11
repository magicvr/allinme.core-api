---
status: active
owner: 后端团队
last_updated: 2026-07-12
applies_to: implemented allinme.core-api HTTP API
---

# 当前 HTTP API

本文只记录当前源码和测试已经实现的 HTTP 行为。待实现的业务原则与 endpoint 草案见 [目标 HTTP API](./03-http-api-target.md)，不得据此推断目标接口当前可调用。

## `GET /healthz`

用于进程存活检查（liveness）。只要 HTTP 进程可以处理请求即返回成功，不检查数据库、migrations、文件目录、页面配置或外部依赖。

成功响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

实现位于 `internal/httpapi/handler.go`，行为由 `internal/httpapi/handler_test.go` 验证。

## `GET /readyz`

数据库文件可访问且 `PRAGMA user_version` 等于二进制最新 migration 版本时返回：

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ready"}
```

数据库缺失、不可用、未迁移、低于或高于支持版本时返回 `503`：

```json
{"error":{"code":"NOT_READY","message":"service is not ready","requestId":"req_..."}}
```

非 `GET` 请求返回 `405`、`Allow: GET` 和 `METHOD_NOT_ALLOWED` envelope。阶段六启用页面模块时再扩展 readiness 检查。

## 运行错误与 Request ID

所有响应回写 `X-Request-ID`。入站值只接受 `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`；空值或非法值替换为 `req_` 加 32 位小写十六进制值。错误响应、header 和访问日志使用同一 request ID。

本阶段冻结 `NOT_READY`、`METHOD_NOT_ALLOWED` 和 `INTERNAL_ERROR`，错误结构为 `{"error":{"code","message","requestId"}}`。panic 发生在响应提交前时返回完整 `500 INTERNAL_ERROR`；提交后不重写已发送响应，只记录 panic 和最终可观察状态。

## 当前未实现

- `/api/v1/*` 认证、页面、订单、退款、附件和看板 API；
- 业务错误码、幂等和版本冲突处理。

新增 endpoint 只有在实现、测试和对应门禁齐全后才能写入本文。实现前的设计调整只修改 [目标 HTTP API](./03-http-api-target.md)。