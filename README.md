# allinme.core-api

Go 实现的 Demo API 核心服务：可运行演示、为 Admin 后台提供真实对接面，并为后续 API 项目提供可复用结构。

## 目标

1. **Demo API** — 可本地启动、可演示的业务闭环接口
2. **Admin 支撑** — 稳定的 HTTP/JSON 约定，便于前台管理面板对接
3. **可复用资产** — 清晰分层，便于迁移到新项目

## 快速开始

要求：Go 1.22+（当前环境为 1.26）。容器方式需 Docker / Docker Compose。

### 本地 Go

```bash
# 可选：复制环境变量示例
cp .env.example .env

# 启动
make run
# 或
go run ./cmd/server
```

### Docker

```bash
# 构建并前台运行（Ctrl+C 退出并删除容器）
make docker-run

# 或用 Compose 后台运行
make docker-up
make docker-logs
make docker-down
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
├── cmd/server/                 # 进程入口（调用 composition root）
├── internal/
│   ├── app/                    # composition root：唯一 New 具体实现并注入
│   ├── config/                 # 环境变量配置
│   ├── handler/                # HTTP 入站适配
│   ├── port/                   # 出站/入站接口（如 MetaStore）
│   ├── repository/
│   │   ├── sqlite/             # SQLite 实现（默认）
│   │   └── memory/             # 测试 double
│   ├── service/
│   │   ├── meta/               # 元数据应用服务（依赖 port）
│   │   ├── auth|order|…/       # 业务 BC 占位（GOAL-002）
│   │   └── …
│   ├── response/               # 统一 JSON 响应
│   └── server/                 # http.Server 组装
├── pkg/version/
├── docs/architecture/modular-ioc.md
└── …
```

约定：

- `cmd/` — 可执行入口，尽量薄
- `internal/app` — **唯一**组装根（IoC / 构造注入）
- `service` 只依赖 `port` 接口；换库只改 `repository/*` + `app` 接线
- `pkg/` — 可被其他模块/项目安全复用的小包

**如何新增业务模块 / 换 Repository**：见 [docs/architecture/modular-ioc.md](docs/architecture/modular-ioc.md)。

## 配置

| 变量 | 默认 | 说明 |
|------|------|------|
| `APP_NAME` | `allinme.core-api` | 应用名 |
| `APP_ENV` | `development` | 环境（镜像默认 `production`） |
| `APP_VERSION` | `0.1.0` | 版本 |
| `HTTP_ADDR` | `:8080` | 监听地址 |
| `HTTP_READ_TIMEOUT` | `5s` | 读超时 |
| `HTTP_WRITE_TIMEOUT` | `10s` | 写超时 |
| `HTTP_IDLE_TIMEOUT` | `60s` | 空闲超时 |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `DB_DRIVER` | `sqlite` | 预留多驱动；MVP 仅实现 sqlite |
| `SQLITE_PATH` | `data/demo.db` | SQLite 文件路径 |
| `HTTP_PORT` | `8080` | Compose 宿主机映射端口 |

Compose 可通过环境变量或取消注释 `env_file: .env` 覆盖配置。

## Docker 说明

- **多阶段构建**：`golang:1.26-alpine` 编译 → `alpine:3.21` 运行，静态链接、非 root 用户
- **健康检查**：容器内 `wget` 访问 `/healthz`（Dockerfile 与 Compose 均配置）
- **构建参数**：`VERSION` / `COMMIT` / `BUILT_AT` 注入 `pkg/version`
- **上下文裁剪**：见 `.dockerignore`（排除 `.git`、二进制、本地密钥等）

```bash
# 仅构建镜像
make docker-build

# 无缓存重建并启动
make docker-rebuild
```

## 常用命令

```bash
make run            # 本地启动
make build          # 输出 bin/allinme.core-api
make test           # 跑测试
make tidy           # go mod tidy
make fmt            # go fmt
make vet            # go vet
make docker-build   # 构建镜像
make docker-run     # 构建并前台运行
make docker-up      # Compose 后台启动
make docker-down    # 停止 Compose
make docker-logs    # 跟踪 api 日志
```

## 接口约定（初稿）

- 成功：JSON body，`Content-Type: application/json`
- 错误：`{"code":"...","message":"..."}`（见 `internal/response`）
- 健康检查：`GET /healthz`、`GET /readyz`（不进鉴权，便于部署探针）
- 业务 API：前缀 `/v1/...`（便于 Admin 与未来多版本共存）
