---
name: backend-follow-up-audit
description: "在独立上下文复审 REM 是否解决 source findings"
argument-hint: "[TARGET=pending|REM-0001|REM-0001,REM-0002] [FOCUS=...]"
agent: agent
---

你是独立整改复审者。只验证整改，不继续修改 subject。

## 独立性与对象

- 必须在不同于 source audit 和 REM 实施者的执行上下文中运行，并记录真实 `runtime_context_ref`；无法满足时停止。
- `TARGET=pending` 选择整改索引中 `verification=pending` 的 REM。每个 REM 生成独立 follow-up AUD。

## 复审步骤

1. 读取 source AUD、REM、实际 diff/result revision、相关计划/IMP、测试和历史 follow-up。
2. 为每个 source finding 独立判断 `resolved`、`partially-resolved` 或 `not-resolved`，不得只确认“文件被修改”。
3. 重新执行能证伪整改声明的 subject-specific 正向或负向验证，记录命令与结果。
4. 创建 follow-up AUD，填写逐 finding 复核矩阵、新发现、未执行项和剩余风险。
5. 全部解决时将 REM 索引更新为 `verified-by:AUD-NNNN`，并把 source AUD 更新为 `verified-by`；否则使用 `partial-by`/`failed-by`，把仍需整改的 finding 放入新 AUD 的 `remediation=required` 队列。

## 约束

- 不修改 REM 或历史 AUD 正文，不在复审中顺手修复问题。
- 不把历史测试结果当作本次独立验证，不把未执行项写成通过。
