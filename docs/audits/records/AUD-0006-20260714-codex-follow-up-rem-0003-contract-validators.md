---
status: closed
audit_id: AUD-0006
auditor: codex
audit_type: follow-up
scope: follow-up:REM-0003
subject: rem-0003-contract-validators
baseline: git:0f602933e4afa3e53e5b964a0990790d59485ea3; worktree:clean
started_at: 2026-07-14T12:28:32+08:00
completed_at: 2026-07-14T12:35:12+08:00
last_updated: 2026-07-14
related_audits: AUD-0005
related_remediations: REM-0003
supersedes: none
related_plans: PLN-0005
---

# REM-0003 计划契约 validator 整改复审

## 目的与范围

独立复审 `REM-0003` 对 `AUD-0005-F001` 与 `AUD-0005-F002` 的整改有效性。范围包括整改提交、`PLN-0005` 与 checklist、文档治理 validator 及其测试；不把尚未实现的阶段五 binary、migration 或真实部署 Evidence 当作本次计划契约整改证据。

## 基线与方法

- 固定基线：`main@0f602933e4afa3e53e5b964a0990790d59485ea3`；复审开始时工作树 clean。
- `REM-0003` 记录的实施基线为 `4d487b4b499d1012ea551960503b75bde9c0cc94` 上未提交整改 diff；当前基线已将该 diff 提交为 `0f602933e4afa3e53e5b964a0990790d59485ea3`。
- 方法：从 source findings 重新建立失败条件，核对实际 commit diff、计划/checklist/validator 一致性，运行整改声明的验证并增加可绕过现有正向 fixture 的 additive contradiction 检查。

## 历史关系

- `AUD-0005` 是 `REM-0002` 的 follow-up audit；本轮只复审转入 `REM-0003` 的两项剩余 finding。
- 本记录不修改已关闭的 `AUD-0005` 或 `REM-0003` 正文。

## 复核矩阵

| Source finding | Claimed remediation | Code/evidence inspected | Independent test | Verdict |
|---|---|---|---|---|
| `AUD-0005-F001` | 增加结构化 P0 deployment Evidence contract，校验 P0-1..P0-25 完整唯一集合，并扫描 P0 checklist/plan 条款拒绝附加 live deployment gate。 | 检查提交 `0f60293`、plan 中 `phase5-p0-deployment-evidence-contract`、checklist P0-1..P0-25、`Test-PhaseFiveDeploymentClauses` 与新增 self-test fixture。当前计划文本和结构化 contract 一致；额外 P0-26、无 deferral 的单行 P0 live gate 与单行 plan live gate 会被拒绝。 | 在真实 P0-22 合法 deferral 后追加“P0 完成前必须提供真实监督器 run、cleanup schedule、watchdog/recovery、ENOSPC Evidence 和 live deployment profile run”，validator exit 0；把同一 live gate 写为 P0-20 的 Markdown continuation，validator 也 exit 0。 | `partially-resolved`；当前文本及既有 fixture 已修正，但 line-level deferral/扫描边界仍可放行附加矛盾条款；见 `AUD-0006-F001` |
| `AUD-0005-F002` | 从 tracked work-package 表解析完整八包 DAG，校验精确输入、未知依赖、自依赖、环与 Release 汇聚，并扫描正文/checklist 的 arrow、depends-on、before/precedes/先于和否定依赖陈述。 | 检查提交 `0f60293`、八包 tracked table、§8 dependency prose、`Get-PhaseFiveDag`、`Test-PhaseFiveDependencyStatements` 与 additive contradiction self-test。当前表的精确直接边正确，已覆盖的显式 `before`/完整否定句会被拒绝。 | 分别追加 `WP-Baseline-Evidence can consume WP-Facts metadata but does not depend on it.`、`WP-Release depends on WP-Facts and WP-Unknown.`、`WP-Facts runs after WP-Baseline-Evidence.`；三种矛盾正文均 validator exit 0。 | `partially-resolved`；tracked table 本身已被完整解析，但正文关系 parser 仍只消费部分句法，无法保证拒绝 additive contradiction；见 `AUD-0006-F002` |

## Findings

### AUD-0006-F001 - P0 live Evidence 扫描可被已有 deferral 或多行条款绕过

