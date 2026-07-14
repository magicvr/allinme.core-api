---
name: backend-implement-plan
description: Implement active or explicitly selected PLN plans after readiness acceptance, update checklist evidence, and create one indexed IMP implementation record per plan.
---

# Backend Implement Plan

1. 解析仓库根目录，并在修改任何文件前完整读取 `.github/prompts/backend-implement-plan.prompt.md`。
2. 将该 prompt 视为唯一规范正文，完整执行计划选择、就绪验收前置检查、IMP 创建、实施、验证和关闭交接。
3. 将调用文本解释为可选 `TARGET`、`IMPLEMENTER` 和 `FOCUS`；默认 `TARGET=active`。
4. 每个计划必须有最新 `acceptance_verdict: ready`，否则停止该计划并说明需要运行 `$backend-plan-audit-until-ready`。
5. 每个计划创建独立 IMP；IMP 和索引建立前不得开始产品实现。
6. 不得把未执行项勾选为完成，不得自动归档计划，也不得修改已关闭 AUD/REM/IMP。
7. 生成的实施记录和最终汇报必须使用中文；代码、命令、路径、ID 及固定状态值保持原样。

```text
$backend-implement-plan
$backend-implement-plan TARGET=PLN-0005
```
