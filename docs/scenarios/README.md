# 后端场景索引

本目录用于记录 `allinme.core-api` 支持的真实业务工作流、权限边界、状态变化和 API 验收证据。Schema-UI 官方场景仍由 [`schema-ui-docs/docs/05-scenarios/`](../../../schema-ui-docs/docs/05-scenarios/README.md) 权威定义，本目录不得复制或修改其中的 YAML。

## 当前覆盖

| 场景 | 当前状态 | 证据 |
|---|---|---|
| HTTP 存活检查 | 已实现 | `internal/httpapi/handler_test.go` |
| Schema-UI 六个官方场景算法编排 | 已由 conformance 覆盖 | `internal/protocol/scenario_execution_test.go` |
| [订单运营完整演示](./order-operations-demo.md) | planned | 目标业务流程与验收标准已定义，代码尚未实现 |

## 新增场景规则

每个业务场景单独使用 Markdown 文件，至少记录：

- 业务目标、参与者、认证和权限；
- 对应的 Schema-UI 官方场景/协议章节；
- 请求、状态变化、成功和失败路径；
- 幂等、并发、审计和敏感数据边界；
- 前端调用方式及 handler/业务/conformance 测试证据。

若场景需要协议尚未定义的新字段或语义，先在 Schema-UI 仓完成 ADR 与版本变更，不能在本目录创建私有协议。