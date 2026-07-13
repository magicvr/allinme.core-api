---
status: active
owner: 后端团队
last_updated: 2026-07-13
applies_to: order operations demo target
---

# Demo API 实施路线图

## 1. 使用方式

本路线从已实现运行基础、认证授权、订单、退款和看板服务推进到完整订单运营 demo。阶段按依赖顺序实施；每阶段只有在代码、测试、API 文档和场景证据齐全后才标记完成。后续阶段的目标接口不代表当前可用。

## 2. 阶段一：运行基础（已实现）

目标：建立可测试、可重置的数据与服务装配基础。

- 定义配置加载与开发/生产模式，创建 SQLite 连接、migrations 和事务边界；
- 增加 migrate、seed、reset 开发命令和可扩展 seed runner；本阶段只写基础 seed 元数据；
- 建立统一 JSON 错误结构、request ID、结构化日志与 panic recovery；
- 将 handler 改为显式依赖注入，保留 liveness `GET /healthz`，新增 readiness `GET /readyz` 检查 migrations 与 SQLite。

完成证据：空库迁移、重复迁移、reset/seed、进程重启持久化和错误映射测试通过。

实现使用 `modernc.org/sqlite v1.53.0`、`cmd/admin` 和 `internal/config|app|store`。实施证据见已归档的 [阶段一 1A 计划](./audit/archived/0001-2026-07-12-plan.md) 与 [checklist](./audit/archived/0001-2026-07-12-checklist.md)。角色账号依赖认证 schema，在阶段二补入 seed；订单和关键业务状态依赖订单 schema，在阶段三补入 seed。

## 3. 阶段二：认证授权（已实现）

目标：本地账号登录、JWT Bearer 和可撤销会话可独立工作。

实施证据见已归档的 [阶段二认证授权计划](./audit/archived/0002-2026-07-12-plan.md) 与 [checklist](./audit/archived/0002-2026-07-12-checklist.md)。已实现 migration v2、development auth seed、production bootstrap、密钥配置、严格 JWT/session 校验、登录限流、角色策略和三条认证 API。

- 实现密码哈希、登录、当前用户和登出；
- JWT 使用短时效、唯一 token ID，并关联 SQLite session；
- 建立 `viewer`、`operator`、`approver`、`admin` 授权策略；
- 扩展 seed runner，创建四类本地演示账号；
- 覆盖错误密码、禁用账号、过期/篡改 token、撤销会话和越权访问。

完成证据：认证集成测试使用真实 HTTP 与临时 SQLite，日志和响应不泄露密码、哈希或 token。

## 4. 阶段三：订单查询与履约（已实现）

目标：打通订单查询、草稿写入和履约 API/CORS 闭环，为阶段六页面配置提供稳定契约。

阶段三按三个端到端里程碑实施，详细门禁见[归档计划](./audit/archived/0003-2026-07-12-plan.md)与[checklist](./audit/archived/0003-2026-07-12-checklist.md)：M1 交付真实登录可用的列表/详情纵切，M2 交付真实 app 中的创建/编辑/幂等纵切，M3 交付履约 Action、CORS、重启/reset 和文档收敛。schema migration 只前进；每个里程碑都必须可启动、可演示、可独立合并，并可通过关闭对应路由或配置回退功能且保持已迁移数据库可用。

- 实现订单搜索、筛选、排序、分页和详情；
- 实现创建、编辑、确认、开始履约、发货、完成和取消；
- 使用整数金额、事务和 `version` 乐观锁；
- 返回基于当前主体和资源状态计算的 `canXxx` 展示字段；
- 扩展 seed runner，覆盖关键订单与支付状态，为退款阶段提供前置数据。

完成证据：M1/M2/M3 的真实认证 HTTP fixture、非法状态/权限/并发测试和自动化可信 origin OPTIONS + 订单请求 smoke 通过。阶段三不创建页面 YAML、不启动页面模块，真实页面加载、Schema-UI mapping 和 L0-L4 校验仍属于阶段六。

## 5. 阶段四：退款与看板（已实现）

目标：覆盖需要审批的状态变化和多数据源只读页面。

- 实现退款幂等创建、审批、拒绝和订单可退款金额计算；
- 强制申请人与审批人分离，所有退款状态变更带版本校验；
- 实现订单数、原始已支付金额、已完成退款金额、净额、状态分布和 7/30 日趋势；
- 明确 seed 数据与看板统计口径，测试跨接口数据一致性。

完成证据：退款写请求成功或失败后重新查询退款队列与订单可得到一致结果，看板响应与订单/退款数据一致；真实页面 Action reload 与渲染证据属于阶段六。

实施证据见已归档的 [阶段四计划](./audit/archived/0004-2026-07-13-plan.md) 与 [checklist](./audit/archived/0004-2026-07-13-checklist.md)：已实现 additive schema v6、退款独立幂等 snapshot、writer fence/CAS 并发不变量、严格 HTTP/JWT/CORS 闭环、固定 seed 看板、UTC 7/30 日趋势、旧 v5 整库恢复边界及全仓 test/vet/race 门禁。

## 6. 阶段五：附件

目标：完成先上传、后绑定、鉴权下载的文件生命周期。

- 实现 multipart 限制、允许类型、服务端文件名、摘要和临时附件记录；
- 创建/编辑订单时绑定附件，拒绝绑定他人、过期或已绑定附件；
- 实现鉴权下载、订单删除清理和未绑定文件过期清理；
- 测试超限、伪造类型、路径穿越、部分失败和磁盘错误。

完成证据：Schema-UI UploadAction 返回稳定附件 ID，订单提交后可通过受保护 API 下载。

## 7. 阶段六：页面配置

目标：后端下发经过固定协议版本校验的完整 demo 页面。

- 在 `internal/pages/yaml/*.yaml` 维护订单列表、订单编辑、退款处理、经营看板和附件场景；
- Go 测试使用 `schema-ui-docs` 固定版本校验页面，`internal/pages` 通过 `go:embed` 嵌入源文件；
- 启动时再次解析和校验嵌入页面，失败则拒绝启动；通过 allowlist 页面 ID 返回 JSON；
- 按角色控制页面入口可见性，API 继续独立授权。

完成证据：所有页面通过协议 L0-L4 校验，页面声明的 datasource/Action 均有实际端点和集成测试，`/readyz` 纳入页面模块状态。

## 8. 完整验收

- 执行 `go test ./...`、`go vet ./...` 和涉及并发/共享状态时的 `go test -race ./...`；
- 在全新数据目录执行 migrate + seed，启动 API，并完成登录、查询、履约、退款、上传、下载和看板 smoke test；
- 用相邻前端仓加载后端页面 JSON，验证六类 Schema-UI 标准场景；
- 检查目标文档中的 endpoint、角色、状态和验证证据均与实现一致；
- 将已实现 endpoint 从目标 API 迁入当前 API，并删除或缩减对应 draft；
- 将验证矩阵对应行改为 `enabled: yes`，补充实际命令、场景证据和 CHANGELOG；
- 仅当整份领域或场景文档均已由实现覆盖时，才把文件级 `target`/`planned` 改为 `active`。
