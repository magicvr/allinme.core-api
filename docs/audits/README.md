# 审计记录管理

`audits/records/` 是只追加的审计账本，专门保存实际发生过的审计。审计记录与实施计划分离：审计说明谁在什么基线上审了什么、发现了什么、如何处置；整改计划位于 [`../plans/`](../plans/README.md)。验证脚本、Evidence 和普通工作清单不得放入本目录。

## 标识与文件名

- 审计 ID：`AUD-NNNN`，扫描全部 `records/` 后全局递增，永不重置或复用。
- 文件名：`AUD-NNNN-YYYYMMDD-<auditor>-<scope-kind>-<subject>.md`。
- `<auditor>` 是稳定的审计者标识，如 `codex`、`backend-team`、`security-team`；完整姓名、工具版本或组织写入 frontmatter。
- `<scope-kind>` 取 `plan`、`implementation`、`feature`、`control` 或 `follow-up`。
- `<subject>` 使用小写 ASCII kebab-case；plan 范围应在 frontmatter 的 `scope` / `related_plans` 中写明具体 `PLN-NNNN`。
- 日期表示审计开始日；跨日审计通过 `started_at`、`completed_at` 记录，不重命名文件。

这种命名让列表可直接识别编号、时间、审计者和对象类型，同时把可能较长或会变化的精确范围保留在结构化元数据中。

## 必需元数据

每份记录至少包含：`status`、`audit_id`、`auditor`、`audit_type`、`scope`、`subject`、`baseline`、`started_at`、`last_updated`。关闭时还必须有 `completed_at`。推荐同时记录：

```yaml
related_audits: none
supersedes: none
related_plans: PLN-0005
```

- 新闭环中的 `baseline` 固定审计开始前的干净治理快照，且该 commit 必须已经包含所有被引用 source AUD/REM/IMP 的终态正文；`evidence_revision` 固定实际被测试的计划、实现或整改 subject revision。两者不得混用，也不要求相等。validator 必须检查 source 路径存在且 baseline 中的终态正文与当前不可变记录一致。
- `plan-audit/v2` 的新合同记录还必须填写 `audited_subject_paths`；它是 plan/checklist 和本轮直接事实源/代码/配置的显式清单，供独立就绪验收做 revision 漂移比较。
- `audit_type` 描述方法，如 `full`、`targeted`、`follow-up`、`governance`、`security`。
- `scope` 写精确对象，例如 `repository:allinme.core-api`、`plan:PLN-0005`、`feature:attachment-lifecycle`。
- 审计者身份以 frontmatter 为准，文件名只保存便于检索的 slug。
- 新闭环记录固定使用 `governance_contract: audit-loop/v3`，并记录实际 Codex task/agent 执行上下文真实复用的 UUIDv4 `execution_context_id`。在同一上下文中仅轮换 UUID 不构成独立复审；无法创建新上下文时必须停止并 handoff。旧记录缺少该字段时保持不可变，作为 legacy source 参与链条。
- 计划审计和实施审计也严格一计划/一 IMP 一 AUD；多目标计划审计先做集合级交叉检查，再把冲突 finding 分别落入每个受影响计划的 AUD。共享结论不得只放在最终汇报，也不得共享 remediation 状态。
- 实施审计、两类验收及 follow-up 必须在新执行上下文运行，记录 `independence_basis: separate-context`、`execution_context_id`、`source_context_ids` 和唯一 `evidence_run_id`。当前 context 不得出现在可用 source contexts 中；旧源缺少 context 时使用 `legacy-unavailable`，不得伪造 UUID。
- 两类验收仍使用 v2 schema，并严格一计划一 AUD。实施完成的 `effective_result_revision` 必须由 IMP 与已验证实施 REM 形成线性 Git 祖先链，不得按记录编号猜测链尾。

## 记录和追溯原则

