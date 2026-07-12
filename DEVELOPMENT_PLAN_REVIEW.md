# 阶段三开发计划与 Checklist 审视建议

审视日期：2026-07-12  
审视范围：[`docs/audit/0003-2026-07-12-plan.md`](docs/audit/0003-2026-07-12-plan.md)、[`docs/audit/0003-2026-07-12-checklist.md`](docs/audit/0003-2026-07-12-checklist.md)，并交叉核对架构、领域模型、路线图、目标 API、验证矩阵和当前代码结构。

## 结论

计划的业务边界、状态机、权限、金额、幂等、并发、CORS 和文档迁移意识都比较完整，三个端到端里程碑的方向也合理。但当前版本把外部契约、内部设计、测试实现细节、人员治理和发布流程同时设为硬门禁，形成了 310 行计划和 69 项 checklist。它可以作为设计资料，但不建议原样作为执行清单开工。

建议结论：**调整后开工**。先修复 3 个阻断问题，再压缩门禁和明确责任边界；其余技术设计可以保留。

## 阻断问题

### 1. M2-A 与已冻结的创建幂等契约冲突

计划 §3 明确 `POST /api/v1/orders` 必须提供 `Idempotency-Key`，首次与重放都要使用不可变 snapshot。可是 M2-A 要求真实 create/edit 正常路径可用，M2-B 才完成 snapshot、replay、冲突和并发竞争，并把 M2-A 作为可回滚的正常 create/edit 基线。

这会产生一个无法同时满足的状态：M2-A 若注册创建路由，就无法满足已经冻结的幂等契约；若不注册路由，又无法满足“真实创建编辑”的 gate。

建议二选一，优先采用方案 A：

- 方案 A：M2-A 只完成未注册到真实 app 的 command、金额、事务和 edit 能力；M2-B 一次性注册 create，并同时交付完整幂等语义。
- 方案 B：把最小 snapshot、replay 和 conflict 移入 M2-A；M2-B 只保留并发竞争、损坏记录和 BUSY/LOCKED 等强化证据。

同步修改 plan §8、checklist M2-13、M2-A、M2-B，避免内部 gate 暴露不完整的外部契约。

### 2. “独立回滚”没有可执行的数据库策略

路线图和计划要求 M1/M2/M3 均可独立回滚，M1-A 甚至把回滚点写为 v2 schema。当前 migration 机制只有连续前向迁移，readiness 会检查数据库是否为当前最新 schema；没有 down migration、兼容窗口或备份恢复流程。部署过 v3 schema 后直接回滚到阶段二二进制，很可能因“数据库版本高于支持版本”而不可用。

建议先明确“回滚”的实际含义：

- 推荐：采用 **roll-forward schema + backward-compatible binary**。数据库迁移只前进，代码回滚要求上一版本能容忍新增表/索引；readiness 不应因无害的更高 schema 版本直接拒绝，或必须定义兼容版本区间。
- 如果坚持回滚到 v2 schema：必须提供并验证备份、停写、恢复和数据丢失边界，不能只在 checklist 中写“回滚点”。
- 将 schema 按里程碑拆分：M1 仅创建 `orders`/`order_items`，M2 再创建 `idempotency_keys`。这样更符合纵切边界，但仍需解决二进制与 schema 的兼容策略。

在策略冻结前，应把“可独立回滚”改为“可独立关闭路由/功能并保持数据库可用”，避免作出当前实现无法证明的承诺。

### 3. `go test -run` 不能单独充当硬门禁

plan §8 的聚焦命令依赖测试名称正则。Go 在 `-run` 没有命中任何测试时仍可能返回 exit 0，因此重命名测试、尚未创建测试或正则漂移都可能让 gate 假通过。这与 checklist “证据缺失直接阻断”的原则冲突。

建议：

- 硬门禁优先执行完整目标包，例如 `go test ./internal/order ./internal/store -count=1`。
- 若必须使用 `-run`，用 `go test -json` 或脚本同时断言至少命中一个预期测试，并记录实际测试数。
- 聚焦正则只作为开发加速命令，不作为唯一发布证据。

## 重要调整

### 4. 事务归属需要与架构文档统一

架构文档规定“业务事务和状态转换由用例层控制”，阶段三计划又把 transaction ownership 固定给 repository。单聚合订单操作可以由 repository 封装事务，但后续退款、附件绑定会需要跨 repository 的原子操作，若现在把事务所有权设为不可变硬边界，阶段四/五很可能再次重构。

建议把规则改为：

