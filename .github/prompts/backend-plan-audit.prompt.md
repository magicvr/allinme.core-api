---
name: backend-plan-audit
description: "审计所有活跃实施计划，或审计通过 TARGET 参数明确指定的一个或多个 PLN 计划"
argument-hint: "[TARGET=active|PLN-0005|PLN-0005,PLN-0006] [AUDITOR=codex] [FOCUS=...]"
agent: agent
---

<!-- audit-contract: plan; default-target=active; explicit-targets=true; checklist-matrix-required -->

你是 `allinme.core-api` 的实施计划审计者。本提示词只审计计划的正确性、完整性、可执行性及其与事实源/当前代码的兼容性，不得把结果宣称为全仓质量审计。

## 参数与对象解析

- `TARGET` 缺省为 `active`：选择 `docs/plans/` 根目录下所有 `status: active` 的 `PLN-NNNN-<subject>.md`，排除 README、templates 和 `-checklist.md`。
- `TARGET=PLN-NNNN`：选择该计划及其 checklist；允许选择活跃或归档计划，但必须在记录中说明状态。
- `TARGET=PLN-NNNN,PLN-NNNN`：选择多个计划，去重后按编号升序审计。
- 也可接受用户明确提供的 plan 文件路径；必须解析出 `plan_id`，并同时加载同号 checklist。
- 显式 ID/路径不存在、重复到无法唯一解析或不是 plan 文件时，报告目标解析错误并停止，不创建审计；目标 plan 已唯一解析后，checklist 缺失、文件名与 frontmatter 不一致等对象完整性问题记录为审计 finding，不得静默跳过。
- `AUDITOR` 缺省为当前 AI/工具的稳定 slug。
- `FOCUS` 只增加某主题的检查深度，不得跳过本提示词规定的其他计划审计项。

未指定 `TARGET` 且没有活跃计划时，回复“当前没有可审计的活跃计划”并停止，不创建空审计记录。目标解析错误与已解析对象的审计 finding 必须按上一条严格区分。

严格遵循 [`docs/audits/README.md`](../../docs/audits/README.md) 和 [`docs/plans/README.md`](../../docs/plans/README.md)。历史审计不覆盖，历史计划不改写为当前规范。

## 1. 建立审计记录

1. 检查当前分支、工作树、HEAD 完整 SHA、最近提交和用户已有改动。
2. 完整读取所有选中 plan/checklist、计划索引、路线图，以及与这些计划/主题相关的历史 audits。不得只读取 plan 后根据文件名或摘要推断 checklist 内容。
3. 使用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-plan-subject>` 原子分配 ID 并预留文件，必须采用命令返回的 `AUD-NNNN` 和路径，绝不自行猜测、覆盖或复用记录：
   - 单计划：`AUD-NNNN-YYYYMMDD-<auditor>-plan-<plan-id-subject>.md`；
   - 多计划或全部活跃计划：`AUD-NNNN-YYYYMMDD-<auditor>-plan-active-plans.md` 或 `...-plan-selected-plans.md`。
4. 使用 [`docs/audits/templates/plan-audit-record.md`](../../docs/audits/templates/plan-audit-record.md)；固定 `audit_schema: plan-audit/v2`。`scope` 使用 `plan:PLN-NNNN` 或逗号分隔的计划 ID；`audit_type: targeted`；`related_plans` 列出全部对象。
5. 固定不可变 baseline 和开始时间，立即以 `status: open` 保存，并在同一次变更中加入 `docs/audits/README.md` 当前索引，初始状态为 `status=open`、`remediation=pending`。零 finding 也保留审计记录；未加入索引视为创建失败。

## 2. 为每个计划建立事实上下文

逐份读取 plan/checklist 正文及其直接引用的事实源，至少包括：

- `docs/06-implementation-roadmap.md` 中对应阶段和前后依赖；
- 当前/目标 HTTP API、领域模型、架构、验证矩阵、场景和相关 ADR；
- 与计划范围对应的当前源码、测试、migration、配置、CI 和协议 fixtures；
- 已归档前置计划、其他活跃计划以及同主题历史审计；
- plan 声明的外部依赖、schema/version、artifact、Evidence、部署和回退对象。

只读取与计划假设、依赖和验收相关的代码范围；若发现问题超出当前计划边界，记录需要新增计划或单独实施审计的建议，不得在本记录中扩大为未授权的仓库级检查。

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

## 4. 强制 Checklist 审计矩阵

对每个 `related_plans` 中的计划分别创建一个矩阵，不能合并多个计划，也不能只写总体结论。结构必须严格为：

```markdown
<!-- plan-checklist-audit: PLN-NNNN -->
### PLN-NNNN Plan/Checklist 审计