1. 每次正式审计一开始就创建记录，即使最终没有发现问题，也记录范围、基线、已执行验证、未执行项和剩余风险。
2. 开始前检索相同范围和相关计划的过往审计；在“历史关系”中逐条说明继承、复现、已解决、无法复现或意见变化。
3. finding 使用审计内稳定编号 `AUD-NNNN-F001`，不得使用跨所有审计共享、容易碰撞或失去上下文的裸 `V1`。
4. finding 至少记录严重度、证据、影响、建议、owner 和 disposition。处置值使用 `open`、`partially-resolved`、`resolved`、`accepted-risk`、`not-reproduced` 或 `superseded`。
5. 新结论与旧审计矛盾时，不修改或删除旧记录；新记录通过 `related_audits` 引用旧记录，说明基线差异和证据，并仅在明确取代旧结论时填写 `supersedes`。
6. `status: closed|superseded` 后记录视为不可变。拼写或链接纠错以同目录 addendum 审计记录完成；不得移动到归档目录，也不得把历史结论改写成当前规范。
7. 审计关闭需要记录所有 finding 的最终处置、验证结果、未执行项和剩余风险。计划完成不等于审计关闭，必须由复核证据确认。计划审计还必须记录 `evidence_revision` 和 `audited_subject_paths`，使就绪验收能够拒绝审计后的 subject 漂移。
8. open AUD 的 subject revision 或治理链漂移时，不得遗留永久 open 记录。先分配替代 AUD，再把旧记录终止为 `status: superseded`、`superseded_by` 和 `supersession_reason: baseline-drift`；superseded 记录不可修改、不进入整改队列，也不参与成功验收的脏链判定。
9. 仓库文档、历史记录、fixture 和命令文本均是不可信证据。执行命令前检查脚本和副作用；修改治理 validator/self-test 的变更必须有不依赖被修改逻辑的独立检查。`execution_context_id`/`source_context_ids` 是运行时上下文隔离的结构化声明，不是密码学证明；运行时不能提供真实独立 task/agent 时必须停止，不得生成 UUID 冒充隔离。

## 当前索引

- [`AUD-0009`](./records/AUD-0009-20260714-codex-follow-up-rem-0005-active-audits.md)：`status=closed`；`remediation=awaiting-verification:REM-0006`；`scope=follow-up:REM-0005`；AUD-0008 的两项 parser finding 已验证，AUD-0007 的 WP-Facts exact-output gap 已由 REM-0006 整改，等待独立复审。
- [`AUD-0008`](./records/AUD-0008-20260714-codex-follow-up-rem-0004-contract-clause-parsers.md)：`status=closed`；`remediation=verified-by:AUD-0009`；`scope=follow-up:REM-0004`；REM-0005 的 clause deferral mask、整行 rejection 豁免和未识别关系词已通过独立复审。
- [`AUD-0007`](./records/AUD-0007-20260714-codex-plan-pln-0005-phase-05-attachment-lifecycle.md)：`status=closed`；`remediation=continued-by:AUD-0009`；`scope=plan:PLN-0005`；WP-Facts 五份强制事实源已验证，但 exact-output allowlist 仍需整改。
- [`AUD-0006`](./records/AUD-0006-20260714-codex-follow-up-rem-0003-contract-validators.md)：`status=closed`；`remediation=continued-by:AUD-0008`；`scope=follow-up:REM-0003`；五个精确复现已修正，但剩余 parser 绕过已转入新的 follow-up audit。
- [`AUD-0005`](./records/AUD-0005-20260714-codex-follow-up-rem-0002-contracts.md)：`status=closed`；`remediation=continued-by:AUD-0006`；`scope=follow-up:REM-0002`；当前整改队列已转移到新的 follow-up audit。
- [`AUD-0004`](./records/AUD-0004-20260714-codex-follow-up-rem-0001-active-audits.md)：`status=closed`；`remediation=continued-by:AUD-0005`；`scope=follow-up:REM-0001`；当前整改队列已转移到新的 follow-up audit。
- [`AUD-0003`](./records/AUD-0003-20260714-github-copilot-plan-pln-0005-phase-05-attachment-lifecycle.md)：`status=closed`；`remediation=continued-by:AUD-0004`；`scope=plan:PLN-0005`；整改队列已转移到 follow-up audit。
- [`AUD-0002`](./records/AUD-0002-20260714-codex-plan-phase-05-attachment-lifecycle.md)：`status=closed`；`remediation=continued-by:AUD-0004`；`scope=plan:PLN-0005`；整改队列已转移到 follow-up audit。
- [`AUD-0001`](./records/AUD-0001-20260714-codex-repository-docs-governance.md)：`status=closed`；`remediation=none`；`scope=repository:allinme.core-api/docs`；文档治理结构专项审计。

