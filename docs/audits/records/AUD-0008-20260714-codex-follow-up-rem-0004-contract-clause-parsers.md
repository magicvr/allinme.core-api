---
status: closed
audit_id: AUD-0008
auditor: codex
audit_type: follow-up
scope: follow-up:REM-0004
subject: rem-0004-contract-clause-parsers
baseline: git:596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1; worktree:dirty (REM-0004 implementation diff plus pre-existing AUD-0007 record/index change)
started_at: 2026-07-14T13:13:16+08:00
completed_at: 2026-07-14T13:16:00+08:00
last_updated: 2026-07-14
related_audits: AUD-0006
related_remediations: REM-0004
supersedes: none
related_plans: PLN-0005
---

# REM-0004 条款级 contract parser 整改复审

## 目的与范围

独立复审 `REM-0004` 对 `AUD-0006-F001` 与 `AUD-0006-F002` 的整改有效性。范围包括 `PLN-0005`、checklist、文档治理 validator/self-test 和当前未提交整改 diff；不把阶段五产品 binary、migration 或真实部署 Evidence 纳入本次文档门禁整改结论。

## 基线与方法

- 固定基线：`main@596477a1bf2c9fc0d4d9f5f27e3e913c9854b4c1`；复审开始时工作树包含 `REM-0004` 的 validator/self-test/index/record 未提交变更，以及整改开始前已存在的 `AUD-0007` record/index 变更。
- 方法：从 `AUD-0006` 的两个 source finding 重新建立可证伪条件，核对 `REM-0004` 声明与实际 diff，运行声明验证，并在隔离的完整文档副本中增加 direct control 与 masked contradiction fixture。

## 历史关系

- `AUD-0006` 是 `REM-0003` 的 follow-up audit；本轮只复审转入 `REM-0004` 的 `AUD-0006-F001` 与 `AUD-0006-F002`。
- 本记录不修改已关闭的 `AUD-0006` 或 `REM-0004` 正文；索引只在复审关闭后流转当前队列。

## 复核矩阵

| Source finding | Claimed remediation | Code/evidence inspected | Independent test | Verdict |
|---|---|---|---|---|
| `AUD-0006-F001` | 从结构化 contract 派生 forbidden category；聚合完整 P0 checklist item；按 clause 区分 deferral 与 obligation；新增同行 deferral 后 obligation 和多行 continuation fixture。 | 检查 `Get-PhaseFiveLiveEvidencePattern`、`Test-PhaseFiveDeploymentClauses`、contract 解析、P0 item block regex 和新增 self-test。`AUD-0006` 的同行句号分隔与 Markdown continuation 复现均已被拒绝。 | direct control 把矛盾义务作为独立句追加到 P0-22，validator exit 1；把同一义务改为中文 `但` 连接、与既有 `P0 不要求...` deferral 保持在同一 scanner clause，validator exit 0。 | `partially-resolved`；原复现已修正，但 deferral 仍可覆盖同一未切分 clause 中的相反 obligation；见 `AUD-0008-F001`。 |
| `AUD-0006-F002` | 完整消费 `depends on` tail；识别 pronoun negation 与 reverse `after`；已知依赖信号出现但未完整消费时 fail-closed；新增三类负向 fixture。 | 检查 `Test-PhaseFiveDependencyStatements` 的 arrow、negation、depends-on、before/after、consumed package 集合与 rejection-clause 豁免。`AUD-0006` 的 pronoun、conjoined unknown 和 reverse after 复现均已被拒绝。 | direct control `WP-Release depends on WP-Unknown` exit 1；同一矛盾放到含 `validator must reject` 的行中 exit 0；`WP-Release depends upon WP-Unknown` 也 exit 0。 | `partially-resolved`；三个原复现已修正，但整行 rejection 豁免与有限 signal 词表仍允许稳定 additive contradiction；见 `AUD-0008-F002`。 |

## Findings

### AUD-0008-F001 - clause 级 deferral 仍可掩盖同一条款中的相反 P0 obligation

