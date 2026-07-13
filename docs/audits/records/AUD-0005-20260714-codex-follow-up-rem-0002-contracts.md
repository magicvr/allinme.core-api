---
status: closed
audit_id: AUD-0005
auditor: codex
audit_type: follow-up
scope: follow-up:REM-0002
subject: rem-0002-contracts
baseline: git:731bcf29e01153bab367abc5151834b7ea252f37; worktree:clean
started_at: 2026-07-14T05:13:25+08:00
completed_at: 2026-07-14T05:20:41+08:00
last_updated: 2026-07-14
related_audits: AUD-0004
related_remediations: REM-0002
supersedes: none
related_plans: PLN-0005
---

# REM-0002 阶段五计划契约整改复审

## 目的与范围

独立复审 `REM-0002` 对 `AUD-0004-F001` 与 `AUD-0004-F002` 的整改有效性。范围包括整改提交、`PLN-0005` 与 checklist、文档治理 validator 及其测试；不把尚未实现的阶段五 binary、migration 或真实部署 Evidence 当作本次计划契约整改证据。

## 基线与方法

- 固定基线：`main@731bcf29e01153bab367abc5151834b7ea252f37`；复审开始时工作树 clean。
- `REM-0002` 记录的实施基线为 `00ff9cbe9034efdd4b9c46700d39dfaa8435ed31` 上未提交整改 diff；当前基线已将该 diff 提交为 `731bcf29e01153bab367abc5151834b7ea252f37`。
- 方法：从 source findings 重新建立失败条件，核对实际 commit diff、计划/checklist/validator 一致性，运行整改声明的验证并增加反向语义检查。

## 历史关系

- `AUD-0004` 是 `REM-0001` 的 follow-up audit；其中四项 source findings 已 resolved，本轮只复审转入 `REM-0002` 的两项剩余 finding。
- 本记录不修改已关闭的 `AUD-0004` 或 `REM-0002` 正文。

## 复核矩阵

| Source finding | Claimed remediation | Code/evidence inspected | Independent test | Verdict |
|---|---|---|---|---|
| `AUD-0004-F001` | P0-22/P0-23 只保留 deployment contract fixture，把真实监督器、cleanup 调度、watchdog/recovery、ENOSPC 和 profile run 后移到 5A-D/5B；validator 增加负向 fixture。 | 检查提交 `731bcf2`、plan 第 39 项、checklist P0-22/P0-23、5A-D-2、5B-4 与 validator/test diff。当前文本已明确 P0 不要求阶段五 binary 或真实部署 run，P0-22 只产生 `artifactKind=contract-fixture`。 | 正常 validator 与自测通过；将 P0-22 的“不要求真实 run”原句替换为真实 run 要求时 validator 拒绝。但保留合法 P0-22/P0-23 后另加 P0-26，重新要求真实 supervisor/cleanup/watchdog/ENOSPC/profile Evidence，validator 仍退出 0。 | `partially-resolved`；当前循环已消除，但语义回归门禁可被附加矛盾条款绕过；见 `AUD-0005-F001` |
| `AUD-0004-F002` | tracked 表为 `WP-Baseline-Evidence` 增加 `WP-Facts` 输入，P0-21 与 validator/test 增加缺边拒绝。 | 检查提交 `731bcf2`、tracked work-package 表、§8 dependency prose、checklist P0-21 与 validator/test diff。表、正文和 checklist 当前均声明 `WP-Baseline-Evidence → WP-Facts`。 | 正常 validator 与自测通过；从 tracked row 删除 `WP-Facts` 时 validator 拒绝。但保留合法表/§8 后另加一句允许 Baseline-Evidence 先于 Facts 且不依赖它，validator 仍退出 0。 | `partially-resolved`；当前 DAG 已一致，但语义回归门禁可被附加矛盾正文绕过；见 `AUD-0005-F002` |

## Findings

### AUD-0005-F001 - P0 deployment Evidence validator 只检查必需短语，未拒绝附加循环门禁

