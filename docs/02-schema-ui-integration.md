---
status: active
owner: 后端团队
last_updated: 2026-07-12
applies_to: schema-ui-protocol v1.0
---

# Schema-UI 后端接入契约

## 1. 契约来源

本仓是协议消费者和 API 宿主，不是协议定义仓。跨前后端行为按以下来源判定：

1. [`schema-ui-docs/docs/00-overview.md`](../../schema-ui-docs/docs/00-overview.md)：术语、职责和版本边界；
2. `01-node-protocol.md`、`02-reaction-expression.md`、`04-datasource-contract.md` 与 `07-actions-contract.md`：规范性语义；
3. `docs/schemas/` 与 `docs/schemas/component-registry.json`：机器结构与组件能力；
4. `conformance/fixtures/`：跨语言可执行行为；
5. `docs/05-scenarios/`：官方端到端页面示例。

本仓 API 文档、Go 类型或处理逻辑与这些来源冲突时，以协议仓当前固定版本为准。

## 2. 当前固定版本

CI 在 [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) 固定读取 Schema-UI commit：

```text
d2f0fc0877dc6550c9fe7e3635b25c7ec72b4ddd
```

该提交对应稳定 `v1.0.0` 发布，页面声明 `meta.protocolVersion: "1.0"`。本地测试默认从相邻 `../schema-ui-docs/conformance/fixtures` 读取；`SCHEMA_UI_FIXTURES` 仅用于显式选择同一结构的 fixture 目录。

## 3. 实现映射

| 协议能力 | 本仓实现 |
|---|---|
| 版本与 capabilities | `internal/protocol/version_negotiation.go` |
| query 序列化 | `internal/protocol/query_serialization.go` |
| request / row request | `internal/protocol/request_construction.go` |
| responseMapping | `internal/protocol/response_mapping.go` |
| 搜索表单与表格状态 | `internal/protocol/table_query_state.go` |
| Reaction 调度 | `internal/protocol/reaction_scheduler.go` |
| Action outcome | `internal/protocol/action_outcome.go` |
| 上传 | `internal/protocol/upload_execution.go` |
| 官方场景编排 | `internal/protocol/scenario_execution.go` |

对应 `*_test.go` 必须直接消费共享 fixtures。禁止复制 expected、跳过未知 case、维护私有 allowlist，或根据 fixture 名称返回特例结果。

## 4. 前后端对接规则

- 页面 YAML/JSON、数据源、Action、Reaction 和响应映射以 Schema-UI 文档为共同语言。
- API 返回结构必须满足协议 responseMapping 与分页契约；已实现业务端点记录在 [当前 HTTP API](./03-http-api.md)，待实现草案记录在 [目标 HTTP API](./03-http-api-target.md)。
- 认证、授权、幂等、资源状态和业务不变量始终由后端执行，不能信任页面配置或前端状态。
- 页面配置不得携带 token 或伪造用户身份；身份来自服务端验证的认证上下文。
- 发现契约缺口时，在 `schema-ui-docs` 提交 ADR、规范、Schema、fixture 和版本变更，本仓随后升级固定 SHA。

## 5. 升级流程

1. 在 Schema-UI 仓完成变更和全部 reference/conformance 门禁。
2. 选择永久可达的稳定 tag 或 main commit，不固定到临时分支提交。
3. 更新 CI pin、本文件和必要实现，并在 CHANGELOG 记录；根 README 仅在叙述性接入说明变化时更新。
4. 使用同一 checkout 运行 `go test ./...` 和 `go vet ./...`。
5. 等待消费者当前提交的远端 CI 成功后，才把升级记录为完成。