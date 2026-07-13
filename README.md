# allinme.core-api

Go API service for Allinme.

**文档入口：从 [`docs/00-overview.md`](./docs/00-overview.md) 开始阅读。**

本仓库是 Schema-UI 的后端消费者和业务 API 宿主。涉及页面结构、数据源、Action、Reaction、版本协商或前后端交互时，必须以 [`schema-ui-docs`](../schema-ui-docs/README.md) 的当前稳定文档与机器契约为核心契约；本仓文档只说明 API 实现、接入方式和验证证据，不重新定义协议。

当前已实现阶段一运行基础、阶段二认证授权、阶段三订单查询与履约，以及阶段四退款和经营看板：SQLite migration/seed/reset、`GET /healthz`、`GET /readyz`、login/me/logout JWT Bearer API、订单列表/详情、幂等订单创建、草稿编辑、履约 Action、可信 origin CORS、幂等退款申请、审批/拒绝、可退金额，以及 summary、订单状态分布和 UTC 7/30 日趋势。附件和 Schema-UI 页面仍按 [`docs/06-implementation-roadmap.md`](./docs/06-implementation-roadmap.md) 分阶段实施。

## Development

Run the service:

```sh
export JWT_SIGNING_KEY="<at-least-32-bytes>"
go run ./cmd/api
```

The server listens on port `8080` by default. Set `PORT` to override it. Development data defaults to `./data`; production requires explicit `APP_ENV=production`, `PORT`, and an absolute `DATA_DIR`.

Initialize or reset the local database before starting the API:

```sh
go run ./cmd/admin -- migrate
export DEMO_ACCOUNT_PASSWORD="<12-to-72-byte-password>"
go run ./cmd/admin -- seed
go run ./cmd/admin -- reset
```

Development seed creates `viewer`, `operator`, `approver`, and `admin`; each username matches its role and uses `DEMO_ACCOUNT_PASSWORD`. `reset` is development-only and requires the API process to be stopped. Production does not create demo users: after migrate, set `BOOTSTRAP_ADMIN_USERNAME` and `BOOTSTRAP_ADMIN_PASSWORD`, then run `go run ./cmd/admin -- bootstrap-admin` once against an empty users table.

Known limits: no refresh token or session purge, rate limiting is in-memory and single-process, proxy headers are not trusted, CORS requires one explicitly configured allowed origin, bootstrap only works on an empty users table, and usernames use trim plus lowercase without Unicode NFKC normalization.

Run the local quality gates with:

```sh
go test ./...
go vet ./...
```

Protocol conformance tests consume fixtures from the sibling `schema-ui-docs` repository by default. Set `SCHEMA_UI_FIXTURES` to override the fixture directory. CI uses a fixed Schema-UI commit so fixture inputs cannot drift between runs; the current pin and upgrade process are maintained only in [`docs/02-schema-ui-integration.md`](./docs/02-schema-ui-integration.md).
