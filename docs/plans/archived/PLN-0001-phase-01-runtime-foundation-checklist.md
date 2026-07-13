---
status: archived
plan_id: PLN-0001
owner: 后端团队
created: 2026-07-12
last_updated: 2026-07-12
applies_to: implementation roadmap phase 1A runtime foundation
---

# 阶段一 1A：运行基础 Checklist

配套计划：[阶段一 1A 运行基础开发计划](./PLN-0001-phase-01-runtime-foundation.md)。只有实现、测试、文档与验证证据全部满足时才勾选；不得因“已有代码”提前标记完成。

## A. 开工前基线

- [x] A1. 记录 `go test ./...` 与 `go vet ./...` 的开工基线结果。
- [x] A2. 阅读候选 SQLite driver 的当前官方文档，确认 Go 1.26、Windows、context、事务和 DSN/pragma 支持。
- [x] A3. 固定 driver 版本并确认 `go.mod` / `go.sum` 只包含预期依赖变化。
- [x] A4. 确认本轮非目标：认证、账号/角色 seed、订单、附件、看板和页面配置均不进入实现。

证据：

- 基线命令与结果：2026-07-12，`go test ./...` 与 `go vet ./...` 均通过。
- driver 选择与版本：`modernc.org/sqlite v1.53.0`，官方文档 <https://pkg.go.dev/modernc.org/sqlite>。
- driver 核对结论：当前环境 `go1.26.0 windows/amd64`；纯 Go driver 通过 `database/sql` 支持 context 与事务，SQLite URI 支持 `mode=rw/rwc`，重复 `_pragma` 在每次物理连接打开时执行；Windows URI 与三项 pragma 已由 `internal/store` 实测。

## B. 配置与应用装配

- [x] B1. 表驱动测试冻结 `APP_ENV`、`PORT`、`DATA_DIR` 的默认值、合法值、未知模式、端口范围和生产要求。
- [x] B2. 监听地址只由 `PORT` 派生为 `:<PORT>`，阶段一不引入其他监听地址环境变量。
- [x] B3. 数据库固定派生为 `<DATA_DIR>/allinme.db`，sidecar 仅为同目录 `-wal`/`-shm` 文件。
- [x] B4. production 要求显式绝对 `DATA_DIR`；development 默认 `./data`。
- [x] B5. 配置错误在监听端口或打开/修改数据库之前返回。
- [x] B6. `cmd/api` 改为薄入口，配置、store、handler 与关闭顺序由 `internal/app` 组装。
- [x] B7. 现有 SIGINT/SIGTERM 优雅停机与 10 秒关闭上限保持有效。
- [x] B8. `go test ./internal/config ./internal/app -count=1` 通过。

证据：`go test ./internal/config ./internal/app -count=1` 通过；覆盖 development/production、端口范围、路径派生、migrate 后同进程恢复 ready 与关闭后 not-ready。

## C. SQLite 与 migrations

- [x] C1. store 使用 `database/sql`，driver 不泄露到 HTTP 或未来业务接口。
- [x] C2. driver DSN/connector 对每个物理连接初始化 `foreign_keys=ON`、`busy_timeout=5000`、`journal_mode=WAL`，不依赖一次性裸 `ExecContext` 配置连接池。
- [x] C3. 阶段一设置 `MaxOpenConns=1`、`MaxIdleConns=1`，在专用 `*sql.Conn` 上读取并验证三项 pragma；任一不符则打开失败。
- [x] C4. migration SQL 位于 `internal/store/migrations/*.sql`，通过包内 `go:embed` 加载。
- [x] C5. runner 校验版本有序且连续，拒绝重复版本、缺号和数据库版本高于二进制。
- [x] C6. 每个 migration 在事务内执行，失败时 schema 与 `PRAGMA user_version` 均不前进。
- [x] C7. 首个 migration 只创建 `seed_versions(name, version, applied_at)`，不创建业务表。
- [x] C8. 通用事务 helper 在回调失败时回滚数据，并由独立测试证明。
- [x] C9. 空库迁移到最新版本，重复 migrate 无副作用。
- [x] C10. 关闭并重开数据库后 schema 版本和基础数据保持一致。
- [x] C11. API readiness probe 使用不创建文件的打开模式，数据库不存在时不产生空 DB 文件且不阻止 HTTP 进程启动。
- [x] C12. readiness probe 每次都重新检查文件并允许重新打开/刷新状态，不永久缓存首次失败；admin migrate 后无需重启 API 即可转为 ready。
- [x] C13. store 以类型化状态区分 `database_missing`、`database_unavailable`、`schema_uninitialized`、`schema_outdated`、`schema_too_new`、`ready`，handler 不比较错误字符串；missing 仅指文件不存在，文件存在但打不开、权限不足、锁超时、损坏或其他 I/O/driver 错误归 unavailable。
- [x] C14. 不存在 DB、`user_version=0` 和高版本均有独立 store 集成测试；版本分类函数用单元测试覆盖 `0 < version < latest` → `schema_outdated`。
- [x] C15. checklist 明确记录：latest 仅为 v1 时不伪造真实低版本数据库；真实 `schema_outdated` 集成测试在 migration v2 加入。
- [x] C16. 所有 store 测试使用 `t.TempDir()`，不访问开发数据库。
- [x] C17. `go test ./internal/store -count=1` 通过。

