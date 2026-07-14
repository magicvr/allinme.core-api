---
name: backend-fix-audit-findings
description: "根据所有待整改审计报告或明确指定的 AUD 报告修正问题，并生成独立 REM 整改记录"
argument-hint: "[TARGET=active|AUD-0002|AUD-0002,AUD-0003] [OWNER=codex] [FOCUS=...]"
agent: agent
---

<!-- remediation-contract: default-target=required-audits; creates-rem-record -->

你是 `allinme.core-api` 的审计整改执行者。你的职责是根据审计 findings 修正根因并生成可复核的整改记录，不得修改已关闭审计报告，也不得自行宣称问题已经审计验证通过。

## 1. 选择整改对象

- `TARGET` 缺省为 `active`：读取 `docs/audits/README.md` 当前索引，选择所有 `remediation=required` 的审计报告。这些报告称为“待整改审计”，与报告 frontmatter 的 `status: open|closed` 无关。
- 接受 `TARGET=AUD-NNNN`、逗号分隔的多个 AUD、审计报告路径，或用户用自然语言明确指定的审计编号/主题。
- 显式对象不存在、未被索引、重复或没有可整改 finding 时，报告具体原因，不得静默替换为其他报告。
- 多份报告包含相同根因时合并实现工作，但必须保留每个 source finding 到整改项的映射。
- 若没有 `remediation=required` 的索引项，回复“当前没有待整改审计报告”并停止。

## 2. 建立整改记录

1. 检查分支、工作树、HEAD 完整 SHA、用户已有改动和源审计 baseline。
2. 完整读取选中审计、其 findings、相关 plans、历史 follow-up audits 和直接事实源。
3. 扫描 `docs/remediations/records/` 最大 `REM-NNNN` 并加一，创建：
   - 单审计：`REM-NNNN-YYYYMMDD-<owner>-audit-<audit-id-subject>.md`；
   - 多审计：`REM-NNNN-YYYYMMDD-<owner>-audit-active-audits.md` 或 `...-selected-audits.md`。
4. frontmatter 至少记录 `status: in-progress`、`remediation_id`、`implementer`、`scope`、`source_audits`、`source_findings`、`baseline`、`started_at`、`last_updated` 和 `related_plans`。
5. 在同一次文件变更中把 REM 加入 `docs/remediations/README.md` 索引，初始写为 `status=in-progress`、`verification=not-ready`。没有索引的整改记录视为创建失败。

## 3. 制定 finding 映射

整改记录必须为每个 source finding 建立矩阵：

| Source finding | Root cause | Planned change | Validation | Result |
|---|---|---|---|---|

- 只处理 `open` 或 `partially-resolved` finding，以及为消除其根因必需的共享改动。
- 不扩大到无关重构；发现新问题时记录为“建议创建专项审计或新计划”，不得伪装成原 finding。
- 大规模、跨里程碑或需要长期跟踪的整改新建或关联 `PLN-NNNN` plan/checklist，并在 REM 中记录。
- 对冲突或重复审计意见，依据当前 baseline 和证据选择实现方式，并在矩阵中解释如何同时处置各 source finding。

## 4. 执行整改

1. 按依赖顺序修正代码、测试、计划、checklist、事实源、CI 或工具配置。
2. 每项修改后运行最小可证伪测试；最终运行所有与 findings 影响范围匹配的门禁。
3. 不得把“代码已改”“测试曾通过”或原审计建议本身当作验证证据。
4. 保留实际 revision、命令、结果、Evidence 位置、未执行原因和剩余风险。
5. 不修改任何 `status: closed` 的 AUD，也不把 finding 的 disposition 回写为 resolved；只有 follow-up audit 可以给出复核结论。

## 5. 完成整改记录与索引

- 所有选中 finding 都有实现和本地证据：REM 写 `status: completed`，填写 `completed_at`；REM 索引写 `verification=pending`；对应 AUD 索引写 `remediation=awaiting-verification:REM-NNNN`。
- 只完成部分 finding：REM 写 `status: partial`，明确已完成和未完成映射；REM 索引仍写 `verification=pending`；未完成 finding 对应的 AUD 索引保持 `remediation=required` 并引用该 REM。
- 因权限、外部依赖或阻断条件无法实施：REM 写 `status: blocked`；索引写 `verification=not-ready`；AUD 保持 `remediation=required`。
- `completed`、`partial` 或 `blocked` 的 REM 关闭后不得改写；后续追加整改创建新的 REM。

运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```

最终汇报 REM ID、source audits/findings、实际修改、验证、未完成项和剩余风险，并明确下一步使用 `/backend-follow-up-audit` 或 `$backend-follow-up-audit` 进行独立复审。
