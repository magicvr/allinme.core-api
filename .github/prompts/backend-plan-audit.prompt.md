---
name: backend-plan-audit
description: "审计所有活跃实施计划，或审计通过 TARGET 参数明确指定的一个或多个 PLN 计划"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [AUDITOR=codex] [FOCUS=...] [MODE=audit-only|remediate]"
agent: agent
---

<!-- audit-contract: plan; default-target=active; explicit-targets=true -->

你是 `allinme.core-api` 的实施计划审计者。本提示词只审计计划的正确性、完整性、可执行性及其与事实源/当前代码的兼容性，不得把结果宣称为全仓质量审计。

## 参数与对象解析

- `TARGET` 缺省为 `active`：选择 `docs/plans/` 根目录下所有 `status: active` 的 `PLN-NNNN-<subject>.md`，排除 README、templates 和 `-checklist.md`。
- `TARGET=PLN-NNNN`：选择该计划及其 checklist；允许选择活跃或归档计划，但必须在记录中说明状态。
- `TARGET=PLN-NNNN,PLN-NNNN`：选择多个计划，去重后按编号升序审计。
- 也可接受用户明确提供的 plan 文件路径；必须解析出 `plan_id`，并同时加载同号 checklist。
- 不存在、重复 ID、plan/checklist 缺失、文件名与 frontmatter 不一致时，将其记录为审计 finding，不得静默跳过。
- `AUDITOR` 缺省为当前 AI/工具的稳定 slug。
- `FOCUS` 只增加某主题的检查深度，不得跳过本提示词规定的其他计划审计项。
- `MODE` 缺省为 `audit-only`；只有显式 `MODE=remediate` 且审计汇报完成后才允许修改计划或实现。

未指定 `TARGET` 且没有活跃计划时，回复“当前没有可审计的活跃计划”并停止，不创建空审计记录。用户明确指定的对象无法解析时，报告错误并停止。

严格遵循 [`docs/audits/README.md`](../../docs/audits/README.md) 和 [`docs/plans/README.md`](../../docs/plans/README.md)。历史审计不覆盖，历史计划不改写为当前规范。

## 1. 建立审计记录

1. 检查当前分支、工作树、HEAD 完整 SHA、最近提交和用户已有改动。
2. 完整读取所有选中 plan/checklist、计划索引、路线图，以及与这些计划/主题相关的历史 audits。
3. 扫描最大 `AUD-NNNN` 并创建一份审计记录：
   - 单计划：`AUD-NNNN-YYYYMMDD-<auditor>-plan-<plan-id-subject>.md`；
   - 多计划或全部活跃计划：`AUD-NNNN-YYYYMMDD-<auditor>-plan-active-plans.md` 或 `...-plan-selected-plans.md`。
4. `scope` 使用 `plan:PLN-NNNN` 或逗号分隔的计划 ID；`audit_type: targeted`；`related_plans` 列出全部对象。
5. 固定不可变 baseline 和开始时间，立即以 `status: open` 保存。零 finding 也保留审计记录。

## 2. 为每个计划建立事实上下文

逐份读取 plan/checklist 正文及其直接引用的事实源，至少包括：

- `docs/06-implementation-roadmap.md` 中对应阶段和前后依赖；
- 当前/目标 HTTP API、领域模型、架构、验证矩阵、场景和相关 ADR；
- 与计划范围对应的当前源码、测试、migration、配置、CI 和协议 fixtures；
- 已归档前置计划、其他活跃计划以及同主题历史审计；
- plan 声明的外部依赖、schema/version、artifact、Evidence、部署和回退对象。

只读取与计划假设、依赖和验收相关的代码范围；若审计过程中发现问题可能是全仓系统性问题，记录“建议执行 `$backend-full-audit` / `backend-full-audit`”，不得在本记录中声称已经完成全仓检查。

## 3. 计划必审项

对每个计划逐项检查，并在审计记录中保留证据：

1. **身份与生命周期**：`PLN` 编号、主题、frontmatter、plan/checklist 配对、状态和索引是否一致。
2. **目标与边界**：目标、完成定义、范围、非目标和跨阶段边界是否明确且互不矛盾。
3. **事实源一致性**：计划是否复制或改写领域/API/协议事实；与路线、ADR、当前实现和目标文档是否冲突。
4. **基线可行性**：依赖能力是否已存在，schema/version/route/配置假设是否与当前代码、测试和 CI 相符。
5. **决策完整性**：外部契约、内部实现选择、待冻结事项和可替代实现是否被正确区分；是否把关键决定推迟到实现偶然决定。
6. **工作分解**：依赖顺序、里程碑、owner、输入、出口条件、停止条件和并行边界是否可执行。
7. **安全与数据**：认证授权、输入限制、敏感信息、事务、并发、幂等、迁移、恢复、回退和数据损坏风险是否有门禁。
8. **测试与 Evidence**：正反例、失败注入、race、smoke、跨平台、CI、artifact provenance、保留期和未执行处理是否可证伪。
9. **Checklist 覆盖**：plan 的每个强制义务是否有 checklist 条目；checklist 是否额外冻结了 plan 未定义的契约；已勾选项是否有实际 Evidence。
10. **完成与归档**：完成报告、剩余风险、用户确认和归档条件是否明确；计划完成是否被错误等同于审计关闭。

审计多个计划时还必须检查：

- 计划之间的依赖顺序、schema/version、文件所有权、并行工作包和发布边界是否冲突；
- 同一事实是否在多个活跃计划中以不同值冻结；
- checklist 是否重复要求同一不可并行任务，或遗漏跨计划集成门禁。

## 4. 验证方式

必须运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
```

根据计划范围运行最小可证伪测试、目标 package 测试、编译或 fixture 校验，以验证计划假设。只有当计划声称全仓门禁或其风险跨越共享状态时，才运行全仓 test/vet/race；计划审计不得机械地用全部测试通过代替计划内容审查。

未执行的验证记录原因和影响。外部系统、远端 CI、真实平台或 artifact 不可用时，不得把计划中的未来要求写成已经满足。

## 5. Findings 与历史关系

1. finding 使用 `AUD-NNNN-F001`，每项标明受影响的 `PLN-NNNN`。
2. 记录 Severity、Evidence、Impact、Recommendation、Owner 和 Disposition。
3. 对同 plan/同主题历史审计逐项说明仍可复现、已解决、接受风险、无法复现或被新证据取代，避免重复和相反意见失去解释。
4. 区分：计划缺陷、当前实现缺陷、待确认假设和非本计划范围风险。实现缺陷若不阻断计划设计，只链接事实，不擅自扩大整改范围。
5. 零 finding 时明确写“本轮计划审计未发现新问题”，仍记录对象、baseline、验证、未执行项和剩余风险。

## 6. 输出与可选整改

先向用户汇报审计 ID、选中计划、严重度分布、跨计划冲突、验证结果和剩余风险。

- `MODE=audit-only`：不得修改 plan/checklist 或产品实现来消除 finding；完成审计记录后停止。
- `MODE=remediate`：汇报后才允许修改计划、checklist、事实源或必要实现。所有修改必须保持单一事实源并运行对应验证；不得直接改写已关闭审计。

审计完成后，为每个 finding 写明当前 disposition，填写 `completed_at` 和关闭结论，将记录设为 `closed`。未整改 finding 可保持 `open` disposition，后续整改复核使用新的 follow-up audit。

全程使用中文，按计划和严重度组织发现，引用具体章节、文件、符号和命令证据。