证据：migration 最新版本为 v1，仅创建 `seed_versions`；`go test ./internal/store -count=1` 通过，覆盖 pragma、空库/重复 migration、失败回滚、事务回滚、missing 不创建、恢复探测和版本分类。latest 为 v1，真实 outdated 数据库集成测试按计划延后到 v2。

## D. Admin 命令与数据恢复

- [x] D1. `cmd/admin migrate` 使用与 API 相同的配置和 store 装配。
- [x] D2. seed runner 使用独立 `seed_versions` 表，不复用 `PRAGMA user_version`；应用与版本更新时间处于同一事务。
- [x] D3. 本轮 seed 写入可查询的 `runtime=1` 版本，且不创建账号或订单数据。
- [x] D4. `cmd/admin seed` 重复执行不产生重复记录或不同结果。
- [x] D5. 数据库中的 seed group 版本高于 runner 支持版本时拒绝执行且不修改数据。
- [x] D6. `cmd/admin reset` 仅在 development 模式且 API 进程已停止时可用；帮助文本和文档明确此前置条件。
- [x] D7. production reset 在删除数据库、WAL 或 SHM 文件之前失败。
- [x] D8. development reset 关闭连接，只清理 `allinme.db`、`allinme.db-wal`、`allinme.db-shm`，再按 migrate → seed 顺序恢复。
- [x] D9. reset 删除目标只由程序固定派生为 `allinme.db`、`allinme.db-wal`、`allinme.db-shm`，不接受调用方输入参与路径或名称匹配。
- [x] D10. reset 对规范化 `DATA_DIR` 和目标执行 `Lstat`/等价 Windows 检查，拒绝符号链接、junction 或可检测 reparse point。
- [x] D11. reset 不删除 `DATA_DIR` 或固定数据库文件集合之外的内容，并测试同目录无关文件保留。
- [x] D12. 目标文件被其他进程占用或删除失败时，reset 返回失败，不继续 migrate/seed，不报告部分成功。
- [x] D13. Windows smoke 验证 API 持有数据库句柄时 reset 安全失败，停止 API 后可成功；若平台无法稳定自动化，记录人工步骤与结果。
- [x] D14. 命令逻辑测试覆盖成功、未知子命令、配置失败、迁移失败、seed 失败和 reset 拒绝。

证据：`go test ./internal/admin -count=1` 通过。Windows smoke：migrate/seed 成功；API 运行时跨进程锁在删除前拒绝 reset 且 readiness 保持 200；停止 API 后 reset 成功恢复 migration v1 与 runtime seed v1。

## E. HTTP 运行面

- [x] E1. `NewHandler` 使用显式依赖，现有 `/healthz` 状态码、Content-Type 和响应体保持兼容。
- [x] E2. `/readyz` 在数据库可访问且 schema 最新时返回 `200`、JSON Content-Type 和 `{"status":"ready"}`。
- [x] E3. `/readyz` 在数据库失败、schema 未迁移或版本不匹配时返回 `503` 和 `NOT_READY` envelope。
- [x] E4. `/readyz` 错误响应不泄露 DSN、文件路径、SQL 或内部错误。
- [x] E5. `/healthz` 不依赖数据库状态；ready 失败时仍返回 `200`。
- [x] E6. request ID 仅接受 `[A-Za-z0-9][A-Za-z0-9._-]{0,63}`；空或非法输入生成 `req_` + 32 位小写十六进制值。
- [x] E7. 所有响应均回写最终 `X-Request-ID`；错误体、响应 header 和访问日志中的 request ID 一致。
- [x] E8. 中间件顺序为 request ID → access log → recovery → handler；response writer 跟踪 header/body 是否已提交，access log 记录最终可观察状态。
- [x] E9. 下游尚未提交响应时，panic recovery 返回完整 `INTERNAL_ERROR` envelope 且进程继续运行。
- [x] E10. 下游已提交 header 或部分 body 后 panic 时，不尝试重写为 JSON/500；记录 panic、保持进程存活，并保留已发送响应语义。
- [x] E11. `httpapi` 使用注入的 `*slog.Logger`，不依赖全局 logger；内存日志测试可断言属性。
- [x] E12. 结构化访问日志包含 request ID、method、path、status、duration，不记录敏感 header。
- [x] E13. 错误 envelope 冻结为 `error.code/message/requestId`，本阶段只冻结 `NOT_READY`、`METHOD_NOT_ALLOWED` 与 `INTERNAL_ERROR`。
- [x] E14. `/readyz` 对非 `GET` method 返回 `405`、`Allow: GET` 和 `METHOD_NOT_ALLOWED` envelope。
- [x] E15. readiness probe 贯穿 request context；取消或超时时及时返回，不遗留连接/资源，外部响应保持安全。
- [x] E16. readiness probe 使用明确的短超时，由测试证明阻塞 store 不会无限挂起 handler。
- [x] E17. store 或 app 已关闭后调用 `/readyz` 返回安全 not-ready，不 panic、不重开已关闭应用资源。
- [x] E18. `go test ./internal/httpapi -count=1` 通过。

