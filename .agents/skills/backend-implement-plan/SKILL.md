---
name: backend-implement-plan
description: Implement a plan that has a current ready acceptance.
---

# backend-implement-plan

1. 从仓库根目录完整读取 `.github/prompts/backend-implement-plan.prompt.md`。
2. 将该 prompt 作为唯一流程规范，实施已通过就绪验收的计划并生成 IMP。
3. 保留用户指定的 `TARGET`、`FOCUS`、cycle 上限及其他参数，不得静默扩大或缩小范围。
4. prompt 缺失、目标无法解析或独立上下文要求无法满足时停止并返回精确 handoff，不得凭记忆重建流程。
5. 遵守仓库现有变更保护：保留用户改动、不修改终态历史记录、如实记录未执行验证和剩余风险。