- Maps to: `AUD-0005-F001`
- Severity: medium
- Evidence: `docs/tools/validate.ps1` 的 `Test-PhaseFiveDeploymentClauses` 逐行判断；只要同一行任意位置命中 `not require`、`不要求`、`defer` 等 deferral，整行就不再因后续 live Evidence obligation 失败。独立 fixture 在当前合法 P0-22 末尾追加相反的 P0 完成义务，保留原有“P0 不要求”文本，validator 仍以 0 退出。另一个 fixture 把 P0-20 的 live supervisor/cleanup/watchdog/ENOSPC/profile obligation 放到下一行 Markdown continuation；父行没有 live Evidence，子行没有 P0 标识，validator 同样以 0 退出。
- Impact: 后续编辑可以保留合法的冻结文本，同时在同一 P0 项后半句或 continuation 中重新引入 P0→真实部署 Evidence 循环，治理门禁会错误放行；结构化 contract 存在但未成为逐项语义判定的唯一约束。
- Recommendation: 以完整 P0 item block 而不是单行作为扫描单位；按 clause 级别区分 deferral 与 obligation，任何 forbidden live Evidence 类别只要被 P0 obligation 引用即 fail-closed。增加“追加到真实 P0-22 合法 deferral 后”和“多行 continuation”两个负向 fixture，并从结构化 contract 派生禁止类别而不是仅校验其固定文本存在。
- Owner: Evidence tooling owner / release owner
- Disposition: partially-resolved

### AUD-0006-F002 - dependency prose parser 未完整消费同一句中的依赖关系

- Maps to: `AUD-0005-F002`
- Severity: medium
- Evidence: tracked table 的八包精确直接边、未知输入、自依赖与环已由 `Get-PhaseFiveDag` 校验；剩余问题位于 `Test-PhaseFiveDependencyStatements` 的句法扫描。否定模式要求 negation 后再次出现完整 `WP-*` prerequisite，因此 `WP-Baseline-Evidence ... WP-Facts ... does not depend on it` 不匹配；`depends on` 模式只消费首个 prerequisite，因此 `WP-Release depends on WP-Facts and WP-Unknown` 只验证合法的首边；反向 `runs after` 未纳入 ordering 模式。三个独立 fixture 均以 0 退出。
- Impact: 当前 DAG 正确，但正文仍可同时写入否定 edge、未知附加 dependency 或反向 ordering 并通过 validator，重新造成 owner、输入 revision、关键路径和 No-Go 口径分叉。
- Recommendation: 对含多个 `WP-*` 的依赖句 fail-closed，要求每个关系都能映射到 tracked DAG；至少补充 pronoun negation、conjoined prerequisites 和 reverse `after` fixture。更稳妥的长期方案是禁止自由文本声明固定依赖，只允许机器可解析的 tracked table/结构化引用，并让正文从该 DAG 生成。
- Owner: Evidence tooling owner / phase-five protocol owner
- Disposition: partially-resolved

## 验证结果

- `git diff 4d487b4b499d1012ea551960503b75bde9c0cc94..0f602933e4afa3e53e5b964a0990790d59485ea3`：整改提交只涉及 plan、validator/tests、REM 与索引；closed `AUD-0005`、`REM-0002` 正文无改写。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；现有合法 fixture 被接受，已编码的 DAG/profile 删除与 additive contradiction fixture 被拒绝。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：复审记录关闭前通过，验证 46 个 Markdown 文件、frontmatter、相对链接和 `git diff HEAD --check`。
- 独立 P0 additive contradiction：真实 P0-22 同行追加 obligation 与 P0-20 多行 continuation 均 exit 0。
- 独立 DAG additive contradiction：negative-pronoun、conjoined unknown dependency 与 reverse `after` 三种 fixture 均 exit 0。

## 未执行项与剩余风险

- 未执行阶段五 binary、migration、文件系统、crash 或真实部署测试：本轮 source findings 针对 plan 契约与 PowerShell 治理门禁，相应产品实现尚不存在。
- 未执行 Go test/vet/race：`REM-0003` 只修改 Markdown 和 PowerShell validator；已运行其声明的 validator/self-test 及独立反向 fixture。
- 当前 plan/checklist、tracked DAG 与 deployment contract 文本一致；剩余风险是自然语言扫描未按完整条款和完整关系 fail-closed，因此不能把防 additive semantic drift 视为 resolved。

## 关闭结论

`REM-0003` 部分通过：两项 source finding 的当前计划文本与结构化表/contract 均正确，且 validator 已覆盖一部分已知删除和附加矛盾；但两类独立 additive contradiction 仍可稳定通过。`AUD-0006` 关闭并进入新的整改队列；`REM-0003` 标记 partial verification，source audit 的活动队列转移到本 follow-up audit。
