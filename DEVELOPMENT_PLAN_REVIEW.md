# 阶段三开发计划与 Checklist 审视建议

审视日期：2026-07-12  
审视对象：`docs/audit/0003-2026-07-12-plan.md`、`docs/audit/0003-2026-07-12-checklist.md`，以及相关领域模型、目标 API、实施路线图、验证规则和当前代码基线。

## 结论

计划的范围、领域规则、HTTP 契约、里程碑依赖和测试门禁总体合理，已经具备进入实际代码开发的主要条件，但当前应按“有限开工”处理，而不是直接全面启动 M1。

- 可以立即开始：`internal/order` 的包结构、读模型、状态枚举、ID/clock、权限与 capability 纯领域测试，以及不改变数据库 schema 的 repository contract 设计。
- 暂不应开始：migration v3、订单 seed、schema-only 发布产物、M1-B 外部订单路由。计划和 checklist 已明确要求这些工作开始前完成 `P0-3a`，但该项目前仍未勾选且没有可执行 Evidence。
- 补完 `P0-3a` 并刷新当前 HEAD 的基线证据后，可以按 M1-A -> M1-B -> M2-A -> M2-B0 -> M2-B1 -> M3-A -> M3-B 顺序正常开发。

因此，当前状态建议标记为：**有条件可开工；schema 与对外 API 尚未达到开工门禁。**

## 审视结果

### 1. 计划结构与范围合理

- 阶段三明确排除了退款、附件、看板、页面 YAML、账号管理、库存、支付网关和物流集成，范围与路线图一致。
- M1/M2/M3 均为可启动、可测试、可关闭的纵向切片，避免长期存在“包已完成但真实 app 未装配”的半成品状态。
- migration v3/v4 分离合理：只读 schema 不提前引入幂等表，写入能力在 v4 schema-only 基线验证后一次性开放。
- 订单金额、状态机、角色、乐观锁、幂等 snapshot、错误优先级和 CORS 行为已经足够明确，开发中不需要再临时决定核心外部契约。
- 测试覆盖了普通路径、权限、并发、取消、BUSY/LOCKED、损坏数据、重放和回退，风险覆盖与本阶段复杂度相称。

### 2. 当前唯一明确的开工阻断项是 P0-3a

`P0-3a` 要求在 migration v3 前冻结 v3/v4 的目标环境、固定产物、路由关闭机制、数据库备份/恢复步骤、回退触发条件和数据损失边界。该要求与 plan §8 的 roll-forward 策略一致，但 checklist 中仍为未完成状态。

建议先新增一份可执行的发布/恢复 Evidence，至少写明：

1. 目标环境是仅本地 demo、共享测试环境，还是存在生产式部署。
2. M1-A、M2-B0 固定产物的构建命令、保存位置和 SHA-256 记录方式。
3. schema-only 产物如何确保对应订单路由未注册，而不是依赖人工约定。
4. SQLite 停写后主库、WAL、SHM 的一致性备份命令和恢复命令。
5. v3/v4 发布失败时选择兼容产物回退还是整库恢复，以及各自的数据损失边界。

如果项目目前没有真实部署环境，应明确将 P0-3a 的目标环境限定为“本地/CI 演练”，而不是保留无法兑现的生产发布措辞。

### 3. P0-4 基线证据需要刷新

checklist 的 P0-4 记录 revision 为 `66b1f219c9b58757f76ec46b812daaa01ef96cf1`，当前审视时 HEAD 为 `b7222b5c16f11967f9776ee81b3aabaa38115ae0`。虽然当前 HEAD 的测试仍然通过，但 checklist 证据没有对应最新 revision。

本次审视在当前 HEAD 得到以下结果：

- `go test ./... -count=1`：通过，23.5 秒。
- `go vet ./...`：通过，5.8 秒。
- `go test -race ./... -count=1`：通过，88.6 秒。
- `docs/audit/validate.ps1`：通过，验证 24 个 Markdown 文件。
- `docs/audit/validate.tests.ps1`：通过。

建议更新 P0-4 Evidence 到当前实际开工 revision，或明确规定“只要 P0 后续仅修改文档，基线可引用最近一次代码 revision”。前者更直接，也更容易审计。

### 4. Checklist 中有三处验收条件需要进一步可执行化

这些问题不阻止纯领域代码开工，但应在对应实现开始前修订：

- `M1-4` 的“查询数不随订单数线性增长”缺少固定测量方法。建议冻结为固定 SQL statement 上限，例如列表请求执行一次 COUNT、一次 page query，以及有界的明细匹配查询；测试通过 query hook 或 repository 计数器断言。
- `M1-5` 的“扫描失败和损坏数据”需要说明测试注入方式。建议允许测试专用 repository fixture、可控 scanner seam，或直接插入绕过应用校验但仍满足/故意破坏约束的数据，避免实现阶段为制造错误而扩大生产接口。
- `M3-6` 的“访问日志区分 canceled/unavailable”需要冻结可观察字段，例如 `outcome=canceled|unavailable`，并明确未提交响应时的 status 记录规则，否则测试容易依赖日志文案。

### 5. 计划复杂度较高，但目前仍可控

计划对幂等 snapshot、错误优先级、route metadata、连接中止和 schema 回退的描述非常细。其好处是降低实现争议；代价是 M2/M3 的代码与测试量会较大。

建议保持现有三个对外里程碑，但在实现任务中继续使用计划已经定义的内部 gate，不再增加新的发布层级。尤其不要把所有 M1 项放在一个超大变更中：先完成 M1-A 的 schema/domain/seed，再完成 M1-B 的查询 HTTP 纵切。

## 建议的开工顺序

1. 完成并勾选 `P0-3a`，保存 v3/v4 发布与恢复模板。
2. 将 P0-4 Evidence 刷新到实际开工 HEAD。
3. 冻结 M1-4、M1-5、M3-6 的可观察测试口径。
4. 开始 M1-A：先建立 `internal/order` 纯领域模型和测试，再实现 migration v3、repository contract、确定性 seed 和 schema-only smoke。
5. M1-A 证据齐全后再注册 M1-B 列表/详情路由。
6. 严格按 checklist 推进后续 gate，不提前创建 v4 幂等表，不提前暴露 create/edit/Action 路由。

## 开发前最小完成条件

以下四项满足后，可以认为阶段三已全面进入实际开发：

- [ ] `P0-3a` 有可执行 Evidence 并已勾选。
- [ ] P0-4 对应实际开工 revision，普通测试、vet、race 均通过。
- [ ] M1-4 查询计数、M1-5 故障注入、M3-6 日志字段的测试口径已写入 plan/checklist。
- [ ] 首个开发变更只进入 M1-A，不注册订单 HTTP 路由，也不创建 `idempotency_keys`。

## 最终判断

新的计划与 checklist 在技术方向上合理，核心契约已经足够稳定，现有代码基线也健康。当前不需要重新规划阶段三，但需要先补齐发布/恢复门禁并刷新证据。完成这些小范围修订后，可以正式启动 M1-A；在此之前仅建议开展不依赖 schema 的领域代码和测试工作。
