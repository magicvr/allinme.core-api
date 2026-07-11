# allinme.core-api

Go API service for Allinme.

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

Protocol conformance tests consume fixtures from the sibling `schema-ui-docs` repository by default. Set `SCHEMA_UI_FIXTURES` to override the fixture directory. CI checks out `magicvr/schema-ui-docs` at commit `152501fb1b8cade02f2780d96a82b5ceb2f5d281` so fixture inputs cannot drift between runs.