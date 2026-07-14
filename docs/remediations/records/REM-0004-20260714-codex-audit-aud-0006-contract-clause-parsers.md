---
status: completed
remediation_id: REM-0004
implementer: codex
scope: audit:AUD-0006
source_audits: AUD-0006
source_findings: AUD-0006-F001, AUD-0006-F002
baseline: git:596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1; worktree:dirty (pre-existing AUD-0007 record and audit index entry)
started_at: 2026-07-14T12:48:25+08:00
completed_at: 2026-07-14T12:55:04+08:00
last_updated: 2026-07-14
related_plans: PLN-0005
---

# AUD-0006 条款级 contract parser 整改

## 对象与边界

本记录整改 `AUD-0006` 中两项 `partially-resolved` finding。范围限定为 `PLN-0005` 的 P0 deployment Evidence 与 tracked dependency DAG 治理门禁及其 self-test；不修改已关闭的 `AUD-0006` 或历史 REM，不实现阶段五产品代码，也不自行把 source finding 标记为 resolved。开始整改前工作树已有未提交的 `AUD-0007` 记录及审计索引变更，本整改保留且不改写该记录。

## Finding 整改矩阵

| Source finding | Root cause | Planned change | Validation | Result |
|---|---|---|---|---|
| `AUD-0006-F001` | P0 deployment scanner 逐行判定，并以整行 deferral 覆盖 obligation；Markdown continuation 又不继承父 P0 item，且禁止类别没有从结构化 contract 驱动扫描。 | 解析结构化 contract 的 `forbidden_p0_live_evidence`；按完整 P0 checklist item block 与 plan clause 扫描；逐 clause 判定 deferral/obligation，并让 continuation 继承 P0 作用域。 | 增加合法 P0-22 deferral 后同行追加 live obligation 与 P0-20 多行 continuation 两个负向 fixture；保留合法 contract fixture 正向用例。 | completed locally; pending follow-up audit |
| `AUD-0006-F002` | dependency prose parser 只消费首个 prerequisite，不识别 pronoun negation 与 reverse `after`，也不验证一个关系 tail 中的全部 `WP-*` token。 | 按关系 clause 解析完整 prerequisite 列表；增加 pronoun negation 与 reverse ordering；对出现依赖语义但未完整消费的多包 clause fail-closed。 | 增加 negative-pronoun、conjoined unknown prerequisite 与 reverse `after` 三个负向 fixture；保留合法 tracked DAG 正向用例。 | completed locally; pending follow-up audit |

## 实际变更

- `docs/tools/validate.ps1`：从 `phase5-p0-deployment-evidence-contract` 解析 `forbidden_p0_live_evidence`，按 contract 类别选择 live Evidence 语义模式；对 checklist 按完整 P0 item block 聚合 continuation，对 plan/checklist 按 clause 分隔 deferral 与 obligation，不再让同行合法 deferral 掩盖后续矛盾义务。
- `docs/tools/validate.ps1`：dependency parser 完整消费 `depends on` 后的所有 `WP-*` prerequisite，识别 pronoun negation 与 `after/follows` 反向顺序，对含依赖信号但未完整消费的多包句子 fail-closed；保留明确的“validator 必须拒绝”规范句豁免。
- `docs/tools/validate.tests.ps1`：增加 `AUD-0006` 复现的五类负向 fixture：P0 deferral 后追加 obligation、P0 continuation obligation、pronoun negation、conjoined unknown dependency 和 reverse `after`。
- 实际 revision：`596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1` 上的未提交整改 diff；未修改任何 closed AUD 或 closed REM 正文，并保留了整改开始前已存在的 `AUD-0007` 工作树变更。

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；合法 fixture 被接受，五类 `AUD-0006` 绕过语义及既有 DAG/profile 删除和附加矛盾均被拒绝。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过；验证 54 个 Markdown 文件的 frontmatter、相对链接和 `git diff HEAD --check`。
- `git diff HEAD --check`：由 repository validator 执行并通过；最终又独立执行一次作为交付门禁。

## 未完成项与剩余风险

- 两项 source finding 均已有本地实现和可证伪负向 fixture，无未完成整改项。
- 未执行 Go test/vet/race 或阶段五 binary、migration、文件系统与真实部署测试：本 REM 只修改 Markdown 治理的 PowerShell validator/self-test，不涉及产品实现。
- 剩余风险是未来可能引入新的自然语言关系词或新 forbidden category；当前多 `WP-*` 句子在命中已知依赖信号但未完整消费时 fail-closed，结构化 contract 中的未知 category 也会失败；仍需 follow-up audit 独立验证不存在其他稳定绕过。

## Follow-up 交接

当前已具备 follow-up audit 条件；使用 `$backend-follow-up-audit TARGET=REM-0004` 独立复审。