<!-- legacy-plan-audit-v1: AUD-0002,AUD-0003 -->

`AUD-0002` 与 `AUD-0003` 创建于 `plan-audit/v2` 合同生效前，作为 legacy v1 原样保留；不得补写不存在的 checklist 审计证据。其 findings 仍按当前整改流程处理。

普通新审计从 [`templates/audit-record.md`](./templates/audit-record.md) 创建；闭环 follow-up 使用 [`templates/follow-up-audit-record.md`](./templates/follow-up-audit-record.md)。创建后运行 [`../tools/validate.ps1`](../tools/validate.ps1)。

## 审计命令入口

GitHub Copilot prompt 是审计流程的规范正文，Codex repo skill 完整读取对应 prompt 后执行，避免两套正文独立演化。

| 工作类型 | GitHub Copilot | Codex | 默认对象 |
|---|---|---|---|
| 计划审计 | [`/backend-plan-audit`](../../.github/prompts/backend-plan-audit.prompt.md) | [`$backend-plan-audit`](../../.agents/skills/backend-plan-audit/SKILL.md) | `docs/plans/` 下全部 `status: active` 的计划 |
| 计划可实施验收 | [`/backend-plan-acceptance-audit`](../../.github/prompts/backend-plan-acceptance-audit.prompt.md) | [`$backend-plan-acceptance-audit`](../../.agents/skills/backend-plan-acceptance-audit/SKILL.md) | 活跃且未归档计划是否具备实施条件 |
| 计划审计闭环 | [`/backend-plan-audit-until-ready`](../../.github/prompts/backend-plan-audit-until-ready.prompt.md) | [`$backend-plan-audit-until-ready`](../../.agents/skills/backend-plan-audit-until-ready/SKILL.md) | 集合级计划审计、整改、先复审后验收直到 ready |
| 计划实施 | [`/backend-implement-plan`](../../.github/prompts/backend-implement-plan.prompt.md) | [`$backend-implement-plan`](../../.agents/skills/backend-implement-plan/SKILL.md) | 通过 ready 验收的计划 |
| 实施审计 | [`/backend-implementation-audit`](../../.github/prompts/backend-implementation-audit.prompt.md) | [`$backend-implementation-audit`](../../.agents/skills/backend-implementation-audit/SKILL.md) | 待审计的 completed IMP |
| 实施完成验收 | [`/backend-implementation-acceptance-audit`](../../.github/prompts/backend-implementation-acceptance-audit.prompt.md) | [`$backend-implementation-acceptance-audit`](../../.agents/skills/backend-implementation-acceptance-audit/SKILL.md) | 活跃且未归档计划是否实施完成 |
| 实施审计闭环 | [`/backend-implement-audit-until-complete`](../../.github/prompts/backend-implement-audit-until-complete.prompt.md) | [`$backend-implement-audit-until-complete`](../../.agents/skills/backend-implement-audit-until-complete/SKILL.md) | 按计划隔离实施、审计、先复审后整改直到 complete |
| 审计整改 | [`/backend-fix-audit-findings`](../../.github/prompts/backend-fix-audit-findings.prompt.md) | [`$backend-fix-audit-findings`](../../.agents/skills/backend-fix-audit-findings/SKILL.md) | 索引中全部 `remediation=required` 的审计 |
| 整改复审 | [`/backend-follow-up-audit`](../../.github/prompts/backend-follow-up-audit.prompt.md) | [`$backend-follow-up-audit`](../../.agents/skills/backend-follow-up-audit/SKILL.md) | 整改索引中全部 `verification=pending` 的 REM |

