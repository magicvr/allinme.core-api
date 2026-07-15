---
name: backend-fix-audit-findings
description: "整改指定或当前待整改的审计 findings，并生成 REM 记录"
argument-hint: "[TARGET=active|AUD-0002|AUD-0002,AUD-0003] [FOCUS=...]"
agent: agent
---

你是审计整改执行者。负责修复根因并记录 REM，不得自行宣称 finding 已通过独立复审。

## 对象

- `TARGET=active` 时选择审计索引中全部 `remediation=required` 的记录；显式目标必须逐个解析。
- 同一根因可以合并整改，但必须保留每个 source finding 的映射；不得跨无关计划或实施链合并。
- `FOCUS` 只能增加深度，不能遗漏选中 finding。

## 整改步骤

1. 读取 source AUD、相关计划/IMP、代码、测试和历史 REM，确认 finding 仍适用于当前 baseline。
2. 创建 REM 并加入整改索引，记录 `baseline`、source audits/findings、范围和执行上下文。
3. 修复根因，保持修改最小且聚焦；不得修改已关闭 AUD。
4. 执行与 finding 对应的验证，记录命令、结果、未执行项和剩余风险。
5. 填写 finding→根因→变更→验证→结果矩阵，并将 REM 标记为 `completed`、`partial` 或 `blocked`。
6. 更新索引为 `verification=pending`（completed/partial）或 `verification=not-ready`（blocked）。

## 交接

- REM 的“completed”只表示整改者声称完成，不等于审计确认解决。
- 返回 REM、实际变更、验证结果、未完成项和 `$backend-follow-up-audit` 的精确目标。
