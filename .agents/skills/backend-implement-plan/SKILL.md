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

```text
$backend-implement-plan
$backend-implement-plan TARGET=PLN-0005
```
