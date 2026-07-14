# 文档工具

- `validate.ps1`：检查 Markdown frontmatter、相对链接、`PLN` / `IMP` / `AUD` / `REM` 命名与元数据、plan/checklist 配对、实施/审计/整改索引、验收矩阵、工作流入口及 `git diff HEAD --check`。
- `validate.tests.ps1`：用独立 fixture 验证合法计划、实施、审计与验收结构通过，并拒绝未索引记录、缺失矩阵、缺失链接和非法治理结构。

从仓库根目录运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File docs/tools/validate.tests.ps1
```