- Maps to: `AUD-0004-F001`
- Severity: medium
- Evidence: 当前 P0-22/P0-23 正确限定为 `contract-fixture`，且将真实 supervisor、cleanup 调度、watchdog/recovery、ENOSPC 与 profile run 后移到 5A-D/5B。`docs/tools/validate.ps1` 仅检查 P0-22/P0-23 行是否包含 `artifactKind=contract-fixture`、`5A-D-2`、`5B-4` 等正向短语；独立测试在保留这些合法行的同时新增 P0-26，重新要求 P0 完成前提供真实 supervisor、cleanup、watchdog/recovery、ENOSPC 和 live profile Evidence，validator 仍以 0 退出。
- Impact: 当前计划已可执行，但后续编辑可以保留被检查的合法条款并在其他 P0 项重新引入同一 P0→M1A/5A-D 循环，治理门禁会错误放行；`REM-0002` 声明的“防止相同语义漂移”尚未成立。
- Recommendation: 将 P0 deployment Evidence 边界抽取为结构化单一事实源并验证所有 P0 项/正文引用；至少增加负向 fixture，证明新增任何要求真实 binary、supervisor、cleanup schedule、watchdog/recovery、ENOSPC 或 live profile run 的 P0 条款都会失败，而后续 5A-D/5B 条款仍被允许。
- Owner: Evidence tooling owner / release owner
- Disposition: partially-resolved

### AUD-0005-F002 - dependency validator 未拒绝 tracked 表之外的相反执行边

- Maps to: `AUD-0004-F002`
- Severity: medium
- Evidence: tracked row、§8 与 P0-21 当前都包含 `WP-Baseline-Evidence → WP-Facts`，删除 tracked row 中的 `WP-Facts` 会被拒绝。但 validator 只要求存在一行匹配固定正向顺序；独立测试在保持表和原 §8 合法文字不变时，另加一句允许 `WP-Baseline-Evidence` 先于 `WP-Facts` 且不依赖它，validator 仍以 0 退出。
- Impact: 当前 DAG 已一致，但后续正文可同时出现两个执行顺序并通过治理门禁，重新造成 `AUD-0004-F002` 的 owner、elapsed、输入 revision 与 No-Go 分叉。
- Recommendation: 从 tracked table 解析完整 DAG，并拒绝正文/checklist 中任何无法映射到该图或与其相反的依赖陈述；新增 additive contradiction fixture，而不只测试删除 tracked edge。
- Owner: Evidence tooling owner / phase-five protocol owner
- Disposition: partially-resolved

## 验证结果

- `git show 731bcf2` 与 `git diff 00ff9cb..731bcf2`：整改范围与 REM 声明一致；closed `AUD-0004`、`REM-0001` 正文无 diff。
- 独立结构检查：`DagRowHasFacts=True`、`DagProseMatches=True`、`P022ContractOnly=True`、`P023DefersLive=True`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1`：通过，验证 44 个 Markdown 文件、frontmatter、相对链接和 `git diff HEAD --check`。
- `powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1`：通过；REM 新增的 exact-removal/exact-replacement 负向 fixture 被拒绝。
- 独立 additive contradiction：只新增相反 DAG 正文时 validator exit 0；只新增 P0 live deployment gate 时 validator exit 0；两项同时新增时同样 exit 0。临时变更均已移除，plan/checklist 文本与 HEAD 无 diff。

## 未执行项与剩余风险

- 未执行阶段五 binary、migration、文件系统、crash 或真实部署测试：本轮 source findings 针对计划契约和治理门禁，相应产品实现尚不存在。
- 未执行 Go test/vet/race：`REM-0002` 只修改 plan/checklist 与 PowerShell 文档 validator；已运行其声明的文档验证及独立反向测试。
- 当前计划文本已消除部署 Evidence 循环并统一 DAG；剩余风险是 validator 不能拒绝附加的相反条款，因此不能把治理防回归部分视为 resolved。

## 关闭结论

`REM-0002` 部分通过：`AUD-0004-F001` 与 `AUD-0004-F002` 的当前计划文本均已修正，但两项用于防止同类语义漂移的 validator 保护都可被附加矛盾条款绕过。`AUD-0005` 关闭并进入新的整改队列；`REM-0002` 标记 partial verification，source audit 的活动队列转移到本 follow-up audit。
