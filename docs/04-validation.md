---
status: active
owner: 后端团队
last_updated: 2026-07-12
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
- 设计阶段的响应结构与 [目标 HTTP API](./03-http-api-target.md) 及 Schema-UI mapping 一致；实现完成后与 [当前 HTTP API](./03-http-api.md) 一致。

## 4. 目标 demo 验证矩阵

以下门禁随对应能力实现启用；`enabled: no` 表示当前没有对应实现或自动化入口，不是当前发布门禁。能力落地时必须在同一变更中把对应行改为 `yes` 并记录实际命令或测试包。

| 能力 | enabled | 启用入口 | 最低可执行证据 |
|---|---|---|---|
| SQLite | yes | `go test ./internal/store -count=1` | 临时数据库覆盖 pragma、空库/重复 migration、事务回滚、版本分类和可重试 probe |
| seed/reset | yes | `go test ./internal/admin -count=1` | runtime seed 幂等、未来版本拒绝、production reset 拒绝及无关文件保留 |
| readiness | yes | `go test ./internal/httpapi ./internal/app -count=1` | method、ready/not-ready、超时、关闭、恢复、错误安全和 liveness 解耦 |
| 认证 | yes | `go test ./internal/auth ./internal/store ./internal/httpapi ./internal/app -count=1` | 严格 JWT、密码边界、真实 SQLite session、登录限流、login/me/logout、撤销 session 与测试专用角色策略 handler |
| 订单查询 | no | 阶段三 M1-B 只读 app 集成通过时 | 覆盖搜索/分页边界、详情、四角色读取和稳定排序；不等待 CORS 门禁 |
| 订单写入/幂等 | no | 阶段三 M2-B1 创建编辑/幂等集成通过时 | 覆盖金额、创建/编辑、相同 key 重放、不同 body 冲突和并发只创建一次；不等待 CORS 门禁 |
| 订单履约 Action | no | 阶段三 M3-A Action 集成通过时 | 覆盖乐观锁、全部合法状态转换、非法转换和版本冲突；不等待 CORS 门禁 |
| CORS | no | 阶段三 M3-B CORS 集成通过时 | 配置失败、actual/preflight、Vary、route metadata、短路优先级和自动化跨源 smoke |
| 退款 | no | 阶段四新增退款集成测试时 | 覆盖可退金额、申请/审批分离、审批事务和订单支付状态一致性 |
| 附件 | no | 阶段五新增文件集成测试时 | 临时目录覆盖超限、类型伪造、危险文件名、摘要、绑定权限、鉴权下载、失败清理和过期清理 |
| 看板 | no | 阶段四新增看板测试时 | 固定 seed 下统计快照与订单/退款查询结果交叉一致 |
| 页面 | no | 阶段六创建 `internal/pages/yaml/*.yaml` 时 | 全部 YAML 通过固定 Schema-UI 版本 L0-L4 校验，页面引用的 endpoint 与 Action 均存在集成测试 |

集成测试不得共享开发数据库、签名密钥或附件目录。时间、ID 和文件适配器应可注入，以稳定验证过期、幂等和清理行为。

## 5. 安全与并发验证

- 对密码、JWT、session、上传和错误响应增加敏感信息泄露断言；日志测试不得记录 Authorization header 或密码。
- 对登录增加速率限制测试，对 JSON/multipart 增加大小限制和未知字段测试。
- 所有 SQL 排序、筛选和 ID 查询使用结构化参数与 allowlist；测试恶意 query 不得改变语句结构。
- 实现 SQLite 写事务、session 撤销、幂等或并发 handler 后运行 `go test -race ./...`。
- 文件测试必须使用临时根目录并断言解析后的路径始终位于该根目录内。

## 6. 页面与前端联调

`internal/pages/yaml/*.yaml` 落地后，Go 测试必须使用 CI 固定的 `schema-ui-docs` checkout 执行页面内容校验；API 启动时还要对嵌入页面执行解析与版本/capability 防御性校验。验收至少覆盖：

1. 登录后取得角色可访问页面列表和 JSON 页面；
2. 搜索表单驱动服务端分页表格；
3. 表单 Reaction 不影响服务端校验；
4. 行级 Action 成功刷新，权限/状态/版本失败可见且不改变数据；
5. 上传返回附件 ID，随订单提交绑定并可鉴权下载；
6. 看板在订单或退款变化后反映同一数据口径。

## 7. CI 与协议升级

[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) 从 `go.mod` 读取 Go 版本，固定 Schema-UI commit 后运行 test 和 vet。协议 pin 变化只有在以下证据齐全时完成：

1. Schema-UI 固定对象永久可达；
2. 本地测试使用同一 fixture checkout 通过；
3. 当前消费者提交的远端 CI 成功；
4. Schema-UI 接入文档和 CHANGELOG 已同步；根 README 仅在叙述性接入说明变化时更新。

未执行的验证必须明确记录，不得写成“通过”。
