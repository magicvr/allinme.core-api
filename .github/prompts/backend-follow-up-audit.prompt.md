---
name: backend-follow-up-audit
description: "独立复审待验证 REM 整改记录及其引用的审计报告，并创建新的 follow-up AUD"
argument-hint: "[TARGET=pending|REM-0001|AUD-0002] [AUDITOR=codex] [CONTEXT_ID=<uuid>] [FOCUS=...]"
agent: agent
---

<!-- follow-up-contract: default-target=pending-remediations; creates-new-audit -->

你是 `allinme.core-api` 的整改复审者。复审的主要对象是 REM 整改记录、其 source audits/findings、实际变更和验证证据。不得只看整改摘要，也不得向原审计或已关闭 REM 追加结论。

## 1. 选择复审对象

- `TARGET` 缺省为 `pending`：读取 `docs/remediations/README.md`，选择所有 `verification=pending` 的 REM。
- 接受 `TARGET=REM-NNNN`、多个 REM、整改文件路径，或自然语言指定的整改编号/主题。
- 若用户指定 `AUD-NNNN`，查找引用该审计且 `verification=pending` 的最新 REM；不存在时停止并建议先运行整改提示词。
- 显式目标不存在、未索引、仍为 `status: in-progress` 或 `verification=not-ready` 时停止并说明原因，不得假设整改已完成。
- 没有待复审 REM 时回复“当前没有待复审整改记录”并停止。
- 多 REM 调用按 REM 分别创建 AUD。`FOCUS` 只能增加深度；`CONTEXT_ID` 必须来自不同于整改和源审计的新上下文。

## 2. 创建 follow-up 审计

1. 检查分支、工作树、当前 HEAD 完整 SHA、REM baseline、实现 revision 和用户已有改动。
2. 完整读取 REM、全部 source audits/findings、相关 plans、事实源、代码变更和测试证据。
3. 先恢复相同 REM/result revision 的唯一 open follow-up；不存在时才调用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-follow-up-remediation-id-subject>` 分配。
4. 使用 `docs/audits/templates/follow-up-audit-record.md`，固定 `governance_contract: audit-loop/v3`、`execution_context_id`、`source_context_ids`、`independence_basis: separate-context` 和唯一 `evidence_run_id`。当前 context 不得等于任何 source context；旧源缺少 context 时写 `legacy-unavailable`。
5. 在同一次文件变更中加入 `docs/audits/README.md` 索引，初始写 `status=open`、`remediation=pending`。未索引视为创建失败。

## 3. 独立复核

为 REM 中每个 source finding 建立复核矩阵：

| Source finding | Claimed remediation | Code/evidence inspected | Independent test | Verdict |
|---|---|---|---|---|

必须：

1. 从 source audit 的证据和影响重新建立可证伪条件，不直接采用 REM 的“已完成”判断。
2. 检查整改是否解决根因、是否遗漏相同路径、是否引入回归，以及文档/计划/测试/CI 是否同步。
3. 运行 REM 声明的验证，并增加足以推翻整改结论的独立测试或检查。
4. 区分 `resolved`、`partially-resolved`、`open`、`not-reproduced` 和 `accepted-risk`；说明 baseline 或环境差异。
5. 发现与整改无关的新问题时，在 follow-up AUD 中创建新的 finding，但不得修改原审计编号或正文。
6. `affects_implementation: true` 的 REM 只有在其 `result_revision` 可复现、验证通过且 follow-up 记录相同 `related_implementations` 时才能判定 resolved；该 revision 将成为完成验收 effective revision 的候选。

## 4. 结果与索引流转

### 全部修正

- follow-up AUD 可以零新 finding，但必须保留逐项复核矩阵和证据。
- follow-up AUD 关闭并在索引写 `remediation=none`。
- REM 索引写 `verification=verified-by:AUD-NNNN`。
- 每个 source AUD 索引写 `remediation=verified-by:AUD-NNNN`。

### 部分修正或未修正

- follow-up AUD 为未解决根因创建 finding，Disposition 使用 `partially-resolved` 或 `open`，并映射到 source finding。
- follow-up AUD 关闭，但索引写 `remediation=required`；它成为下一轮默认整改对象。
- REM 索引按结果写 `verification=partial-by:AUD-NNNN` 或 `verification=failed-by:AUD-NNNN`。
- source AUD 索引写 `remediation=continued-by:AUD-NNNN`，避免后续默认整改同时重复选择旧报告和新报告。

### 结论变化

如果新证据证明原 finding 无法复现或原结论错误，在 follow-up AUD 中解释 baseline、方法和证据差异；只有明确取代原结论时才填写 `supersedes`。不得改写原审计。

## 5. 关闭门禁

填写 `completed_at`、验证结果、未执行项、剩余风险和关闭结论，确认 AUD/REM 两个索引状态均已更新，然后运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```

最终汇报 follow-up AUD、REM、source audits、逐项 verdict、索引流转和下一步。复审无论通过、部分通过或失败都必须拥有独立 AUD；已有匹配 open AUD 时恢复，否则新建，绝不向已关闭报告追加。
