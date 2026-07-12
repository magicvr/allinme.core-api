# 阶段三开发计划与 Checklist 审视建议

审视日期：2026-07-12
审视对象：`docs/audit/0003-2026-07-12-plan.md`、`docs/audit/0003-2026-07-12-checklist.md`
交叉核对：领域模型、架构、实施路线图、目标 HTTP API、验证规则、migration/readiness 实现与 CI

## 结论

新版计划总体合理，已经解决上一版的主要执行问题：

- M1 只创建订单查询 schema，幂等表延后到 M2；
- 不再允许 M2-A 暴露缺少 snapshot/replay 的 create API；
- schema 明确采用 additive migration 和 roll-forward 策略；
- `go test -run` 已降为开发加速命令，完整包测试才是硬门禁；
- 事务语义归用例层，SQL 事务实现归 store，并允许后续引入事务执行器；
- checklist 已压缩到 34 项，能力项、集成 gate 和发布 gate 的结构清晰。

建议结论：**修正 1 个阻断项并澄清 2 个重要边界后开工**。其余设计可在现有框架下实施。

## 开工前必须明确

### 1. v4 迁移缺少可执行的兼容回退 gate（阻断）

计划第 198 行要求迁移前存在能够容忍 v3/v4 的无订单路由兼容 binary；但第 41 行又要求 M1 不得提前引入 v4，第 224 行的 M2-A 也不包含 v4，第 225 行的 M2-B 则同时交付 v4 migration 和 create/edit 路由。

当前实现会严格拒绝比 binary 更新的数据库：

- `internal/store/migrations.go:40`：`currentVersion > len(migrations)` 直接失败；
- `internal/store/readiness.go:27`：`version > latest` 分类为 `SchemaTooNew`。

因此 M2-B 部署并迁移到 v4 后，回退到 M1-B 或 M2-A binary 会不可用。P0-3 目前只有原则，没有指定哪个构建产物真正承担 v4 兼容基线。

建议二选一，优先方案 A：

- **方案 A：拆出 M2-B0 schema compatibility gate。** 先交付包含 migration v4、能在 v4 数据库启动、但不注册 create/edit 路由的 binary；验证后再在 M2-B1 打开完整 create/edit。两步仍可属于 M2，但必须有独立产物和部署顺序。
- **方案 B：实现显式 schema 兼容区间。** readiness/migrate 不再简单以 `LatestSchemaVersion()` 判断过新，而是声明 binary 可接受的最小/最大 schema，并用测试证明旧功能在 v4 上可用。该方案会改变现有 migration 语义，需同步架构和测试。

同时把 P0-3 拆成 v3、v4 两次迁移前置检查；不能在阶段开始时一次勾选后覆盖后续 v4 风险。

### 2. M2-A 的 PATCH 暴露时点有歧义（重要）

计划第 224 行和 checklist M2-2 使用“可独立启用的 edit/version 路径”，且只明确禁止注册 `POST /api/v1/orders`；计划第 225 行和 checklist M2-7 又要求 M2-B“一次性注册完整 create/edit 外部能力”。

这会让执行者无法确定 M2-A 是否允许真实 app 提前注册 `PATCH /api/v1/orders/{id}`。

推荐统一为：

> M2-A 只完成 service/store 层可测试的 create/edit 能力，不注册任何订单写 HTTP 路由；M2-B 在完整幂等 create 就绪后一次性注册 POST 与 PATCH。

如果确实希望 PATCH 在 M2-A 对外可用，则应把它定义为独立外部 gate，并补齐真实 token、HTTP 回归、文档和验证矩阵要求，不能继续称 M2-B 为一次性注册完整写能力。

### 3. 验证矩阵的启用时点应进入各里程碑 checklist（重要）

`docs/04-validation.md:53` 要求能力落地时在同一变更中把矩阵改为 `enabled: yes`。计划第 306-307 行也要求分别在 M1-B、M2-B、M3-A、M3-B 后启用。

但 checklist 只在最终 R2 检查一次，容易出现能力已合并、矩阵仍为 `no`，直到发布阶段才被发现。

建议直接加入对应 gate：

- M1-7：订单查询改为 `yes`；
- M2-8：订单写入/幂等改为 `yes`；
- M3-4：订单履约 Action 改为 `yes`；
- M3-8：CORS 改为 `yes`；
- R2 只复核四项状态与实际命令一致，不再承担首次更新。

## 执行中建议优化

### 4. M1 仍提前承担部分写侧领域逻辑

checklist M1-2 要求状态机和金额 checked arithmetic 全部通过，M2-1 又要求金额与溢出校验，M3 才真正交付状态 Action。这样会让“只读纵切”提前实现未使用的写侧代码，也造成验收重复。

建议按首次使用点拆分：

- M1：订单读模型、状态枚举、读取权限、capability、ID、clock；
- M2：金额计算、create/edit 校验与 version；
- M3：状态转换函数和 Action 冲突分类。

如果团队决定提前实现纯函数，也应在 M1-2 标注为非阻断准备项，不要让只读里程碑因未来写逻辑延迟。

### 5. 里程碑 race 证据缺少明确命令

M1-7、M2-8、M3-8 都要求“记录普通测试和 race 耗时”，但硬门禁只列普通 `go test`，完整 race 仅在 R4 明确。

建议明确一种规则：

- 每个里程碑运行受影响包的 `go test -race ... -count=1`，并在 Evidence 记录命令和耗时；或
- 里程碑只做普通包测试，删除 race 耗时占位，统一由 R4 执行全量 race。

前者更早发现问题，后者更精简；不建议保留当前无法判断是否完成的表述。

### 6. 计划正文很完整，但实施时应防止契约细节反向绑死内部实现

计划已明确区分外部硬契约、架构边界和推荐验证方式，这是正确方向。实施评审时应持续检查：helper 名、repository 方法名、SQL 条数观察方式、barrier 和连接中止手段只作为等价证据，不应因为测试先写成某种 seam 就升级为长期 API。

## 建议保留的设计

- 金额使用 `int64` 最小货币单位，服务端重算并执行 checked arithmetic；
- create 使用不可变 snapshot v1，重放不读取当前订单；
- version/state 冲突按存在性、version、状态稳定分类；
- COUNT 和 page 位于同一 SQLite snapshot，列表避免 N+1；
- route metadata 统一驱动 mux、Allow、HEAD、known path 和后续 CORS；
- CORS 配置在打开数据库和监听端口前校验；
- context cancel 与 SQLite BUSY/LOCKED 分开分类和测试；
- endpoint 只在代码与集成证据完成后从目标 API 迁入当前 API；
- 页面 YAML、退款、附件和看板保持在后续阶段，不扩大阶段三范围。

## 建议的最小修改清单

1. 为 v4 增加明确的 schema-only 兼容 binary gate，或实现并验证 schema 兼容区间。
2. 明确 M2-A 不注册任何写 HTTP 路由，或正式把 PATCH 拆成独立外部 gate。
3. 将四个验证矩阵更新动作移动到对应里程碑 checklist。
4. 将金额和状态转换测试移动到首次使用的 M2/M3，收紧 M1 只读边界。
5. 为里程碑 race 证据写出实际命令，或只在 R4 保留全量 race。

完成前 3 项后，计划具备稳定开工条件；第 4-5 项属于降低执行摩擦和重复验收的优化。
