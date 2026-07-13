# Changelog

本文件记录 `allinme.core-api` 的 API、架构边界、协议 pin 和质量门禁变化。Schema-UI 协议本身的版本历史以 [`schema-ui-docs/docs/CHANGELOG.md`](../../schema-ui-docs/docs/CHANGELOG.md) 为准。

## Unreleased

### Added

- 实现阶段四退款闭环：additive schema v6、独立退款幂等 snapshot、可退额度、申请/审批/拒绝、writer fence/双 CAS、跨连接竞争、严格 UTF-8/JSON/JWT/CORS 路由和真实 app 重启集成。
- 实现阶段四经营看板：summary、订单状态分布、UTC 7/30 日趋势、固定 seed 快照、负 trend bucket、单 SQLite snapshot、退款后金额变化和独立路由禁用装配。
- 增加 schema-only v6 产物证据与旧 v5 binary 整库恢复验证，明确回退必须停写并恢复 migration 前主库/WAL/SHM 集合，恢复点之后写入全部丢失。
- 实现阶段三 M3-B 可选可信 origin CORS、严格 preflight/actual 短路、连接取消中止和结构化 access log outcome，并补充真实 app 跨源 smoke、重启回归与全仓测试证据。
- 实现阶段三 M3-A 订单履约 Action：confirm/fulfill/ship/complete/cancel 状态机、version 条件更新、冲突分类、并发单次成功、真实 JWT/SQLite 集成和路由关闭回退。
- 实现阶段三 M2 订单草稿创建/编辑：严格 JSON 与整数词法、服务端金额计算、version 乐观锁、四角色授权和真实 JWT/SQLite 集成。
- 增加 principal/method/route/key 作用域幂等、normalized SHA-256 digest、不可变 snapshot v1、重放/冲突、双 DB 竞争、连接等待取消及结构化 SQLite BUSY/LOCKED 分类。
- 实现阶段三 M1 订单只读纵切：参数化列表查询、同 snapshot COUNT/page、详情明细、四角色真实认证访问和只读 route gate。
- 增加订单查询 DTO/capability、稳定排序与分页边界、查询次数 observer、损坏数据/扫描失败和基础 app 回退测试。
- 实现阶段二认证授权：bcrypt 本地账号、严格 HS256 JWT、SQLite 可撤销 session、四角色策略和 login/me/logout API。
- 增加 development 四角色 auth seed、production 空库 `bootstrap-admin`、固定窗口登录限流和 migration v2。
- 实现阶段一运行基础：配置与应用装配、纯 Go SQLite、嵌入式 migration、runtime seed，以及 development-only reset。
- 增加 `GET /readyz`、统一运行错误 envelope、request ID、结构化访问日志和 panic recovery。
- 建立后端文档总纲、架构、Schema-UI 接入、HTTP API、验证、场景、ADR 与审计规则。
- 明确 Schema-UI 文档仓是前后端对接的核心契约，本仓文档不重新定义协议。
- 初始化订单运营 demo 的目标领域、状态机、角色权限、HTTP API、完整业务场景和分阶段实施路线。
- 记录 SQLite + 本地文件、本地 JWT + 可撤销 session、后端 YAML 页面 + JSON API 三项架构决策。
- 增加认证、幂等、并发、退款、附件、看板和页面联调的目标验证矩阵。
- 建立文档状态词汇和单一事实源规则，拆分当前 HTTP API 与 draft 目标 API。
- 明确 liveness/readiness、目标门禁启用状态，以及 `internal/pages/yaml/*.yaml` 的包内嵌入与校验链路。
- 收敛目标 API 的 baseline/draft 层级、场景错误语义和 endpoint 实现迁移流程。
- 建立阶段一 1A 运行基础开发计划与可执行 checklist，明确 migration/seed/reset、readiness 和错误框架的实施证据。
- 建立阶段二认证授权开发计划与可执行 checklist，冻结 auth migration/seed、JWT/session、角色策略、登录限流、安全验证和文档收敛边界。

### Fixed

- 加固阶段四退款与看板 transport 契约：退款 JSON 字段名严格区分大小写并拒绝重复/缺失字段，看板 trend 拒绝 URL 编码后的正负号；同步根 README 的阶段四能力状态，并更正归档审计对趋势索引 runtime 使用情况的过强表述。

### Changed

- 增加 additive migration v5 与独立 snapshot SHA-256，拒绝合法 item ID 替换；补充非 UTC `Z` 损坏时间、COUNT/page 同 SQLite snapshot 并发证据和 listener 提前失败 watcher 回收。
- 修复订单读取聚合完整性、v4 幂等记录升级重放、shutdown 超时强制关闭、严格 query string 解析、重复 CORS 单值 header 和编辑资源错误优先级；补充对应损坏数据与回归测试。
- 修复阶段三验收复核发现的 shutdown 超时资源生命周期、订单 `405`/`Allow`、写请求短路顺序、只读 SQLite 错误分类、64 KiB body 上限、非 ASCII 搜索原值和幂等快照完整性校验偏差，并更新项目总纲。
- 协议 fixture pin 的当前值统一由 Schema-UI 接入文档维护，README 和 CHANGELOG 不再复制 SHA。
