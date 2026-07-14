# 文档工具

- `validate.ps1`：检查 Markdown frontmatter、相对链接、`PLN` / `IMP` / `AUD` / `REM` 命名与元数据、对象引用图、plan/checklist 配对、实施/审计/整改索引、验收矩阵与 verdict 状态机、验收独立性/evidence revision、工作流目标透传及 `git diff HEAD --check`。
- `validate.tests.ps1`：用独立 fixture 验证合法计划、实施、审计与验收结构通过，并拒绝未索引记录、缺失矩阵、矛盾验收 verdict、脏验收基线、失效对象引用、缺失链接和非法治理结构。
- `reserve-governance-record.ps1`：在跨进程互斥区内分配 `AUD` / `REM` / `IMP` 编号，并通过原子 `CreateNew` 预留目标文件，避免并发执行复用同一 ID。

从仓库根目录运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/reserve-governance-record.ps1 -Kind AUD -Suffix 20260714-codex-plan-example
```
