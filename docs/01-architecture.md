---
status: active
owner: 后端团队
last_updated: 2026-07-11
applies_to: allinme.core-api
---

# 后端架构

## 1. 当前形态

服务使用 Go 标准库 HTTP 栈：`cmd/api/main.go` 启动进程，`internal/httpapi/handler.go` 注册路由。当前唯一 HTTP 端点是 `GET /healthz`；`internal/protocol/` 提供 Schema-UI 共享算法的 Go 实现和 conformance 证明，尚未暴露业务 API。

| 层次 | 路径 | 当前职责 |
|---|---|---|
| 进程入口 | `cmd/api/` | 配置监听地址、启动 HTTP 服务 |
| HTTP 适配 | `internal/httpapi/` | 路由、HTTP 输入输出和状态码 |
| 协议执行 | `internal/protocol/` | 与 Schema-UI conformance 对齐的纯算法 |

业务能力增长时应保持 transport、业务用例和协议算法边界，避免把鉴权或业务规则写进 fixture 适配代码。

## 2. HTTP 层规则

- handler 负责 method、Content-Type、输入限制、状态码和稳定错误结构。
- 请求 context 必须贯穿业务调用；阻塞 I/O 应响应取消和超时。
- 读取 body 时设置合理大小限制，JSON 输入应明确未知字段和多值处理策略。
- 写响应前完成可能失败的编码准备，避免部分响应后改变状态码。
- 认证、授权、幂等和资源状态由后端执行，不信任前端显隐、确认或页面配置。
- 新端点必须同步 [HTTP API](./03-http-api.md)、handler 测试和相关场景。

## 3. 协议模块约束

- Go 的 nil、空值、缺失字段和 JSON `null` 必须保持协议可观察语义。
- 需要区分缺失/空值时使用 `json.RawMessage`、指针或显式 presence 类型，不依赖零值猜测。
- query、path 和 body 使用结构化构造，并保持与参考实现相同的字节序列化。
- `scenario_execution.go` 只编排已定义步骤，不维护另一套业务语义。
- 每个协议模块的行为变化必须同步相邻测试，并由共享 fixture 提供正例和反例。

## 4. 依赖与安全

- 优先使用标准库；新增依赖必须有明确收益、固定版本和维护状态评估。
- `go.mod` / `go.sum` 必须由 Go 工具维护，不手工删改校验记录。
- 错误对外提供稳定、安全的信息，对内保留可诊断上下文；不得泄露 token、内部路径或敏感数据。
- goroutine、timer、response body 和其他资源必须有清晰所有权与关闭路径。

## 5. 依赖方向

`cmd/api` 可依赖 `internal/httpapi`；HTTP 层可编排未来业务用例；协议层不应依赖 HTTP handler。跨前后端契约变化遵循 [Schema-UI 接入契约](./02-schema-ui-integration.md)，本仓内部架构变化记录在 [`decisions/`](./decisions/README.md)。