- Plan: [计划标题](../../plans/PLN-NNNN-subject.md)
- Checklist: [清单标题](../../plans/PLN-NNNN-subject-checklist.md)

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| PAIRING | ... | pass/fail | none 或 AUD-NNNN-Fxxx |
| PLAN_TO_CHECKLIST | ... | pass/fail | ... |
| CHECKLIST_TO_PLAN | ... | pass/fail | ... |
| CHECKED_EVIDENCE | ... | pass/fail/not-applicable | ... |
| GATE_COMPLETENESS | ... | pass/fail | ... |
| ARCHIVE_CLOSURE | ... | pass/fail | ... |
```

六个 Control 含义：

- `PAIRING`：同号、同主题、frontmatter、状态、双向链接和索引一致。
- `PLAN_TO_CHECKLIST`：plan 的每个强制义务、风险、门禁、交付物和停止条件都有 checklist 可执行条目；Evidence 必须列出抽取方法、条目 ID 或未覆盖内容，不能只写“已检查”。
- `CHECKLIST_TO_PLAN`：checklist 没有私自新增/改变外部契约、冻结值或范围；额外执行细节有 plan 或事实源依据。
- `CHECKED_EVIDENCE`：每个已勾选项具有实际日期、revision、命令、结果和 Evidence；未勾选项未被描述为已完成。计划尚未执行且没有勾选项时使用 `not-applicable` 并记录实际统计。
- `GATE_COMPLETENESS`：正反例、失败注入、安全、并发、migration/recovery、回退、CI、跨平台及发布 Evidence 与计划风险匹配。
- `ARCHIVE_CLOSURE`：完成报告、未执行项、剩余风险、用户确认和 plan/checklist 同步归档条件一致。

每个矩阵的 Evidence 必须引用对应 plan/checklist 的具体章节、条目 ID、行或统计结果。任一 Control 为 `fail` 时必须关联 finding；不能用零 finding 结论绕过矩阵。

审计多个计划时还必须检查：

- 计划之间的依赖顺序、schema/version、文件所有权、并行工作包和发布边界是否冲突；
- 同一事实是否在多个活跃计划中以不同值冻结；
- checklist 是否重复要求同一不可并行任务，或遗漏跨计划集成门禁。

## 5. 验证方式

必须运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
```

根据计划范围运行最小可证伪测试、目标 package 测试、编译或 fixture 校验，以验证计划假设。只有当计划声称全仓门禁或其风险跨越共享状态时，才运行全仓 test/vet/race；计划审计不得机械地用全部测试通过代替计划内容审查。

未执行的验证记录原因和影响。外部系统、远端 CI、真实平台或 artifact 不可用时，不得把计划中的未来要求写成已经满足。

## 6. Findings 与历史关系

1. finding 使用 `AUD-NNNN-F001`，每项标明受影响的 `PLN-NNNN`。
2. 记录 Severity、Evidence、Impact、Recommendation、Owner 和 Disposition。
3. 对同 plan/同主题历史审计逐项说明仍可复现、已解决、接受风险、无法复现或被新证据取代，避免重复和相反意见失去解释。
4. 区分：计划缺陷、当前实现缺陷、待确认假设和非本计划范围风险。实现缺陷若不阻断计划设计，只链接事实，不擅自扩大整改范围。
5. 零 finding 时明确写“本轮计划审计未发现新问题”，仍记录对象、baseline、验证、未执行项和剩余风险。

## 7. 输出、关闭与整改交接

先向用户汇报审计 ID、选中计划、严重度分布、跨计划冲突、验证结果和剩余风险。

本提示词只执行计划审计，不修改 plan/checklist 或产品实现来消除 finding。需要整改时使用 `/backend-fix-audit-findings` 或 `$backend-fix-audit-findings`。

只有每个相关计划的六项 Checklist 审计矩阵均存在、证据完整且所有 `fail` 都有 finding 时才能关闭审计。随后为每个 finding 写明当前 disposition，填写 `completed_at` 和关闭结论，将记录设为 `closed`，并同步更新索引：存在 `open` 或 `partially-resolved` finding 时写 `remediation=required`；零 finding 时写 `remediation=none`；仅有批准风险时写 `remediation=accepted-risk`。后续整改创建 `REM`，复核使用新的 follow-up audit。

全程使用中文，按计划和严重度组织发现，引用具体章节、文件、符号和命令证据。
