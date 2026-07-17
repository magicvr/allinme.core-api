# allinme.core-api

Go 实现的 Demo API 核心服务：可运行演示、为 Admin 后台提供真实对接面，并为后续 API 项目提供可复用结构。

## 目标

1. **Demo API** — 可本地启动、可演示的业务闭环接口
2. **Admin 支撑** — 稳定的 HTTP/JSON 约定，便于前台管理面板对接
3. **可复用资产** — 清晰分层，便于迁移到新项目

## 快速开始

要求：Go 1.22+（当前环境为 1.26）

```bash
# 可选：复制环境变量示例
cp .env.example .env

# 启动
make run
# 或
go run ./cmd/server
```

默认监听 `:8080`。

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/v1/ping
```

## 项目结构

```
.
├── cmd/server/          # 进程入口
├── internal/
│   ├── config/          # 环境变量配置
│   ├── handler/         # HTTP 路由与处理器
│   ├── response/        # 统一 JSON 响应
│   └── server/          # http.Server 组装
├── pkg/
│   └── version/         # 版本信息（可 -ldflags 注入）
├── .env.example
├── .gitignore
├── go.mod
├── Makefile
└── README.md
```

约定：

- `cmd/` — 可执行入口，尽量薄
- `internal/` — 本服务私有逻辑，不对外 import
- `pkg/` — 可被其他模块/项目安全复用的小包

后续可按需增加 `internal/service`、`internal/repository`、`internal/middleware`、`api/openapi` 等，优先服务 Demo 闭环与 Admin 对接，避免空转的治理层。

## 配置

| 变量 | 默认 | 说明 |
|------|------|------|
| `APP_NAME` | `allinme.core-api` | 应用名 |
| `APP_ENV` | `development` | 环境 |
| `APP_VERSION` | `0.1.0` | 版本 |
| `HTTP_ADDR` | `:8080` | 监听地址 |
| `HTTP_READ_TIMEOUT` | `5s` | 读超时 |
| `HTTP_WRITE_TIMEOUT` | `10s` | 写超时 |
| `HTTP_IDLE_TIMEOUT` | `60s` | 空闲超时 |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

## 常用命令

```bash
make run    # 本地启动
make build  # 输出 bin/allinme.core-api
make test   # 跑测试
make tidy   # go mod tidy
make fmt    # go fmt
make vet    # go vet
```

## 接口约定（初稿）

- 成功：JSON body，`Content-Type: application/json`
- 错误：`{"code":"...","message":"..."}`（见 `internal/response`）
- 健康检查：`GET /healthz`、`GET /readyz`（不进鉴权，便于部署探针）
- 业务 API：前缀 `/v1/...`（便于 Admin 与未来多版本共存）