- Maps to: `AUD-0006-F001`
- Severity: medium
- Evidence: `Test-PhaseFiveDeploymentClauses` 会按句号、分号、英文 `but/however/yet` 等切分 clause，但不切分中文转折 `但`；随后只要整个 clause 任意位置匹配 `$deferral`，即使同一 clause 后半段同时匹配 live Evidence 与 `$obligation`，第 291 行的 `-notmatch $deferral` 也会豁免整段。隔离 fixture 将真实 P0-22 的 `P0 不要求...ENOSPC run。` 改为 `P0 不要求...ENOSPC run，但 P0 完成前必须提供真实监督器 run、cleanup 调度、watchdog/recovery、ENOSPC Evidence 和 live deployment profile。`，validator exit 0；把相反义务改成句号后的独立 clause 时 control exit 1 并命中预期错误。
- Impact: 后续计划编辑仍可在合法冻结语句后用未被 clause splitter 识别的连接方式重新引入 P0→真实部署 Evidence 循环，validator 会把相反 obligation 当作 deferral 的一部分放行。
- Recommendation: 不要用 clause 内“存在任意 deferral token”作为整体豁免；应按 live-Evidence occurrence 建立局部 polarity/obligation 判断，至少覆盖中文转折/并列连接，并增加同一 clause 中 deferral→obligation 与 obligation→deferral 的双向负向 fixture。
- Owner: Evidence tooling owner / release owner
- Disposition: partially-resolved

### AUD-0008-F002 - rejection-clause 整行豁免与有限关系词表仍可绕过 dependency parser

- Maps to: `AUD-0006-F002`
- Severity: medium
- Evidence: `Test-PhaseFiveDependencyStatements` 只要一行任意位置命中 `reject|must reject|拒绝` 就设置 `$isRejectionClause`，除 arrow 外的 negation/depends/before/after 与 unconsumed package 检查全部跳过。隔离 fixture 追加 `The validator must reject malformed dependency prose, but WP-Release depends on WP-Unknown.`，validator exit 0；direct control `WP-Release depends on WP-Unknown.` exit 1。另一个 fixture `WP-Release depends upon WP-Unknown.` 因 `$relationshipSignal` 和 positive parser 都不识别 `depends upon` 而 exit 0。
- Impact: 正文可以在 validator 规范句同一行追加未知依赖，或使用常见等价关系词，重新造成 tracked DAG 与 prose 的 owner、输入 revision 和关键路径分叉。
- Recommendation: 把 rejection 豁免限制到明确隔离的示例/引用 payload，不得豁免同一行其他 clause；关系声明应优先禁止自由文本并从 tracked DAG 生成，否则扩充 canonical relation grammar，并对含多个 `WP-*` 且存在未识别关系连接词的句子 fail-closed。新增 rejection-line contamination 与 `depends upon` 负向 fixture。
- Owner: Evidence tooling owner / phase-five protocol owner
- Disposition: partially-resolved

## 验证结果

- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；`REM-0004` 声明的五类回归 fixture 均按预期拒绝，合法 fixture 被接受。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：复审关闭后通过，验证 55 个 Markdown 文件、frontmatter、相对链接与 `git diff HEAD --check`。
- `git diff HEAD --check`：复审关闭后独立通过。
- 隔离文档副本 baseline：validator exit 0；direct F001/F002 controls 均 exit 1 并命中预期 failure。
- 独立 F001 same-clause deferral mask：validator exit 0。
- 独立 F002 rejection-line mask：validator exit 0。
- 独立 F002 `depends upon`：validator exit 0。

## 未执行项与剩余风险

- 未执行阶段五 binary、migration、文件系统、crash、Go test/vet/race 或真实部署测试：本轮 source findings 与整改仅针对 Markdown governance 的 PowerShell validator/self-test。
- 当前 `PLN-0005` tracked DAG、结构化 deployment contract 与已有 P0 文本仍通过 validator；本结论不否定当前文本，只证明防 additive semantic drift 的门禁仍不完备。
- 工作树中的 `AUD-0007` record/index 是 `REM-0004` 开始前已有改动，本复审保留且未改写其正文。

## 关闭结论

`REM-0004` 部分通过：`AUD-0006` 的五个精确复现均已修正并进入 self-test，但两个根因仍存在可稳定复现的相邻语法绕过。`AUD-0008` 关闭并以两项 `partially-resolved` finding 成为下一轮整改对象；`REM-0004` 标记 partial verification，source audit 的当前队列转移到本 follow-up audit。
