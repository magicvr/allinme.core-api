---
name: backend-implement-plan
description: Implement active or explicitly selected PLN plans after readiness acceptance, update checklist evidence, and create one indexed IMP implementation record per plan.
---

# Backend Implement Plan

1. 解析仓库根目录，并在修改任何文件前完整读取 `.github/prompts/backend-implement-plan.prompt.md`。
2. 将该 prompt 视为唯一规范正文，完整执行计划选择、就绪验收前置检查、IMP 创建、实施、验证和关闭交接。
3. 将调用文本解释为可选 `TARGET`、`IMPLEMENTER`、`CONTEXT_ID`、`CONTEXT_REF` 和 `FOCUS`；默认 `TARGET=active`。`FOCUS` 不得缩小计划范围；`CONTEXT_ID` 只作为运行关联 ID。
4. 每个计划必须有最新且已关闭的 `acceptance_verdict: ready`，其完整 peer 快照、subject paths 和审计链均未漂移；否则停止该计划并说明需要运行 `$backend-plan-audit-until-ready`。
5. 每个计划创建或恢复独立 IMP，记录 `execution_context_id`；`baseline` 固定治理快照，交付 subject commit 与最终 IMP/索引治理提交分开。若由完成验收的 implement action 触发，记录 `trigger_audits` 并把源 AUD 索引流转到 `implemented-by:IMP-NNNN`；IMP 和索引建立前不得开始产品实现。
6. 只允许在 checklist 写实际 Evidence；plan 契约、范围或冻结值变化必须关闭当前 IMP 并重新计划审计/验收。不得自动归档计划或修改已关闭 AUD/REM/IMP。
7. 生成的实施记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。
8. 所有实际提交必须调用 `docs/tools/invoke-governance-transaction.ps1`：以完整 HEAD 作为 `ExpectedHead`，只列出本阶段精确 `Paths`，在 Git common-dir 锁内对 HEAD 和 `refs/allinme/governance-head` 做 CAS。open checkpoint 必须精确包含 IMP、必要索引及本记录自身的 `runtime_context_attestation` 文件；subject/result commit 与 terminal governance commit 必须线性分开并返回干净 `governance_revision`。禁止裸 `git add`/`git commit`、预暂存或混入用户/并行改动。
9. 新建 IMP 前必须由仓库外运行时适配器提供真实 `CONTEXT_REF` 和绑定 scope/baseline/task、exact `record_id`/`record_path` 的单次签名 `runtime_context_attestation`；仓库不得持有私钥，缺失 trust anchor 或验签失败时停止。
10. 每份新 IMP 必须由真实、独立的逐记录 child task/context 创建，并使用全局唯一 `execution_context_id`；批量入口严格串行，禁止复用 `CONTEXT_ID`/`CONTEXT_REF`。恢复 in-progress IMP 仅限当前真实 ref 与记录一致且原 task 可恢复；否则必须按 canonical prompt 以 `context-loss` 原子 supersede 旧 IMP 并创建新记录，禁止接管。`partial`/`blocked` 也必须通过 helper 形成必要的 subject/result 与独立 terminal governance transaction，并返回干净 `governance_revision`。

```text
$backend-implement-plan
$backend-implement-plan TARGET=PLN-0005
```
