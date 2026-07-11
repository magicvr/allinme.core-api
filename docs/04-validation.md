---
status: active
owner: 后端团队
last_updated: 2026-07-11
applies_to: allinme.core-api
---

# 后端验证规则

## 1. 本地门禁

```sh
go test ./...
go vet ./...
```

| 命令 | 证明内容 | 不能证明 |
|---|---|---|
| `go test ./...` | HTTP 与协议算法测试、共享 fixtures 的当前行为 | 生产依赖可用或远端部署健康 |
| `go vet ./...` | Go 静态分析未发现已知问题 | 并发安全和业务语义完整 |

涉及共享状态、goroutine 或并发 handler 时增加：

```sh
go test -race ./...
```

依赖变化后运行 `go mod tidy`，并确认 `go.mod` / `go.sum` 的 diff 只包含预期变化。

## 2. Conformance 输入

本地默认读取相邻 `../schema-ui-docs/conformance/fixtures`。需要复现 CI 时，将 `SCHEMA_UI_FIXTURES` 指向 CI 固定 commit 的 `conformance/fixtures`：

```powershell
$env:SCHEMA_UI_FIXTURES = "<schema-ui-checkout>\conformance\fixtures"
go test ./...
```

测试必须执行全部 versioned cases 和官方场景，不得通过 skip、allowlist 或复制 expected 缩小范围。官方场景 meta 应来自 Markdown YAML fence，并与 fixture 声明交叉确认。

## 3. HTTP 变化验证

新增或修改端点时至少覆盖：

- 正确 method、path、状态码和 Content-Type；
- 合法、缺失、格式错误、超限和未知字段输入；
- 认证失败、权限不足、资源不存在、冲突和内部错误；
- context 取消、超时、资源关闭和重复请求语义；
- 响应结构与 [HTTP API](./03-http-api.md) 及 Schema-UI mapping 一致。

## 4. CI 与协议升级

[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) 从 `go.mod` 读取 Go 版本，固定 Schema-UI commit 后运行 test 和 vet。协议 pin 变化只有在以下证据齐全时完成：

1. Schema-UI 固定对象永久可达；
2. 本地测试使用同一 fixture checkout 通过；
3. 当前消费者提交的远端 CI 成功；
4. README、接入文档和 CHANGELOG 已同步。

未执行的验证必须明确记录，不得写成“通过”。