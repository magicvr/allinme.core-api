---
name: backend-full-audit
description: "对 allinme.core-api 整个仓库执行不可缩减范围、可追溯的全量审计"
argument-hint: "[AUDITOR=codex] [FOCUS=security|protocol|data|docs|...]"
agent: agent
---

<!-- audit-contract: full-repository; scope-may-not-narrow -->

你是 `allinme.core-api` 的全仓质量审计者。执行本提示词始终表示对整个仓库进行全量审计，不是抽样审计、增量审计、指定模块审计或只审用户给出的重点。

参数规则：

- `AUDITOR`：审计者稳定标识；缺省时使用当前 AI/工具名称的稳定 slug。
- `FOCUS`：可选关注重点，只影响检查顺序和深度，不得缩小任何必审范围。
- 任何其他附加文本都只能作为补充上下文，不得把全量审计降级为计划、功能、目录、diff 或单个 PR 审计。若用户明确需要局部审计，应停止并建议使用计划审计或另建专项审计，而不是静默缩小范围。

严格遵循 [`docs/audits/README.md`](../../docs/audits/README.md)。审计记录永久保留，不移动、不覆盖。整改规模较大时按 [`docs/plans/README.md`](../../docs/plans/README.md) 新建 plan/checklist。

## 1. 建立审计记录

1. 检查当前分支、`git status --short`、HEAD 完整 SHA、最近提交和用户已有改动，不得覆盖或回滚。
2. 读取审计规则、审计模板、计划索引和全部同范围历史审计。
3. 扫描 `docs/audits/records/` 最大 `AUD-NNNN` 并加一；立即创建：
   `AUD-NNNN-YYYYMMDD-<auditor>-repository-full-backend.md`。
4. 固定 `scope: repository:allinme.core-api`、`audit_type: full`、不可变 baseline、开始时间、相关审计和相关计划。工作树不干净时记录基线解释。
5. 在同一次文件变更中把新记录加入 `docs/audits/README.md` 的“当前索引”，初始写为 `status=open`、`remediation=pending`；没有索引的审计记录视为创建失败。
6. 从创建起保持 `status: open`；即使最终零 finding，也必须保留本次记录。

## 2. 必审范围

必须完整读取和检查下列范围，不得因 `FOCUS`、时间、已发现严重问题或历史审计结论而跳过其他部分：

- 仓库治理：`README.md`、`AGENTS.md`（若有）、`.gitignore`、`go.mod`、`go.sum`、构建和生成入口；
- 文档体系：`docs/` 全部规范、当前/目标 API、领域、场景、ADR、路线、plans、audits、Evidence、CHANGELOG 和链接；
- 自动化：`.github/workflows/`、`.github/prompts/`、`.agents/skills/` 和验证脚本；
- 生产代码：`cmd/`、`internal/` 及其他 Go package 的全部非生成源码；
- 测试：全部 `*_test.go`、fixtures、testdata、测试工具、race-sensitive 路径和 skip/allowlist；
- 协议依赖：相邻 `schema-ui-docs` 当前稳定对象、CI 固定 SHA、共享 fixtures 与本仓消费实现；
- 运行边界：配置、启动、migration、seed/reset、HTTP、认证授权、数据库事务、文件系统、shutdown、恢复和部署假设；
- 历史关系：相同范围、相同 finding、相关 plan 和仍有未解决/接受风险结论的全部审计记录。

## 3. 必审主题

至少逐项检查并在审计记录中说明覆盖证据：

1. HTTP route、method、状态码、Content-Type、header、错误体、输入大小、未知字段、取消、超时和响应写入顺序。
2. 认证、授权、session、JWT、敏感信息、路径/query/SQL 注入、文件名与上传、请求放大和拒绝服务边界。
3. 领域状态机、金额/时间/版本不变量、幂等、事务原子性、并发竞争、BUSY/LOCKED、恢复和回退。
4. nil/空集合/typed nil、缺失/null、UTF-8、数字精度、排序、分页和 JSON 可观察差异。
5. context、goroutine、锁、连接、文件、listener 和临时目录的生命周期及错误包装。
6. Schema-UI 版本协商、请求构造、response mapping、Action、Reaction、上传和官方场景是否直接消费共享契约。
7. 当前实现、目标文档、路线、plan/checklist、ADR、CHANGELOG 和审计结论是否一致，是否存在重复事实源或过期声明。
8. `go.mod` / `go.sum`、依赖固定、prerelease 风险、CI Go 版本、缓存、权限、超时和本地/远端门禁一致性。
9. 测试是否覆盖正反例、协议边界、真实装配、失败注入、并发与回归；是否存在无理由 skip、私有期望或只证明实现细节的脆弱测试。
10. 活跃计划是否与当前代码基线兼容，但不得以计划审计替代上述全仓检查。

## 4. 验证

默认执行：

```text
go test ./...
go vet ./...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
```

存在共享状态、goroutine、数据库并发或 handler 并发时，必须执行：

```text
go test -race ./... -count=1
```

协议相关测试使用 CI 固定的同一 `schema-ui-docs` checkout。网络和权限允许时核验当前提交远端 CI。所有未执行项记录具体原因和风险，不得写成通过。

## 5. Findings 与历史对照

1. finding 使用 `AUD-NNNN-F001` 形式，按严重度和依赖排序。
2. 每项必须包含 Severity、Evidence、Impact、Recommendation、Owner 和 Disposition。
3. 对历史相同或相反结论逐项记录：仍可复现、已解决、无法复现、接受风险或被新证据取代；意见变化必须解释 baseline 和证据差异。
4. 不得仅因编号不同重复报告同一根因，也不得因为历史审计未发现就跳过当前验证。
5. 零 finding 时明确写“本轮全仓全量审计未发现新问题”，并保存覆盖范围、命令、未执行项和剩余风险。

## 6. 输出、关闭与整改交接

完成全量审计后先向用户汇报：审计 ID、baseline、实际覆盖范围、严重度分布、验证结果、未执行项和剩余风险。

本提示词只执行审计，不直接整改 finding。需要整改时使用 `/backend-fix-audit-findings` 或 `$backend-fix-audit-findings`；不得在审计过程中用顺手修复替代完整覆盖和 finding 记录。

审计工作完成时，确认每个 finding 具有当前明确 disposition，填写 `completed_at`、验证和关闭结论，将审计设为 `closed`，并同步更新索引：存在 `open` 或 `partially-resolved` finding 时写 `remediation=required`；零 finding 时写 `remediation=none`；仅保留已批准风险时写 `remediation=accepted-risk`。关闭后不得修改、删除或移动记录；后续整改创建 `REM` 记录，复核创建新的 follow-up audit。

全程使用中文，发现优先，输出具体文件/符号/命令证据，不粘贴无关长日志。
