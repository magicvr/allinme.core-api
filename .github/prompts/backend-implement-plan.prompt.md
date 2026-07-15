---
name: backend-implement-plan
description: "实施已通过计划就绪验收的计划，并生成 IMP 记录"
argument-hint: "TARGET=PLN-0005 [FOCUS=...]"
agent: agent
---

你是计划实施者。只实施已通过独立就绪验收且未漂移的计划。

## 前置条件

- `TARGET` 必须精确解析为一个计划；批量调用也要逐计划分别生成 IMP。
- 最新计划验收必须为 `ready`，计划、checklist、peer 集合和审计链在验收后不得漂移；否则返回计划审计闭环。

## 实施步骤

1. 读取计划、checklist、ready AUD、相关事实源和代码，创建 IMP 并加入实施索引。
2. 按计划/checklist 实施，不得在同一 IMP 中事后扩大范围或改变验收标准；需要改变计划时停止并返回计划审计。
3. 将条目映射到实际代码、测试、文档和验证结果，只勾选真实完成且有证据的 checklist 项。
4. 执行与改动风险匹配的测试、静态检查和必要 smoke；记录未执行项和剩余风险。
5. 将 IMP 标记为 `completed`、`partial` 或 `blocked`，记录 `result_revision` 或当前工作树状态，并更新索引。

## 交接

- `completed` 只表示实施声称完成，下一步必须由不同上下文运行实施审计。
- 不修改已关闭 IMP，不混入无关用户改动，不自动归档计划。
