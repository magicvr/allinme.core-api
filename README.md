# allinme.core-api

Go API service for Allinme.

**文档入口：从 [`docs/00-overview.md`](./docs/00-overview.md) 开始阅读。**

本仓库是 Schema-UI 的后端消费者和业务 API 宿主。涉及页面结构、数据源、Action、Reaction、版本协商或前后端交互时，必须以 [`schema-ui-docs`](../schema-ui-docs/README.md) 的当前稳定文档与机器契约为核心契约；本仓文档只说明 API 实现、接入方式和验证证据，不重新定义协议。

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

Protocol conformance tests consume fixtures from the sibling `schema-ui-docs` repository by default. Set `SCHEMA_UI_FIXTURES` to override the fixture directory. CI checks out `magicvr/schema-ui-docs` at commit `d2f0fc0877dc6550c9fe7e3635b25c7ec72b4ddd` so fixture inputs cannot drift between runs.