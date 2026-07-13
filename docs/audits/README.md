# 审计记录管理

`audits/records/` 是只追加的审计账本，专门保存实际发生过的审计。审计记录与实施计划分离：审计说明谁在什么基线上审了什么、发现了什么、如何处置；整改计划位于 [`../plans/`](../plans/README.md)。验证脚本、Evidence 和普通工作清单不得放入本目录。

## 标识与文件名

- 审计 ID：`AUD-NNNN`，扫描全部 `records/` 后全局递增，永不重置或复用。
- 文件名：`AUD-NNNN-YYYYMMDD-<auditor>-<scope-kind>-<subject>.md`。
- `<auditor>` 是稳定的审计者标识，如 `codex`、`backend-team`、`security-team`；完整姓名、工具版本或组织写入 frontmatter。
- `<scope-kind>` 取 `repository`、`plan`、`feature`、`control` 或 `follow-up`。
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

- `baseline` 必须固定到 commit SHA、tag、artifact digest 或其他不可变对象，并说明工作树是否干净。
- `audit_type` 描述方法，如 `full`、`targeted`、`follow-up`、`governance`、`security`。
- `scope` 写精确对象，例如 `repository:allinme.core-api`、`plan:PLN-0005`、`feature:attachment-lifecycle`。
- 审计者身份以 frontmatter 为准，文件名只保存便于检索的 slug。

## 记录和追溯原则

1. 每次正式审计一开始就创建记录，即使最终没有发现问题，也记录范围、基线、已执行验证、未执行项和剩余风险。
2. 开始前检索相同范围和相关计划的过往审计；在“历史关系”中逐条说明继承、复现、已解决、无法复现或意见变化。
3. finding 使用审计内稳定编号 `AUD-NNNN-F001`，不得使用跨所有审计共享、容易碰撞或失去上下文的裸 `V1`。
4. finding 至少记录严重度、证据、影响、建议、owner 和 disposition。处置值使用 `open`、`resolved`、`accepted-risk`、`not-reproduced` 或 `superseded`。
5. 新结论与旧审计矛盾时，不修改或删除旧记录；新记录通过 `related_audits` 引用旧记录，说明基线差异和证据，并仅在明确取代旧结论时填写 `supersedes`。
6. `status: closed` 后记录视为不可变。拼写或链接纠错以同目录 addendum 审计记录完成；不得移动到归档目录，也不得把历史结论改写成当前规范。
7. 审计关闭需要记录所有 finding 的最终处置、验证结果、未执行项和剩余风险。计划完成不等于审计关闭，必须由复核证据确认。

## 当前索引

- [`AUD-0001`](./records/AUD-0001-20260714-codex-repository-docs-governance.md)：文档治理结构专项审计。

新审计从 [`templates/audit-record.md`](./templates/audit-record.md) 创建，并运行 [`../tools/validate.ps1`](../tools/validate.ps1)。

## 审计命令入口

GitHub Copilot prompt 是审计流程的规范正文，Codex repo skill 完整读取对应 prompt 后执行，避免两套正文独立演化。

| 审计类型 | GitHub Copilot | Codex | 默认对象 |
|---|---|---|---|
| 全仓全量审计 | [`/backend-full-audit`](../../.github/prompts/backend-full-audit.prompt.md) | [`$backend-full-audit`](../../.agents/skills/backend-full-audit/SKILL.md) | 整个 `allinme.core-api`，不可缩小范围 |
| 计划审计 | [`/backend-plan-audit`](../../.github/prompts/backend-plan-audit.prompt.md) | [`$backend-plan-audit`](../../.agents/skills/backend-plan-audit/SKILL.md) | `docs/plans/` 下全部 `status: active` 的计划 |

调用示例：

```text
/backend-full-audit FOCUS=security
/backend-plan-audit TARGET=PLN-0005
$backend-full-audit AUDITOR=codex MODE=audit-only
$backend-plan-audit TARGET="PLN-0005,PLN-0006" FOCUS=recovery
```

- 全量审计的 `FOCUS` 只能增加检查深度，不能把范围缩小为 plan、feature、diff、目录或 PR。
- 计划审计的 `TARGET` 缺省为 `active`，也可指定一个或多个 `PLN` ID；它只证明选中计划的质量，不代表全仓审计。
- 两类审计的 `MODE` 缺省为 `audit-only`。只有显式 `MODE=remediate` 且审计结果已经记录和汇报后，才进入整改。
- Codex 官方已弃用只存在于个人 `~/.codex/prompts` 的 custom prompts；仓库使用可版本化的 `.agents/skills`，通过 `$skill-name` 显式调用，并关闭隐式触发。
