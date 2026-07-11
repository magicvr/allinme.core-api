# allinme.core-api

Go API service for Allinme.

**文档入口：从 [`docs/00-overview.md`](./docs/00-overview.md) 开始阅读。**

本仓库是 Schema-UI 的后端消费者和业务 API 宿主。涉及页面结构、数据源、Action、Reaction、版本协商或前后端交互时，必须以 [`schema-ui-docs`](../schema-ui-docs/README.md) 的当前稳定文档与机器契约为核心契约；本仓文档只说明 API 实现、接入方式和验证证据，不重新定义协议。

当前可运行 HTTP 能力只有 `GET /healthz`。文档已初始化订单运营 demo 的目标态：SQLite 持久化、本地账号与 JWT、订单履约/退款、附件、经营看板，以及由后端下发的 Schema-UI 页面；这些目标能力按 [`docs/06-implementation-roadmap.md`](./docs/06-implementation-roadmap.md) 分阶段实施，不代表当前已可调用。

## Development

Run the service:

```sh
go run ./cmd/api
```

The server listens on port `8080` by default. Set `PORT` to override it. Health checks are available at `GET /healthz`.

Run the local quality gates with:

```sh
go test ./...
go vet ./...
```

Protocol conformance tests consume fixtures from the sibling `schema-ui-docs` repository by default. Set `SCHEMA_UI_FIXTURES` to override the fixture directory. CI uses a fixed Schema-UI commit so fixture inputs cannot drift between runs; the current pin and upgrade process are maintained only in [`docs/02-schema-ui-integration.md`](./docs/02-schema-ui-integration.md).