# 文档工具

- `validate.ps1`：检查 Markdown frontmatter、相对链接、`PLN` / `IMP` / `AUD` / `REM` 命名与元数据、对象引用图、plan/checklist 配对、revision-bound 计划审计、实施/审计/整改索引、验收矩阵与 verdict 状态机、subject-specific 独立验证、状态跳转引用、真实 subject revision、最新 IMP、计划漂移、IMP/REM effective revision 链、source 终态快照、审计工作流契约和跨提交只追加历史，并运行 `git diff HEAD --check`。
- `validate-audit-workflows.ps1`：只读检查两个闭环及其原子 prompt/skill 是否保留真实 runtime task 隔离、open/terminal governance commit、clean `governance_revision`、足够 cycle budget、single-transition child 路由和 `TARGET`/`ADVANCE_SET`/`PEER_SET` 分离。
- `validate.tests.ps1`：用独立 fixture 验证合法计划、实施、审计、整改和验收结构通过，并拒绝未绑定 revision 的计划审计、治理命令冒充 subject 验证、partial REM 重复排队、倒置时间、多计划单 verdict、旧 IMP 验收、伪造状态跳转、漏列 effective REM、脏验收基线、失效对象引用和非法治理结构。
- `reserve-governance-record.ps1`：在跨进程互斥区内分配 `AUD` / `REM` / `IMP` 编号，并通过原子 `CreateNew` 预留目标文件，避免并发执行复用同一 ID。

从仓库根目录运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate-audit-workflows.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1 -HistoryBase <merge-base-full-sha>
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix 20260714-codex-plan-example
```

CI 必须使用完整 Git 历史，并通过 `AUDIT_HISTORY_BASE` 或 `-HistoryBase` 传入 PR merge-base/push 前一 revision；否则只能检查当前工作树，不能证明终态 AUD/REM/IMP 在多提交分支中未被改写。
