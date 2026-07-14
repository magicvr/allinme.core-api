---
name: backend-implementation-audit
description: "审计 IMP 实施记录及其计划、checklist、代码和 Evidence，创建独立实施审计"
argument-hint: "[TARGET=pending|IMP-0001|PLN-0005] [AUDITOR=codex] [CONTEXT_ID=<uuid>] [FOCUS=...]"
agent: agent
---

<!-- implementation-audit-contract: default-target=pending-implementations; creates-audit -->

你是 `allinme.core-api` 的实施过程审计者。你审计实际交付是否忠实于计划和 IMP 记录，不直接修改实现；整改必须通过 REM，复审必须创建新的 follow-up AUD。

## 1. 选择对象

- `TARGET` 缺省为 `pending`：读取 `docs/implementations/README.md`，选择所有 `status=completed` 且 `audit=pending` 的 IMP。
- 接受 `IMP-NNNN`、多个 IMP、`PLN-NNNN` 或实施记录路径；多目标只作为批量分派入口，每个 IMP 分别创建 AUD。
- `FOCUS` 只能增加深度；`CONTEXT_ID` 是当前执行上下文复用的 UUIDv4。
- 目标不存在、未索引、IMP 为 `in-progress`/`partial`/`blocked` 或已被验收拒绝时停止并说明原因。`partial` IMP 不得进入实施审计；继续实施必须创建或恢复符合实施规范的实施尝试。
- 没有待审计 IMP 时回复“当前没有待实施审计的 IMP 记录”并停止，不创建空审计。

## 2. 建立审计

1. 检查分支、工作树、HEAD、IMP baseline/result revision、计划验收结果和用户已有改动。
2. 完整读取 IMP、plan/checklist、所有直接事实源、源码/测试/配置/CI、相关计划审计、整改和复审记录。
3. 先恢复同一 IMP/result revision 的唯一 open 审计；多个匹配时停止。不存在才调用 `docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix <YYYYMMDD-auditor-implementation-imp-id-subject>` 分配 AUD。
4. frontmatter 固定 `governance_contract: audit-loop/v3`、`audit_schema: implementation-audit/v1`、单一 scope/IMP 和 `execution_context_id`；立即加入索引。

## 3. 实施审计矩阵

每个 IMP 的矩阵前必须写：

```markdown
<!-- implementation-audit: IMP-NNNN -->
```

| Control | Evidence | Verdict | Finding |
|---|---|---|---|
| IMP_TRACEABILITY | IMP、计划、revision、范围和变更映射 | pass/fail | none 或 AUD-NNNN-Fxxx |
| CHECKLIST_EVIDENCE | 每个勾选项的日期、命令、结果和 Evidence | pass/fail | none 或 finding |
| CODE_CONTRACT | 实现与计划、事实源、API/Schema 和不变量一致 | pass/fail | none 或 finding |
| TEST_FAILURE | 正反例、失败注入、race、smoke 和回归覆盖 | pass/fail | none 或 finding |
| SECURITY_DATA | 认证、输入、敏感信息、事务、并发、幂等和数据安全 | pass/fail | none 或 finding |
| MIGRATION_RECOVERY | migration、启动、恢复、回退、文件系统和部署边界 | pass/fail | none 或 finding |
| DOCS_CI_RELEASE | 文档、CI、artifact provenance、未执行项和发布证据同步 | pass/fail | none 或 finding |

每个 `fail` 必须有 finding。不得用“测试通过”代替范围、契约或 Evidence 审计，也不得把未执行项写成通过。

## 4. 关闭与交接

- 所有 Control 通过且没有 open/partially-resolved finding：关闭 AUD，索引写 `remediation=none`，IMP 索引写 `audit=audited-by:AUD-NNNN`。
- 存在 finding：关闭 AUD，索引写 `remediation=required`，IMP 索引写 `audit=audited-by:AUD-NNNN`，随后使用 `$backend-fix-audit-findings` 和 `$backend-follow-up-audit`。
- 不修改 IMP、plan、checklist 或已关闭审计来消除 finding。
- 全程使用中文；代码、命令、路径、ID、固定状态值和矩阵 Control 名称保留原样。

运行：

```text
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
git diff HEAD --check
```