证据：`go test ./internal/httpapi -count=1` 通过；覆盖 health/ready、405、safe envelope、request ID 透传/替换、超时、panic 提交前后与结构化日志属性。

## F. 集成与回归

- [x] F1. `internal/app` 集成测试在临时数据目录完成 migrate → seed → API 装配，`/healthz` 与 `/readyz` 均成功；逻辑不只存在于 `main`。
- [x] F2. 关闭并重开应用后 readiness 仍成功。
- [x] F3. API 启动不自动 migrate；数据库缺失、未迁移、损坏或版本过新使 readiness 失败，但 liveness 成功；真实低版本集成用例延后到 migration v2。
- [x] F4. 测试之间不共享数据库、端口、环境变量或全局可变状态。
- [x] F5. `go test ./...` 通过。
- [x] F6. `go vet ./...` 通过。
- [x] F7. `go test -race ./...` 通过；若环境不可运行，记录原因且不勾选，并至少补 `go test ./... -count=1` 与关键共享状态/并发路径说明。

证据：2026-07-12，`go test ./... -count=1`、`go vet ./...`、`go test -race ./...` 均通过；app 临时目录集成覆盖 missing、uninitialized、corrupt、too-new、外部 migrate 后恢复及关闭后调用。

## G. 文档与门禁收敛

- [x] G1. [当前 HTTP API](../../03-http-api.md) 新增已实现 `/readyz` 及 handler/test 证据。
- [x] G2. [目标 HTTP API](../../03-http-api-target.md) 删除 readiness draft，记录本阶段已冻结错误结构。
- [x] G3. [验证矩阵](../../04-validation.md) 的 SQLite、seed/reset、readiness 改为 `enabled: yes`，写明实际入口。
- [x] G4. [架构](../../01-architecture.md) 与 [路线图](../../06-implementation-roadmap.md) 同步实际目录、driver 和命令设计。
- [x] G5. CHANGELOG 记录阶段一 1A 已实现能力，不把后续目标写成已交付。
- [x] G6. `git diff --check` 通过，文档相对链接与 frontmatter 检查通过。

G6 证据必须记录实际命令与输出；相对链接检查不得只写“人工确认”。

```powershell
$errors = @()
Get-ChildItem docs -Recurse -File -Filter '*.md' | ForEach-Object {
	$file = $_
	$content = Get-Content $file.FullName -Raw
	[regex]::Matches($content, '\[[^\]]+\]\((?!https?://|#)([^)#]+)(?:#[^)]+)?\)') | ForEach-Object {
		$target = Join-Path $file.DirectoryName ([uri]::UnescapeDataString($_.Groups[1].Value))
		if (-not (Test-Path $target)) { $errors += "$($file.FullName) -> $target" }
	}
}
if ($errors.Count) { $errors; exit 1 }
```

frontmatter 最小自动检查：

```powershell
$errors = @()
$root = (Resolve-Path '.').Path.TrimEnd('\')
Get-ChildItem docs -Recurse -File -Filter '*.md' | ForEach-Object {
	$file = $_
	$content = Get-Content $file.FullName -Raw
	if ($content.StartsWith("---`n") -or $content.StartsWith("---`r`n")) {
		$relative = $file.FullName.Substring($root.Length + 1).Replace('\', '/')
		$keys = if ($relative -like 'docs/decisions/*') {
			@('status:', 'date:')
		} else {
			@('status:', 'owner:', 'last_updated:', 'applies_to:')
		}
		foreach ($key in $keys) {
			if ($content -notmatch "(?m)^$([regex]::Escape($key))\s*.+$") { $errors += "$($file.FullName) missing $key" }
		}
	}
}
if ($errors.Count) { $errors; exit 1 }
```

证据：2026-07-12，`git diff --check` 通过；checklist 中 PowerShell 相对链接检查输出 `relative links: ok`，frontmatter 检查输出 `frontmatter: ok`。

## H. 完成与归档

- [x] H1. 计划中的完成标准全部满足，非目标未被意外纳入。
- [x] H2. 确认本 checklist 完成即代表路线图阶段一完成；页面 readiness 留待阶段六扩展。
- [x] H3. 向用户汇报实现、验证、未执行项和剩余风险。
- [x] H4. 获得用户明确归档确认。
- [x] H5. plan/checklist 全部完成后移入 `archived/`，更新活跃与归档索引。

## 最终验证记录

| 日期 | 命令/检查 | 结果 | 备注 |
|---|---|---|---|
| 2026-07-12 | `go test ./... -count=1` | 通过 | 所有包通过 |
| 2026-07-12 | `go vet ./...` | 通过 | 无输出 |
| 2026-07-12 | `go test -race ./...` | 通过 | Windows amd64 race detector 通过 |
| 2026-07-12 | `git diff --check` | 通过 | 仅 Git 提示工作区 LF 将来可能转换为 CRLF，无 whitespace error |