> 用例层拥有业务事务语义；store 提供事务执行器或聚合级原子 repository 方法。阶段三单订单操作允许 repository 内部事务，但不得把该实现提升为跨阶段不可变架构契约。

若团队确认长期采用 repository-owned transaction，应补一条 ADR，解释后续跨订单/退款/附件事务如何组合。

### 5. Checklist 过度重复，降低真实完成度的可见性

当前 69 项中存在多层重复：具体条目、`M*-A/B` gate、`M*-14/16` 聚焦门禁、R8/R9 全量门禁反复验证相同结果。大量 evidence 占位也会让维护者更关注勾选和复制命令，而不是交付可观察能力。

建议压缩为约 25-35 项：

- 每个里程碑保留 5-8 个能力结果。
- 每个里程碑只保留 1 个集成 gate 和 1 个质量 gate。
- 通用 test/vet/race/doc 检查只在发布门禁保留一次；里程碑只记录增量包测试。
- 测试 helper、barrier、executor、具体测试名放到测试代码或完成报告，不放入长期 checklist。
- P0-6 的姓名、替补、日期和升级路径移到独立执行元数据；它是项目管理信息，不是产品正确性证据。

### 6. M1 的范围可进一步收紧

M1 名为只读纵切，却同时创建幂等表、冻结写事务边界并承担后续 CORS route metadata 的完整设计。建议 M1 只交付：

- 订单与明细 schema、seed；
- 领域读取模型和 capability；
- 列表/详情 repository、service、HTTP；
- 共享 DB app wiring；
- 能支撑后续写路由和 CORS 的最小 route metadata。

幂等表、snapshot DTO 和写错误分类放到 M2。这样 M1 更容易独立评审，也能减少尚未使用 schema 的提前承诺。

### 7. 部分“唯一事实源”仍存在时序漂移风险

计划说明 CORS 在 M3-B 启用，但验证矩阵当前写的是“阶段三 M3 CORS 集成通过时”；计划与 checklist 又要求 M3-B。虽然含义接近，自动审查和执行人员可能据此在不同时间启用矩阵。

建议所有跨文档 gate 使用同一个稳定标识，例如 `M3-B`，不要混用 `M3`、`M3 CORS 集成` 和“阶段三完成”。

## 保留项

以下设计合理，建议保留：

- 阶段三明确排除退款、附件、看板、页面 YAML 和外部支付/物流。
- 金额使用 `int64` 最小货币单位并执行 checked arithmetic。
- version 与状态冲突按存在性、version、状态稳定分类。
- 创建幂等保存不可变 snapshot，重放不读取当前订单。
- 查询 COUNT 与 page 使用同一 SQLite snapshot，列表避免 N+1。
- route metadata 同时支撑 method、Allow、HEAD、known path 和 CORS。
- CORS 配置在打开数据库和监听端口前校验。
- 真实登录 token、重启、reset、并发和连接取消都纳入自动化证据。
- 页面能力明确留到阶段六，避免用 API smoke 冒充页面联调完成。

## 建议后的里程碑

### M1：只读订单 API

交付 orders/order_items migration、确定性 seed、列表/详情、四角色真实 token、稳定查询与 capability。完成后订单查询验证矩阵启用。

### M2：完整草稿写入与幂等

内部先完成 command、金额和事务；对真实 app 注册 create 时必须同时具备 `Idempotency-Key`、snapshot、replay 和 conflict。edit 与 version 一并交付。并发与 BUSY/LOCKED 作为 M2 完成门禁，不再暴露不完整的 M2-A 外部接口。

### M3：履约与浏览器 transport

先完成五个 Action 和并发冲突，再完成 CORS、连接取消、日志、重启/reset 和文档迁移。履约矩阵与 CORS 矩阵分别按 `M3-A`、`M3-B` 启用。

## 开工前最小修改清单

1. 修正 M2-A/M2-B 的创建接口与幂等交付顺序。
2. 定义数据库迁移后的实际回滚策略，并修改“回滚到 v2 schema”的表述。
3. 将 `go test -run` 从唯一硬门禁降级，增加零测试命中保护。
4. 统一架构文档与计划中的事务所有权表述，必要时新增 ADR。
5. 将 checklist 从 69 项压缩到能力结果和发布证据，删除重复 gate。
6. 统一验证矩阵的 `M1-B`、`M2-B`、`M3-A`、`M3-B` 标识。

完成以上六项后，阶段三计划具备可执行性；其余细节可以在实现过程中按现有契约推进。