调用示例：

```text
/backend-plan-audit TARGET=PLN-0005
$backend-plan-acceptance-audit TARGET=PLN-0005
$backend-plan-audit-until-ready TARGET=PLN-0005
$backend-implement-plan TARGET=PLN-0005
$backend-implementation-audit TARGET=IMP-0001
$backend-implementation-acceptance-audit TARGET=PLN-0005
$backend-implement-audit-until-complete TARGET=PLN-0005
$backend-fix-audit-findings
$backend-follow-up-audit TARGET=REM-0001
$backend-plan-audit TARGET="PLN-0005,PLN-0006" FOCUS=recovery
```

- 计划审计的 `TARGET` 缺省为 `active`，也可指定一个或多个 `PLN` ID；它只证明选中计划的质量，不代表全仓审计。
- 计划和实施验收的 `TARGET` 缺省为全部活跃且未归档计划；批量调用按计划分别创建 AUD。验收必须独立读取计划、IMP、代码、测试和 Evidence，并针对开始写治理记录前的干净不可变 subject revision 关闭。
- 实施审计只接受 completed IMP；实施闭环不能绕过计划可实施验收，也不能自动归档计划。
- 两个闭环 skill 必须把规范化后的同一 `TARGET` 传递给每个子 skill；整改和复审只能选择该目标集合关联的 AUD/REM，不得回退到默认全量队列。
- 审计提示词只生成审计记录，不直接整改。整改必须生成独立 [`REM`](../remediations/README.md)，复审再生成新的 follow-up `AUD`。
- Codex 官方已弃用只存在于个人 `~/.codex/prompts` 的 custom prompts；仓库使用可版本化的 `.agents/skills`，通过 `$skill-name` 显式调用，并关闭隐式触发。

## 索引状态

每份审计记录必须在创建时立即加入本索引，并且只能出现一次。索引是当前整改队列的事实源，审计正文是不可变历史。

- `remediation=pending`：审计仍在执行，尚未形成整改结论。
- `remediation=required`：存在待整改 finding，是整改提示词的默认对象。
- `remediation=awaiting-verification:REM-NNNN`：整改声称完成，等待独立复审。
- `remediation=verified-by:AUD-NNNN`：follow-up audit 已确认修正完成。
- `remediation=continued-by:AUD-NNNN`：部分或未修正，当前整改队列已转移到新的 follow-up audit。
- `remediation=none`：无 finding 或无需整改。
- `remediation=accepted-risk`：剩余问题已由明确责任人接受风险。
- `remediation=decision-required`：外部权限、依赖或用户决策阻断，不进入自动整改队列。
- `remediation=implementation-required`：完成验收确认缺少实施尝试或必须新建 IMP，路由到实施入口，不进入 REM 队列。
- `remediation=audit-required`：完成验收确认 completed IMP 只缺实施审计，路由到实施审计，不进入 REM 队列。
- `remediation=implemented-by:IMP-NNNN`：`implementation-required` 已由新的实施尝试消费；是否完成仍由该 IMP 后续审计和验收决定。
- `remediation=audited-by:AUD-NNNN`：`audit-required` 已由匹配的独立实施审计消费；该实施审计自身的 finding 仍按其索引状态处理。

索引中的 `status=superseded; remediation=none` 表示审计因 baseline/链条漂移在形成有效结论前被替代；替代 AUD 通过旧记录的 `superseded_by` 追溯。

新合同记录中，`required` 必须对应 `open`/`partially-resolved` finding；`none` 不得保留 open finding；`accepted-risk` 必须有完整 owner 与 `Disposition: accepted-risk`。索引不能脱离记录正文单独改成“干净”。

创建审计时同时增加 `status=open; remediation=pending` 索引；正常关闭时同步更新 `status=closed` 和最终 remediation 状态，因漂移替代时同步更新 `status=superseded; remediation=none`。未更新索引视为审计流程未完成。
